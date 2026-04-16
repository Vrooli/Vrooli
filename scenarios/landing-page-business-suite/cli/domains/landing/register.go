package landing

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Landing",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "landing-config", Method: "GET", Path: "/landing-config", Description: "Fetch landing configuration"},
			{Name: "plans", Method: "GET", Path: "/plans", Description: "List pricing plans"},
			{Name: "variant-space", Method: "GET", Path: "/variant-space", Description: "Fetch variant space"},
			{Name: "customize", Method: "POST", Path: "/customize", Description: "Customize landing content"},
		}),
	}
}
