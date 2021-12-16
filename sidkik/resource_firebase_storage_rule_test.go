package sidkik

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFirebaseStorageRule_rule(t *testing.T) {
	t.Parallel()

	context := map[string]interface{}{}

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFirebaseStorageRuleDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccFirebaseStorageRule_rule(context),
			},
			{
				ResourceName:      "sidkik_firebase_storage_rule.rule",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccFirebaseStorageRule_rule(context map[string]interface{}) string {
	return Nprintf(`
resource "sidkik_firebase_storage_rule" "rule" {
	rule = <<EOT
rules_version = '2';
service firebase.storage {
	match /b/{bucket}/o {
		match /{allPaths=**} {
			allow read, write: if request.auth != null;
		}
	}
}
EOT
}
`, context)
}

func testAccCheckFirebaseStorageRuleDestroyProducer(t *testing.T) func(s *terraform.State) error {
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

func Test_flattenFirebaseRuleName(t *testing.T) {
	type args struct {
		response string
		ruleType string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "firestore",
			args: args{
				ruleType: "cloud.firestore",
				response: `{
					"releases": [
						{
							"name": "projects/acct-1tcgeonjjgqi/releases/cloud.firestore",
							"rulesetName": "projects/acct-1tcgeonjjgqi/rulesets/016ba66a-680d-4b81-9217-5c754f558abd",
							"createTime": "2021-12-15T18:57:44.361618Z",
							"updateTime": "2021-12-16T18:05:35.515330Z"
						},
						{
							"name": "projects/acct-1tcgeonjjgqi/releases/firebase.storage/acct-1tcgeonjjgqi.appspot.com",
							"rulesetName": "projects/acct-1tcgeonjjgqi/rulesets/cf5bfeae-f139-4c2e-9d05-2805ab316a2f",
							"createTime": "2021-12-15T02:42:56.748909Z",
							"updateTime": "2021-12-16T03:02:16.043347Z"
						}
					]
				}`,
			},
			want: "projects/acct-1tcgeonjjgqi/rulesets/016ba66a-680d-4b81-9217-5c754f558abd",
		},
		{
			name: "storage",
			args: args{
				ruleType: "firebase.storage",
				response: `{
					"releases": [
						{
							"name": "projects/acct-1tcgeonjjgqi/releases/cloud.firestore",
							"rulesetName": "projects/acct-1tcgeonjjgqi/rulesets/016ba66a-680d-4b81-9217-5c754f558abd",
							"createTime": "2021-12-15T18:57:44.361618Z",
							"updateTime": "2021-12-16T18:05:35.515330Z"
						},
						{
							"name": "projects/acct-1tcgeonjjgqi/releases/firebase.storage/acct-1tcgeonjjgqi.appspot.com",
							"rulesetName": "projects/acct-1tcgeonjjgqi/rulesets/cf5bfeae-f139-4c2e-9d05-2805ab316a2f",
							"createTime": "2021-12-15T02:42:56.748909Z",
							"updateTime": "2021-12-16T03:02:16.043347Z"
						}
					]
				}`,
			},
			want: "projects/acct-1tcgeonjjgqi/rulesets/cf5bfeae-f139-4c2e-9d05-2805ab316a2f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			// Unmarshal or Decode the JSON to the interface.
			json.Unmarshal([]byte(tt.args.response), &result)
			if got := flattenFirebaseRuleName(result["releases"].([]interface{}), tt.args.ruleType, nil, nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("flattenFirebaseRuleName() = %v, want %v", got, tt.want)
			}
		})
	}
}
