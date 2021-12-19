package sidkik

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFirebaseAuthConfig() *schema.Resource {

	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceFirebaseAuthConfig().Schema)

	// Set 'Optional' schema elements
	addOptionalFieldsToSchema(dsSchema, "project")

	return &schema.Resource{
		Read:   dataSourceFirebaseAuthConfigRead,
		Schema: dsSchema,
	}
}

func dataSourceFirebaseAuthConfigRead(d *schema.ResourceData, meta interface{}) error {
	return resourceFirebaseAuthConfigRead(d, meta)
}
