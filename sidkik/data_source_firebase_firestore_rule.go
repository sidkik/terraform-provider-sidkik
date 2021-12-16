package sidkik

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFirebaseFirestoreRule() *schema.Resource {

	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceFirebaseFirestoreRule().Schema)

	// Set 'Optional' schema elements
	addOptionalFieldsToSchema(dsSchema, "project")

	return &schema.Resource{
		Read:   dataSourceFirebaseFirestoreRuleRead,
		Schema: dsSchema,
	}
}

func dataSourceFirebaseFirestoreRuleRead(d *schema.ResourceData, meta interface{}) error {
	return resourceFirebaseFirestoreRuleRead(d, meta)
}
