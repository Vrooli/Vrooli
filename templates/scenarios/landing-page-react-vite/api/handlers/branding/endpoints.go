package branding

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the branding module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "branding_get",
		Path:        landingconnect.BrandingServiceGetBrandingProcedure,
		Method:      "POST",
		Summary:     "Get site branding",
		Description: "Returns the full singleton site-branding record (admin).",
		Category:    "branding",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"branding": "SiteBranding"}},
	},
	{
		ID:          "branding_update",
		Path:        landingconnect.BrandingServiceUpdateBrandingProcedure,
		Method:      "POST",
		Summary:     "Update site branding",
		Description: "Partially updates branding; unset fields are preserved (admin).",
		Category:    "branding",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"*": "optional branding fields"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"branding": "SiteBranding"}},
	},
	{
		ID:          "branding_clear_field",
		Path:        landingconnect.BrandingServiceClearBrandingFieldProcedure,
		Method:      "POST",
		Summary:     "Clear a branding field",
		Description: "Nulls a single nullable branding field (admin).",
		Category:    "branding",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"field": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"branding": "SiteBranding"}},
	},
	{
		ID:          "branding_get_public",
		Path:        landingconnect.BrandingServiceGetPublicBrandingProcedure,
		Method:      "POST",
		Summary:     "Get public branding",
		Description: "Returns the redacted public branding subset.",
		Category:    "branding",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"branding": "PublicBranding"}},
	},
}
