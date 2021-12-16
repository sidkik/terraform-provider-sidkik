package sidkik

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirebaseFirestoreRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirebaseFirestoreRuleCreate,
		Read:   resourceFirebaseFirestoreRuleRead,
		// Update: resourceFirebaseFirestoreRuleCreate,
		Delete: resourceFirebaseFirestoreRuleDelete,

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
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: `name of the firestore rule`,
			},
			"rule": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: `Source for the firestore rule. Provided as a string with the correct rules schems`,
			},
			"project": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
		UseJSONNumber: true,
	}
}

func resourceFirebaseFirestoreRuleRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}
	urlReleases, err := replaceVars(d, config, "{{FirebaseRulesBasePath}}projects/{{project}}/releases")
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for FirebaseRules: %s", err)
	}

	// grab the existing rules - there are rules by default when the firebase account is created
	res, err := sendRequest(config, "GET", project, urlReleases, userAgent, nil)

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("FirebaseRules %q", d.Id()))
	}

	if err := d.Set("project", project); err != nil {
		return fmt.Errorf("Error reading FirebaseRules: %s", err)
	}

	ruleSets := res["releases"].([]interface{})

	if err := d.Set("name", flattenFirebaseRuleName(ruleSets, "cloud.firestore", d, config)); err != nil {
		return fmt.Errorf("Error reading FirebaseRules: %s", err)
	}

	// Set the ID now
	d.SetId(d.Get("name").(string))

	// now grab the actual rule text

	// firestore rule

	urlFirestoreRule, err := replaceVars(d, config, "{{FirebaseRulesBasePath}}"+d.Get("name").(string))
	if err != nil {
		return err
	}
	resFirestoreRule, err := sendRequest(config, "GET", project, urlFirestoreRule, userAgent, nil)

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("FirebaseRules %q", d.Id()))
	}

	if err := d.Set("rule", flattenRuleFromSource(resFirestoreRule["source"], d, config)); err != nil {
		return fmt.Errorf("Error reading FirebaseRules: %s", err)
	}

	log.Println("[DEBUG] Read Firebase Firestore Rule", d.Get("name"), d.Get("rule"))

	return nil
}

func resourceFirebaseFirestoreRuleDelete(d *schema.ResourceData, meta interface{}) error {
	// config := meta.(*Config)
	// userAgent, err := generateUserAgentString(d, config.userAgent)
	// if err != nil {
	// 	return err
	// }
	// url, err := replaceVars(d, config, "{{FirebaseRulesBasePath}}{{name}}")
	// if err != nil {
	// 	return err
	// }

	// project, err := getProject(d, config)
	// if err != nil {
	// 	return fmt.Errorf("Error fetching project for FirebaseRules: %s", err)
	// }

	// res, err := sendRequestWithTimeout(config, "DELETE", project, url, userAgent, nil, d.Timeout(schema.TimeoutDelete))
	// if err != nil {
	// 	return handleNotFoundError(err, d, "Rule")
	// }

	// log.Printf("[DEBUG] Finished deleting Rule %q: %#v", d.Id(), res)
	log.Printf("[INFO] Not deleting rule - new rule had been made the latest release")
	return nil
}

func resourceFirebaseFirestoreRuleCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}
	url, err := replaceVars(d, config, "{{FirebaseRulesBasePath}}projects/{{project}}/rulesets")
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for FirebaseRules: %s", err)
	}

	log.Printf("[DEBUG] Building new Firebase Rule: %#v", d)

	obj := make(map[string]interface{})

	if !isEmptyValue(reflect.ValueOf(d.Get("rule"))) {
		obj["rule"] = d.Get("rule")
		obj["ruleType"] = "firestore.rules"
	}

	obj, err = resourceFirebaseRuleEncoder(d, meta, obj)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating new Firebase Rule: %#v", obj)

	res, err := sendRequestWithTimeout(config, "POST", project, url, userAgent, obj, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return fmt.Errorf("Error creating Firebase Rule: %s", err)
	}

	// Store the ID now
	d.SetId(res["name"].(string))
	d.Set("name", res["name"].(string))

	log.Printf("[DEBUG] Finished creating Firebase Rule %q: %#v", d.Id(), res)

	log.Printf("[DEBUG] Updating release with new Firebase Rule: %#v", d)

	urlRelease, err := replaceVars(d, config, "{{FirebaseRulesBasePath}}projects/{{project}}/releases/cloud.firestore")
	if err != nil {
		return err
	}

	releaseObj := make(map[string]interface{})
	releaseName, err := replaceVars(d, config, "projects/{{project}}/releases/cloud.firestore")
	releaseObj["name"] = releaseName
	releaseObj["ruleName"] = res["name"].(string)

	releaseObj, err = resourceFirebaseReleasePatchEncoder(d, meta, releaseObj)
	if err != nil {
		log.Printf("[ERROR] Cannot convert release to encoded obj: %v", err)
		return err
	}

	log.Printf("[DEBUG] release obj: %v", releaseObj)
	_, err = sendRequestWithTimeout(config, "PATCH", project, urlRelease, userAgent, releaseObj, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return fmt.Errorf("Error creating Firebase Release: %s", err)
	}

	log.Printf("[DEBUG] Finished release with new Firebase Rule: %#v", d)

	return resourceFirebaseFirestoreRuleRead(d, meta)
}
