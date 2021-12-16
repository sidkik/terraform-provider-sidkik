package sidkik

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccFirebaseRuleStorageDatasource_rule(t *testing.T) {
	t.Parallel()

	context := map[string]interface{}{}

	vcrTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		// CheckDestroy: testAccCheckFirebaseRulesDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccFirebaseRuleStorageDatasource_rule(context),
			},
		},
	})
}

func testAccFirebaseRuleStorageDatasource_rule(context map[string]interface{}) string {
	return Nprintf(`
data "sidkik_firebase_storage_rule" "rule" {
}
`, context)
}
