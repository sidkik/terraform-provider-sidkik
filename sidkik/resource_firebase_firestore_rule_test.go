package sidkik

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFirebaseFirestoreRule_rule(t *testing.T) {
	t.Parallel()

	context := map[string]interface{}{}

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFirebaseFirestoreRuleDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccFirebaseFirestoreRule_rule(context),
			},
			{
				ResourceName:      "sidkik_firebase_firestore_rule.rule",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccFirebaseFirestoreRule_rule(context map[string]interface{}) string {
	return Nprintf(`
resource "sidkik_firebase_firestore_rule" "rule" {
	rule = <<EOT
rules_version = '2';
service cloud.firestore {
	match /databases/{database}/documents {
		match /{document=**} {
				// hello
			allow read, write: if false;
		}
	}
}
EOT
}
`, context)
}

func testAccCheckFirebaseFirestoreRuleDestroyProducer(t *testing.T) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		// for name, rs := range s.RootModule().Resources {
		// 	if rs.Type != "sidkik_firebase" {
		// 		continue
		// 	}
		// 	if strings.HasPrefix(name, "data.") {
		// 		continue
		// 	}

		// 	config := googleProviderConfig(t)

		// 	url, err := replaceVarsForTest(config, rs, "{{FirebaseRulesBasePath}}projects/{{project}}/rulesets")
		// 	if err != nil {
		// 		return err
		// 	}

		// 	project := ""

		// 	if config.Project != "" {
		// 		project = config.Project
		// 	}

		// 	_, err = sendRequest(config, "GET", project, url, config.userAgent, nil)
		// 	if err == nil {
		// 		return fmt.Errorf("ComputeHealthCheck still exists at %s", url)
		// 	}
		// }

		return nil
	}
}
