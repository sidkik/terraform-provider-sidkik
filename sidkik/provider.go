package sidkik

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	googleoauth "golang.org/x/oauth2/google"
)

// Global MutexKV
var mutexKV = NewMutexKV()

const TestEnvVar = "TF_ACC"

// Provider -
func Provider() *schema.Provider {

	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"credentials": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validateCredentials,
				ConflictsWith: []string{"access_token"},
			},

			"access_token": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"credentials"},
			},

			"impersonate_service_account": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_IMPERSONATE_SERVICE_ACCOUNT",
				}, nil),
			},

			"impersonate_service_account_delegates": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_PROJECT",
					"GOOGLE_CLOUD_PROJECT",
					"GCLOUD_PROJECT",
					"CLOUDSDK_CORE_PROJECT",
				}, nil),
			},

			"billing_project": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_BILLING_PROJECT",
				}, nil),
			},

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_REGION",
					"GCLOUD_REGION",
					"CLOUDSDK_COMPUTE_REGION",
				}, nil),
			},

			"zone": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_ZONE",
					"GCLOUD_ZONE",
					"CLOUDSDK_COMPUTE_ZONE",
				}, nil),
			},

			"scopes": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user_project_override": {
				Type:     schema.TypeBool,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"USER_PROJECT_OVERRIDE",
				}, nil),
			},

			"request_timeout": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"request_reason": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"CLOUDSDK_CORE_REQUEST_REASON",
				}, nil),
			},

			"firebase_rules_custom_endpoint": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCustomEndpoint,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"SIDKIK_FIREBASE_RULES_CUSTOM_ENDPOINT",
				}, DefaultBasePaths[FirebaseRulesBasePathKey]),
			},
		},
		ProviderMetaSchema: map[string]*schema.Schema{
			"module_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"sidkik_firebase_firestore_rule": dataSourceFirebaseFirestoreRule(),
			"sidkik_firebase_storage_rule":   dataSourceFirebaseStorageRule(),
		},
		ResourcesMap: resourceMap(),
	}
	provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, provider)
	}

	return provider
}

func resourceMap() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"sidkik_firebase_firestore_rule": resourceFirebaseFirestoreRule(),
		"sidkik_firebase_storage_rule":   resourceFirebaseStorageRule(),
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, p *schema.Provider) (interface{}, diag.Diagnostics) {
	config := Config{
		Project:             d.Get("project").(string),
		Region:              d.Get("region").(string),
		Zone:                d.Get("zone").(string),
		UserProjectOverride: d.Get("user_project_override").(bool),
		BillingProject:      d.Get("billing_project").(string),
		userAgent:           p.UserAgent("terraform-sidkik-firebase", "0.1.0"), //version.ProviderVersion),
	}

	// opt in extension for adding to the User-Agent header
	if ext := os.Getenv("GOOGLE_TERRAFORM_USERAGENT_EXTENSION"); ext != "" {
		ua := config.userAgent
		config.userAgent = fmt.Sprintf("%s %s", ua, ext)
	}

	if v, ok := d.GetOk("request_timeout"); ok {
		var err error
		config.RequestTimeout, err = time.ParseDuration(v.(string))
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}

	if v, ok := d.GetOk("request_reason"); ok {
		config.RequestReason = v.(string)
	}

	// Check for primary credentials in config. Note that if neither is set, ADCs
	// will be used if available.
	if v, ok := d.GetOk("access_token"); ok {
		config.AccessToken = v.(string)
	}

	if v, ok := d.GetOk("credentials"); ok {
		config.Credentials = v.(string)
	}

	// only check environment variables if neither value was set in config- this
	// means config beats env var in all cases.
	if config.AccessToken == "" && config.Credentials == "" {
		config.Credentials = multiEnvSearch([]string{
			"GOOGLE_CREDENTIALS",
			"GOOGLE_CLOUD_KEYFILE_JSON",
			"GCLOUD_KEYFILE_JSON",
		})

		config.AccessToken = multiEnvSearch([]string{
			"GOOGLE_OAUTH_ACCESS_TOKEN",
		})
	}

	// Given that impersonate_service_account is a secondary auth method, it has
	// no conflicts to worry about. We pull the env var in a DefaultFunc.
	if v, ok := d.GetOk("impersonate_service_account"); ok {
		config.ImpersonateServiceAccount = v.(string)
	}

	delegates := d.Get("impersonate_service_account_delegates").([]interface{})
	if len(delegates) > 0 {
		config.ImpersonateServiceAccountDelegates = make([]string, len(delegates))
	}
	for i, delegate := range delegates {
		config.ImpersonateServiceAccountDelegates[i] = delegate.(string)
	}

	scopes := d.Get("scopes").([]interface{})
	if len(scopes) > 0 {
		config.Scopes = make([]string, len(scopes))
	}
	for i, scope := range scopes {
		config.Scopes[i] = scope.(string)
	}

	// Generated products
	config.FirebaseRulesBasePath = d.Get("firebase_rules_custom_endpoint").(string)

	stopCtx, ok := schema.StopContext(ctx)
	if !ok {
		stopCtx = ctx
	}
	if err := config.LoadAndValidate(stopCtx); err != nil {
		return nil, diag.FromErr(err)
	}

	return &config, nil
}

func validateCredentials(v interface{}, k string) (warnings []string, errors []error) {
	if v == nil || v.(string) == "" {
		return
	}
	creds := v.(string)
	// if this is a path and we can stat it, assume it's ok
	if _, err := os.Stat(creds); err == nil {
		return
	}
	if _, err := googleoauth.CredentialsFromJSON(context.Background(), []byte(creds)); err != nil {
		errors = append(errors,
			fmt.Errorf("JSON credentials in %q are not valid: %s", creds, err))
	}

	return
}

func validateCustomEndpoint(v interface{}, k string) (ws []string, errors []error) {
	re := `.*/[^/]+/$`
	return validateRegexp(re)(v, k)
}
