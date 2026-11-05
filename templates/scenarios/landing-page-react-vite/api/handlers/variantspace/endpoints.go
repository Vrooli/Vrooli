package variantspace

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the variant_space module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "variant_space_get", Path: landingconnect.VariantSpaceServiceGetVariantSpaceProcedure, Method: "POST",
		Summary: "Get variant space", Description: "Returns the active variant space as verbatim JSON bytes.", Category: "variant_space",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"raw_json": "bytes"}},
	},
}
