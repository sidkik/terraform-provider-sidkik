package sidkik

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/transport"
	"google.golang.org/grpc"
)

type providerMeta struct {
	ModuleName string `cty:"module_name"`
}

type Formatter struct {
	TimestampFormat string
	LogFormat       string
}

// Borrowed logic from https://github.com/sirupsen/logrus/blob/master/json_formatter.go and https://github.com/t-tomalak/logrus-easy-formatter/blob/master/formatter.go
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Suppress logs if TF_LOG is not DEBUG or TRACE
	// also suppress frequent transport spam
	if !logging.IsDebugOrHigher() || strings.Contains(entry.Message, "transport is closing") {
		return nil, nil
	}
	output := f.LogFormat
	entry.Level = logrus.DebugLevel // Force Entries to be Debug

	timestampFormat := f.TimestampFormat

	output = strings.Replace(output, "%time%", entry.Time.Format(timestampFormat), 1)

	output = strings.Replace(output, "%msg%", entry.Message, 1)

	level := strings.ToUpper(entry.Level.String())
	output = strings.Replace(output, "%lvl%", level, 1)

	var gRPCMessageFlag bool
	for k, val := range entry.Data {
		switch v := val.(type) {
		case string:
			output = strings.Replace(output, "%"+k+"%", v, 1)
		case int:
			s := strconv.Itoa(v)
			output = strings.Replace(output, "%"+k+"%", s, 1)
		case bool:
			s := strconv.FormatBool(v)
			output = strings.Replace(output, "%"+k+"%", s, 1)
		}

		if k != "system" {
			gRPCMessageFlag = true
		}
	}

	if gRPCMessageFlag {
		data := make(logrus.Fields, len(entry.Data)+4)
		for k, v := range entry.Data {
			switch v := v.(type) {
			case error:
				// Otherwise errors are ignored by `encoding/json`
				// https://github.com/sirupsen/logrus/issues/137
				data[k] = v.Error()
			default:
				data[k] = v
			}
		}

		var b *bytes.Buffer
		if entry.Buffer != nil {
			b = entry.Buffer
		} else {
			b = &bytes.Buffer{}
		}

		encoder := json.NewEncoder(b)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
		}

		finalOutput := append([]byte(output), b.Bytes()...)
		return finalOutput, nil
	}

	return []byte(output), nil
}

// Config is the configuration structure used to instantiate the Google
// provider.
type Config struct {
	AccessToken                        string
	Credentials                        string
	ImpersonateServiceAccount          string
	ImpersonateServiceAccountDelegates []string
	Project                            string
	Region                             string
	BillingProject                     string
	Zone                               string
	Scopes                             []string
	BatchingConfig                     *batchingConfig
	UserProjectOverride                bool
	RequestReason                      string
	RequestTimeout                     time.Duration
	// PollInterval is passed to resource.StateChangeConf in common_operation.go
	// It controls the interval at which we poll for successful operations
	PollInterval time.Duration

	client             *http.Client
	context            context.Context
	userAgent          string
	gRPCLoggingOptions []option.ClientOption

	tokenSource oauth2.TokenSource

	FirebaseRulesBasePath string

	ComputeBasePath string

	requestBatcherServiceUsage *RequestBatcher
	requestBatcherIam          *RequestBatcher
}

const FirebaseRulesBasePathKey = "FirebaseRules"

// Generated product base paths
var DefaultBasePaths = map[string]string{
	FirebaseRulesBasePathKey: "https://firebaserules.googleapis.com/v1/",
}

var DefaultClientScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
}

func (c *Config) LoadAndValidate(ctx context.Context) error {
	if len(c.Scopes) == 0 {
		c.Scopes = DefaultClientScopes
	}

	c.context = ctx

	tokenSource, err := c.getTokenSource(c.Scopes, false)
	if err != nil {
		return err
	}

	c.tokenSource = tokenSource

	cleanCtx := context.WithValue(ctx, oauth2.HTTPClient, cleanhttp.DefaultClient())

	// 1. MTLS TRANSPORT/CLIENT - sets up proper auth headers
	client, _, err := transport.NewHTTPClient(cleanCtx, option.WithTokenSource(tokenSource))
	if err != nil {
		return err
	}

	// Userinfo is fetched before request logging is enabled to reduce additional noise.
	err = c.logGoogleIdentities()
	if err != nil {
		return err
	}

	// 2. Logging Transport - ensure we log HTTP requests to GCP APIs.
	loggingTransport := logging.NewTransport("Google", client.Transport)

	// 3. Retry Transport - retries common temporary errors
	// Keep order for wrapping logging so we log each retried request as well.
	// This value should be used if needed to create shallow copies with additional retry predicates.
	// See ClientWithAdditionalRetries
	retryTransport := NewTransportWithDefaultRetries(loggingTransport)

	// 4. Header Transport - outer wrapper to inject additional headers we want to apply
	// before making requests
	headerTransport := newTransportWithHeaders(retryTransport)
	if c.RequestReason != "" {
		headerTransport.Set("X-Goog-Request-Reason", c.RequestReason)
	}

	// Ensure $userProject is set for all HTTP requests using the client if specified by the provider config
	// See https://cloud.google.com/apis/docs/system-parameters
	if c.UserProjectOverride && c.BillingProject != "" {
		headerTransport.Set("X-Goog-User-Project", c.BillingProject)
	}

	// Set final transport value.
	client.Transport = headerTransport

	// This timeout is a timeout per HTTP request, not per logical operation.
	client.Timeout = c.synchronousTimeout()

	c.client = client
	c.context = ctx
	c.Region = GetRegionFromRegionSelfLink(c.Region)
	c.requestBatcherServiceUsage = NewRequestBatcher("Service Usage", ctx, c.BatchingConfig)
	c.requestBatcherIam = NewRequestBatcher("IAM", ctx, c.BatchingConfig)
	c.PollInterval = 10 * time.Second

	// gRPC Logging setup
	logger := logrus.StandardLogger()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&Formatter{
		TimestampFormat: "2006/01/02 15:04:05",
		LogFormat:       "%time% [%lvl%] %msg% \n",
	})

	alwaysLoggingDeciderClient := func(ctx context.Context, fullMethodName string) bool { return true }
	grpc_logrus.ReplaceGrpcLogger(logrus.NewEntry(logger))

	c.gRPCLoggingOptions = append(
		c.gRPCLoggingOptions, option.WithGRPCDialOption(grpc.WithUnaryInterceptor(
			grpc_logrus.PayloadUnaryClientInterceptor(logrus.NewEntry(logger), alwaysLoggingDeciderClient))),
		option.WithGRPCDialOption(grpc.WithStreamInterceptor(
			grpc_logrus.PayloadStreamClientInterceptor(logrus.NewEntry(logger), alwaysLoggingDeciderClient))),
	)

	return nil
}

func expandProviderBatchingConfig(v interface{}) (*batchingConfig, error) {
	config := &batchingConfig{
		sendAfter:      time.Second * defaultBatchSendIntervalSec,
		enableBatching: true,
	}

	if v == nil {
		return config, nil
	}
	ls := v.([]interface{})
	if len(ls) == 0 || ls[0] == nil {
		return config, nil
	}

	cfgV := ls[0].(map[string]interface{})
	if sendAfterV, ok := cfgV["send_after"]; ok {
		sendAfter, err := time.ParseDuration(sendAfterV.(string))
		if err != nil {
			return nil, fmt.Errorf("unable to parse duration from 'send_after' value %q", sendAfterV)
		}
		config.sendAfter = sendAfter
	}

	if enable, ok := cfgV["enable_batching"]; ok {
		config.enableBatching = enable.(bool)
	}

	return config, nil
}

func (c *Config) synchronousTimeout() time.Duration {
	if c.RequestTimeout == 0 {
		return 120 * time.Second
	}
	return c.RequestTimeout
}

// Print Identities executing terraform API Calls.
func (c *Config) logGoogleIdentities() error {
	if c.ImpersonateServiceAccount == "" {

		tokenSource, err := c.getTokenSource(c.Scopes, true)
		if err != nil {
			return err
		}
		c.client = oauth2.NewClient(c.context, tokenSource) // c.client isn't initialised fully when this code is called.

		email, err := GetCurrentUserEmail(c, c.userAgent)
		if err != nil {
			log.Printf("[INFO] error retrieving userinfo for your provider credentials. have you enabled the 'https://www.googleapis.com/auth/userinfo.email' scope? error: %s", err)
		}

		log.Printf("[INFO] Terraform is using this identity: %s", email)

		return nil

	}

	// Drop Impersonated ClientOption from OAuth2 TokenSource to infer original identity

	tokenSource, err := c.getTokenSource(c.Scopes, true)
	if err != nil {
		return err
	}
	c.client = oauth2.NewClient(c.context, tokenSource) // c.client isn't initialised fully when this code is called.

	email, err := GetCurrentUserEmail(c, c.userAgent)
	if err != nil {
		log.Printf("[INFO] error retrieving userinfo for your provider credentials. have you enabled the 'https://www.googleapis.com/auth/userinfo.email' scope? error: %s", err)
	}

	log.Printf("[INFO] Terraform is configured with service account impersonation, original identity: %s, impersonated identity: %s", email, c.ImpersonateServiceAccount)

	// Add the Impersonated ClientOption back in to the OAuth2 TokenSource

	tokenSource, err = c.getTokenSource(c.Scopes, false)
	if err != nil {
		return err
	}
	c.client = oauth2.NewClient(c.context, tokenSource) // c.client isn't initialised fully when this code is called.

	return nil
}

// Get a TokenSource based on the Google Credentials configured.
// If initialCredentialsOnly is true, don't follow the impersonation settings and return the initial set of creds.
func (c *Config) getTokenSource(clientScopes []string, initialCredentialsOnly bool) (oauth2.TokenSource, error) {
	creds, err := c.GetCredentials(clientScopes, initialCredentialsOnly)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}
	return creds.TokenSource, nil
}

// Methods to create new services from config
// Some base paths below need the version and possibly more of the path
// set on them. The client libraries are inconsistent about which values they need;
// while most only want the host URL, some older ones also want the version and some
// of those "projects" as well. You can find out if this is required by looking at
// the basePath value in the client library file.
func (c *Config) NewComputeClient(userAgent string) *compute.Service {
	log.Printf("[INFO] Instantiating GCE client for path %s", c.ComputeBasePath)
	clientCompute, err := compute.NewService(c.context, option.WithHTTPClient(c.client))
	if err != nil {
		log.Printf("[WARN] Error creating client compute: %s", err)
		return nil
	}
	clientCompute.UserAgent = userAgent
	clientCompute.BasePath = c.ComputeBasePath

	return clientCompute
}

// staticTokenSource is used to be able to identify static token sources without reflection.
type staticTokenSource struct {
	oauth2.TokenSource
}

// Get a set of credentials with a given scope (clientScopes) based on the Config object.
// If initialCredentialsOnly is true, don't follow the impersonation settings and return the initial set of creds
// instead.
func (c *Config) GetCredentials(clientScopes []string, initialCredentialsOnly bool) (googleoauth.Credentials, error) {
	if c.AccessToken != "" {
		contents, _, err := pathOrContents(c.AccessToken)
		if err != nil {
			return googleoauth.Credentials{}, fmt.Errorf("Error loading access token: %s", err)
		}

		token := &oauth2.Token{AccessToken: contents}
		if c.ImpersonateServiceAccount != "" && !initialCredentialsOnly {
			opts := []option.ClientOption{option.WithTokenSource(oauth2.StaticTokenSource(token)), option.ImpersonateCredentials(c.ImpersonateServiceAccount, c.ImpersonateServiceAccountDelegates...), option.WithScopes(clientScopes...)}
			creds, err := transport.Creds(context.TODO(), opts...)
			if err != nil {
				return googleoauth.Credentials{}, err
			}
			return *creds, nil
		}

		log.Printf("[INFO] Authenticating using configured Google JSON 'access_token'...")
		log.Printf("[INFO]   -- Scopes: %s", clientScopes)
		return googleoauth.Credentials{
			TokenSource: staticTokenSource{oauth2.StaticTokenSource(token)},
		}, nil
	}

	if c.Credentials != "" {
		contents, _, err := pathOrContents(c.Credentials)
		if err != nil {
			return googleoauth.Credentials{}, fmt.Errorf("error loading credentials: %s", err)
		}

		if c.ImpersonateServiceAccount != "" && !initialCredentialsOnly {
			opts := []option.ClientOption{option.WithCredentialsJSON([]byte(contents)), option.ImpersonateCredentials(c.ImpersonateServiceAccount, c.ImpersonateServiceAccountDelegates...), option.WithScopes(clientScopes...)}
			creds, err := transport.Creds(context.TODO(), opts...)
			if err != nil {
				return googleoauth.Credentials{}, err
			}
			return *creds, nil
		}

		creds, err := googleoauth.CredentialsFromJSON(c.context, []byte(contents), clientScopes...)
		if err != nil {
			return googleoauth.Credentials{}, fmt.Errorf("unable to parse credentials from '%s': %s", contents, err)
		}

		log.Printf("[INFO] Authenticating using configured Google JSON 'credentials'...")
		log.Printf("[INFO]   -- Scopes: %s", clientScopes)
		return *creds, nil
	}

	if c.ImpersonateServiceAccount != "" && !initialCredentialsOnly {
		opts := option.ImpersonateCredentials(c.ImpersonateServiceAccount, c.ImpersonateServiceAccountDelegates...)
		creds, err := transport.Creds(context.TODO(), opts, option.WithScopes(clientScopes...))
		if err != nil {
			return googleoauth.Credentials{}, err
		}

		return *creds, nil
	}

	log.Printf("[INFO] Authenticating using DefaultClient...")
	log.Printf("[INFO]   -- Scopes: %s", clientScopes)
	defaultTS, err := googleoauth.DefaultTokenSource(context.Background(), clientScopes...)
	if err != nil {
		return googleoauth.Credentials{}, fmt.Errorf("Attempted to load application default credentials since neither `credentials` nor `access_token` was set in the provider block.  No credentials loaded. To use your gcloud credentials, run 'gcloud auth application-default login'.  Original error: %w", err)
	}

	return googleoauth.Credentials{
		TokenSource: defaultTS,
	}, err
}

// Remove the `/{{version}}/` from a base path if present.
func removeBasePathVersion(url string) string {
	re := regexp.MustCompile(`(?P<base>http[s]://.*)(?P<version>/[^/]+?/$)`)
	return re.ReplaceAllString(url, "$1/")
}

// For a consumer of config.go that isn't a full fledged provider and doesn't
// have its own endpoint mechanism such as sweepers, init {{service}}BasePath
// values to a default. After using this, you should call config.LoadAndValidate.
func ConfigureBasePaths(c *Config) {
	c.FirebaseRulesBasePath = DefaultBasePaths[FirebaseRulesBasePathKey]
}
