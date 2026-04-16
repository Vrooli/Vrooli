package bundles

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Admin Commerce - Bundles",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "admin-bundles", Method: "GET", Path: "/admin/bundles", Description: "List bundles"},
			{Name: "admin-bundle-price-update", Method: "PATCH", Path: "/admin/bundles/{bundle_key}/prices/{price_id}", Description: "Update bundle price"},
			{Name: "admin-bundle-price-create", Method: "POST", Path: "/admin/bundles/{bundle_key}/prices", Description: "Create bundle price"},
			{Name: "admin-bundle-price-delete", Method: "DELETE", Path: "/admin/bundles/{bundle_key}/prices/{price_id}", Description: "Delete bundle price"},
		}),
	}
}
