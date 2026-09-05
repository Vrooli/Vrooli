package account

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Account",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "me-subscription", Method: "GET", Path: "/me/subscription", Description: "Get current subscription"},
			{Name: "me-credits", Method: "GET", Path: "/me/credits", Description: "Get credit balance"},
			{Name: "entitlements", Method: "GET", Path: "/entitlements", Description: "Get current entitlements"},
			{Name: "downloads", Method: "GET", Path: "/downloads", Description: "List available downloads"},
		}),
	}
}
