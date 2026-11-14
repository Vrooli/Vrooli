package grouptemplates

import (
	grouptemplatesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/grouptemplates/grouptemplates_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the group-templates module's public surface. Connect-RPC
// method paths reference generated *Procedure constants so adding or renaming
// an RPC in grouptemplates.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "grouptemplates_list_templates",
		Path:        grouptemplatesconnect.GroupTemplatesServiceListTemplatesProcedure,
		Method:      "POST",
		Summary:     "List group templates",
		Description: "Returns every saved group template with its ordered role list.",
		Category:    "grouptemplates",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"templates": "[]Template"},
		},
	},
	{
		ID:          "grouptemplates_upsert_template",
		Path:        grouptemplatesconnect.GroupTemplatesServiceUpsertTemplateProcedure,
		Method:      "POST",
		Summary:     "Create or update a group template",
		Description: "Upserts a template by id. A blank id is assigned server-side. Rejects a blank name and an unknown role start_mode.",
		Category:    "grouptemplates",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"template": "Template"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Blank name, or a role with an unknown start_mode"},
		},
	},
	{
		ID:          "grouptemplates_delete_template",
		Path:        grouptemplatesconnect.GroupTemplatesServiceDeleteTemplateProcedure,
		Method:      "POST",
		Summary:     "Delete a group template",
		Description: "Idempotent: succeeds whether the id exists or not. Every template is deletable, including a shipped example.",
		Category:    "grouptemplates",
	},
}
