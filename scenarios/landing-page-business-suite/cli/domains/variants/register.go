package variants

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Variants",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "variants-select", Method: "GET", Path: "/variants/select", Description: "Select a variant for visitor"},
			{Name: "public-variant", Method: "GET", Path: "/public/variants/{slug}", Description: "Get public variant by slug"},
			{Name: "public-variant-sections", Method: "GET", Path: "/public/variants/{variant_slug}/sections", Description: "Get public sections for a variant"},
			{Name: "variants-list", Method: "GET", Path: "/variants", Description: "List variants (admin)"},
			{Name: "variants-get", Method: "GET", Path: "/variants/{slug}", Description: "Get variant by slug (admin)"},
			{Name: "variants-update", Method: "PATCH", Path: "/variants/{slug}", Description: "Update variant by slug (admin)"},
			{Name: "variants-delete", Method: "DELETE", Path: "/variants/{slug}", Description: "Delete variant by slug (admin)"},
			{Name: "variants-sections", Method: "GET", Path: "/variants/{variant_slug}/sections", Description: "Get variant sections (admin)"},
			{Name: "admin-variants-sync", Method: "POST", Path: "/admin/variants/sync", Description: "Sync variants from snapshots"},
			{Name: "admin-variants-export", Method: "GET", Path: "/admin/variants/{slug}/export", Description: "Export variant snapshot"},
			{Name: "admin-variants-import", Method: "PUT", Path: "/admin/variants/{slug}/import", Description: "Import variant snapshot"},
		}),
	}
}
