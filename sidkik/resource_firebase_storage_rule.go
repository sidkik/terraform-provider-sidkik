package sidkik

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirebaseStorageRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirebaseStorageRuleCreate,
		Read:   resourceFirebaseStorageRuleRead,
		// Update: resourceFirebaseStorageRuleCreate,
		Delete: resourceFirebaseStorageRuleDelete,

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
				Description: `name of the storage rule`,
			},
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: `id of the storage rule`,
			},
			"rule": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: `Source for the storage rule. Provided as a string with the correct rules schems`,
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

func resourceFirebaseStorageRuleRead(d *schema.ResourceData, meta interface{}) error {

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

	releases := res["releases"].([]interface{})

	if err := d.Set("name", flattenFirebaseRuleName(releases, "firebase.storage", d, config)); err != nil {
		return fmt.Errorf("Error reading FirebaseRules: %s", err)
	}

	// Set the ID now
	d.SetId(d.Get("name").(string))

	// now grab the actual rule text

	// storage rule

	urlStorageRule, err := replaceVars(d, config, "{{FirebaseRulesBasePath}}"+d.Get("name").(string))
	if err != nil {
		return err
	}
	resStorageRule, err := sendRequest(config, "GET", project, urlStorageRule, userAgent, nil)

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("FirebaseRules %q", d.Id()))
	}

	if err := d.Set("rule", flattenRuleFromSource(resStorageRule["source"], d, config)); err != nil {
		return fmt.Errorf("Error reading FirebaseRules: %s", err)
	}

	log.Println("[DEBUG] Read Firebase Storage Rule", d.Get("name"), d.Get("rule"))

	return nil
}

func resourceFirebaseStorageRuleDelete(d *schema.ResourceData, meta interface{}) error {
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

	log.Printf("[INFO] Not deleting rule - new rule had been made the latest release")
	return nil
}

func resourceFirebaseStorageRuleCreate(d *schema.ResourceData, meta interface{}) error {
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

	obj := make(map[string]interface{})

	if !isEmptyValue(reflect.ValueOf(d.Get("rule"))) {
		obj["rule"] = d.Get("rule")
		obj["ruleType"] = "storage.rules"
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

	urlRelease, err := replaceVars(d, config, "{{FirebaseRulesBasePath}}projects/{{project}}/releases/firebase.storage/{{project}}.appspot.com")
	if err != nil {
		return err
	}

	releaseObj := make(map[string]interface{})
	releaseName, err := replaceVars(d, config, "projects/{{project}}/releases/firebase.storage/{{project}}.appspot.com")
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

	return resourceFirebaseStorageRuleRead(d, meta)
}

func flattenFirebaseRuleName(v []interface{}, ruleType string, d *schema.ResourceData, config *Config) interface{} {
	ls := getLatestRulesetByType(v, ruleType)
	if ls != nil {
		return ls
	}
	return nil
}

func flattenRuleFromSource(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	files := v.(map[string]interface{})["files"]
	fd := files.([]interface{})[0]
	content := fd.(map[string]interface{})["content"]

	return content
}

func getLatestRulesetByType(list []interface{}, typeOfRuleset string) interface{} {
	filtered := []map[string]interface{}{}
	for _, m := range list {
		rulesetName := m.(map[string]interface{})["name"].(string)
		if strings.Contains(rulesetName, typeOfRuleset) {
			filtered = append(filtered, m.(map[string]interface{}))
		}
	}

	// sort it descending if there are rules and grab the first one. this is the latest applied rule in the history of the rules
	if len(filtered) > 0 {
		sort.SliceStable(filtered, func(i, j int) bool {
			iTime, _ := time.Parse(
				time.RFC3339,
				filtered[i]["createTime"].(string))
			jTime, _ := time.Parse(
				time.RFC3339,
				filtered[j]["createTime"].(string))
			return iTime.After(jTime)
		})
		return filtered[0]["rulesetName"]
	}

	return nil
}

func resourceFirebaseRuleEncoder(d *schema.ResourceData, meta interface{}, obj map[string]interface{}) (map[string]interface{}, error) {

	if _, ok := d.GetOk("rule"); ok {
		rule := make(map[string]interface{})
		source := make(map[string]interface{})
		files := []map[string]interface{}{}
		file := make(map[string]interface{})
		file["content"] = obj["rule"]
		file["name"] = obj["ruleType"]
		files = append(files, file)
		source["files"] = files
		rule["source"] = source

		return rule, nil
	}

	return nil, fmt.Errorf("error in Rule %s: No rule block specified.", d.Get("name").(string))
}

func resourceFirebaseReleasePatchEncoder(d *schema.ResourceData, meta interface{}, obj map[string]interface{}) (map[string]interface{}, error) {
	release := make(map[string]interface{})
	release["name"] = obj["name"]
	release["rulesetName"] = obj["ruleName"]
	wrapper := make(map[string]interface{})
	wrapper["updateMask"] = "rulesetName"
	wrapper["release"] = release

	return wrapper, nil
}
