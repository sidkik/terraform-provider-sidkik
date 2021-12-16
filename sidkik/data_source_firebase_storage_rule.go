package sidkik

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFirebaseStorageRule() *schema.Resource {

	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceFirebaseStorageRule().Schema)

	// Set 'Optional' schema elements
	addOptionalFieldsToSchema(dsSchema, "project")

	return &schema.Resource{
		Read:   dataSourceFirebaseStorageRuleRead,
		Schema: dsSchema,
	}
}

func dataSourceFirebaseStorageRuleRead(d *schema.ResourceData, meta interface{}) error {
	return resourceFirebaseStorageRuleRead(d, meta)
}
