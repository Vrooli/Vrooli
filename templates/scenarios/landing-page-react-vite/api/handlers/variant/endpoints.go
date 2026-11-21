package variant

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the variant module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "variant_select", Path: landingconnect.VariantServiceSelectVariantProcedure, Method: "POST",
		Summary: "Select variant", Description: "Returns a weighted-random active variant (public).", Category: "variant",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variant": "Variant"}},
	},
	{
		ID: "variant_get_public", Path: landingconnect.VariantServiceGetPublicVariantProcedure, Method: "POST",
		Summary: "Get public variant", Description: "Fetches an active variant by slug (public).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variant": "Variant"}},
	},
	{
		ID: "variant_get", Path: landingconnect.VariantServiceGetVariantProcedure, Method: "POST",
		Summary: "Get variant", Description: "Fetches a variant by slug including SEO (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variant": "Variant"}},
	},
	{
		ID: "variant_list", Path: landingconnect.VariantServiceListVariantsProcedure, Method: "POST",
		Summary: "List variants", Description: "Lists variants filtered by status (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"status_filter": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variants": "Variant[]"}},
	},
	{
		ID: "variant_create", Path: landingconnect.VariantServiceCreateVariantProcedure, Method: "POST",
		Summary: "Create variant", Description: "Creates a new variant, copying control sections (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string", "name": "string", "axes": "object"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variant": "Variant"}},
	},
	{
		ID: "variant_update", Path: landingconnect.VariantServiceUpdateVariantProcedure, Method: "POST",
		Summary: "Update variant", Description: "Partially updates a variant (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string", "*": "optional variant fields"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variant": "Variant"}},
	},
	{
		ID: "variant_archive", Path: landingconnect.VariantServiceArchiveVariantProcedure, Method: "POST",
		Summary: "Archive variant", Description: "Archives a variant (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variant": "Variant"}},
	},
	{
		ID: "variant_delete", Path: landingconnect.VariantServiceDeleteVariantProcedure, Method: "POST",
		Summary: "Delete variant", Description: "Soft-deletes a variant (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deleted": "bool"}},
	},
	{
		ID: "variant_export_snapshot", Path: landingconnect.VariantServiceExportVariantSnapshotProcedure, Method: "POST",
		Summary: "Export variant snapshot", Description: "Exports a variant snapshot (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"snapshot": "VariantSnapshot"}},
	},
	{
		ID: "variant_import_snapshot", Path: landingconnect.VariantServiceImportVariantSnapshotProcedure, Method: "POST",
		Summary: "Import variant snapshot", Description: "Replaces a variant from a snapshot (admin).", Category: "variant",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"slug": "string", "snapshot": "VariantSnapshot"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"variant": "Variant"}},
	},
}
