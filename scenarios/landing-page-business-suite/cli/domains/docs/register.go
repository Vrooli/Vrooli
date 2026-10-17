package docs

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Docs",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "admin-docs-tree", Method: "GET", Path: "/admin/docs/tree", Description: "Get docs tree (admin)"},
			{Name: "admin-docs-content", Method: "GET", Path: "/admin/docs/content", Description: "Get docs content (admin)"},
		}),
	}
}
