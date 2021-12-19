package sidkik

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirebaseAuthConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirebaseAuthConfigCreate,
		Read:   resourceFirebaseAuthConfigRead,
		Update: resourceFirebaseAuthConfigUpdate,
		Delete: resourceFirebaseAuthConfigDelete,

		// Importer: &schema.ResourceImporter{
		// 	State: resourceComputeHealthCheckImport,
		// },

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(4 * time.Minute),
			Update: schema.DefaultTimeout(4 * time.Minute),
			Delete: schema.DefaultTimeout(4 * time.Minute),
		},

		// CustomizeDiff: rulesCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"email": {
				Type: schema.TypeBool,
				// Default:     false,
				Optional:    true,
				Computed:    true,
				ForceNew:    false,
				Description: `enable email signin`,
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    false,
				Description: `id of the config`,
			},
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    false,
				Description: `id of the config`,
			},
			"authorized_domains": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: `list of authorized domains for authentication`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"project": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
		},
		UseJSONNumber: true,
	}
}

func resourceFirebaseAuthConfigRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	url, err := replaceVars(d, config, "{{IdentityPlatformBasePath}}projects/{{project}}/config")
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for AuthConfig: %s", err)
	}

	// grab the existing config - there are rules by default when the firebase account is created
	res, err := sendRequest(config, "GET", project, url, userAgent, nil)

	fmt.Println("Config ___________________", res, err)

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("AuthConfig %q", d.Id()))
	}

	if err := d.Set("project", project); err != nil {
		return fmt.Errorf("Error reading AuthConfig: %s", err)
	}

	if err := d.Set("name", flattenAuthConfigName(res["name"], d, config)); err != nil {
		return fmt.Errorf("Error reading AuthConfig: %s", err)
	}

	if err := d.Set("email", flattenEmail(res["signIn"], d, config)); err != nil {
		return fmt.Errorf("Error reading AuthConfig: %s", err)
	}

	if err := d.Set("authorized_domains", flattenAuthorizedDomains(res["authorizedDomains"], d, config)); err != nil {
		return fmt.Errorf("Error reading AuthConfig: %s", err)
	}

	// Set the ID now
	d.SetId(flattenAuthConfigName(res["name"], d, config).(string))

	log.Println("[DEBUG] Read Firebase AuthConfig", d.Get("name"))

	return nil
}

func resourceFirebaseAuthConfigDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Not deleting rule - new rule had been made the latest release")
	return nil
}

func resourceFirebaseAuthConfigUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	url, err := replaceVars(d, config, "{{IdentityPlatformBasePath}}projects/{{project}}/config?updateMask=signIn.email,authorizedDomains")
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for AuthConfig: %s", err)
	}
	configObj := make(map[string]interface{})
	configObj["email"] = d.Get("email")
	configObj["authorized_domains"] = d.Get("authorized_domains")
	configObj, err = resourceFirebaseAuthConfigPatchEncoder(d, meta, configObj)

	// grab the existing config - there are rules by default when the firebase account is created
	_, err = sendRequest(config, "PATCH", project, url, userAgent, configObj)

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("AuthConfig %q", d.Id()))
	}

	log.Printf("[INFO] Updated auth config")
	return resourceFirebaseAuthConfigRead(d, meta)
}

func resourceFirebaseAuthConfigCreate(d *schema.ResourceData, meta interface{}) error {
	// will always be an update
	return resourceFirebaseAuthConfigUpdate(d, meta)
}

func flattenAuthConfigName(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenEmail(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	signIn := v.(map[string]interface{})
	if signIn["email"] == nil {
		return false
	}
	email := signIn["email"].(map[string]interface{})

	return email["enabled"]
}

func flattenAuthorizedDomains(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func resourceFirebaseAuthConfigPatchEncoder(d *schema.ResourceData, meta interface{}, obj map[string]interface{}) (map[string]interface{}, error) {
	emailProviderConfig := make(map[string]interface{})
	emailProviderConfig["enabled"] = obj["email"]
	emailProviderConfig["passwordRequired"] = obj["email"]
	email := make(map[string]interface{})
	email["email"] = emailProviderConfig
	wrapper := make(map[string]interface{})
	wrapper["signIn"] = email
	wrapper["authorizedDomains"] = obj["authorized_domains"]

	return wrapper, nil
}
