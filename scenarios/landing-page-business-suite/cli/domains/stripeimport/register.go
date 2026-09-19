package stripeimport

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Admin Commerce - Stripe Import",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "admin-stripe-import-preview", Method: "GET", Path: "/admin/stripe/import-preview", Description: "Stripe import preview"},
			{Name: "admin-stripe-import", Method: "POST", Path: "/admin/stripe/import", Description: "Run Stripe import"},
		}),
	}
}
