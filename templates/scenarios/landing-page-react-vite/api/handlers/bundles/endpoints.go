package bundles

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the bundle-admin module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "bundles_list_catalog", Path: landingconnect.BundleAdminServiceListBundleCatalogProcedure, Method: "POST",
		Summary: "List bundle catalog", Description: "Lists all bundles and prices, including hidden ones (admin).", Category: "bundles",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"bundles": "BundleCatalogEntry[]"}},
	},
	{
		ID: "bundles_update_price", Path: landingconnect.BundleAdminServiceUpdateBundlePriceProcedure, Method: "POST",
		Summary: "Update bundle price", Description: "Partially updates a single price row's display metadata (admin).", Category: "bundles",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"bundle_key": "string", "price_id": "string", "*": "optional display fields"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"price": "PlanOption"}},
	},
}
