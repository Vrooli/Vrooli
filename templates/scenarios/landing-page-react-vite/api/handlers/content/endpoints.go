package content

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the content module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "content_get_public_sections",
		Path:        landingconnect.ContentServiceGetPublicSectionsProcedure,
		Method:      "POST",
		Summary:     "Get public sections",
		Description: "Lists enabled sections for a variant (public).",
		Category:    "content",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"variant_id": "int64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"sections": "ContentSection[]"}},
	},
	{
		ID:          "content_get_sections",
		Path:        landingconnect.ContentServiceGetSectionsProcedure,
		Method:      "POST",
		Summary:     "Get sections",
		Description: "Lists all sections for a variant (admin).",
		Category:    "content",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"variant_id": "int64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"sections": "ContentSection[]"}},
	},
	{
		ID:          "content_get_section",
		Path:        landingconnect.ContentServiceGetSectionProcedure,
		Method:      "POST",
		Summary:     "Get section",
		Description: "Fetches a single section by id (admin).",
		Category:    "content",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "int64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"section": "ContentSection"}},
	},
	{
		ID:          "content_create_section",
		Path:        landingconnect.ContentServiceCreateSectionProcedure,
		Method:      "POST",
		Summary:     "Create section",
		Description: "Creates a section for a variant (admin).",
		Category:    "content",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"variant_id": "int64", "section_type": "string", "content": "object"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"section": "ContentSection"}},
	},
	{
		ID:          "content_update_section",
		Path:        landingconnect.ContentServiceUpdateSectionProcedure,
		Method:      "POST",
		Summary:     "Update section",
		Description: "Replaces a section's content payload (admin).",
		Category:    "content",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "int64", "content": "object"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"section": "ContentSection"}},
	},
	{
		ID:          "content_delete_section",
		Path:        landingconnect.ContentServiceDeleteSectionProcedure,
		Method:      "POST",
		Summary:     "Delete section",
		Description: "Deletes a section by id (admin).",
		Category:    "content",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "int64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"deleted": "bool"}},
	},
}
