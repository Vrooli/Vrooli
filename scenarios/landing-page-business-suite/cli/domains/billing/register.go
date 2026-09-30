package billing

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Billing & Payments",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "billing-checkout", Method: "POST", Path: "/billing/create-checkout-session", Description: "Create billing checkout session"},
			{Name: "billing-credits-checkout", Method: "POST", Path: "/billing/create-credits-checkout-session", Description: "Create credits checkout session"},
			{Name: "billing-portal", Method: "GET", Path: "/billing/portal-url", Description: "Get billing portal URL"},
			{Name: "checkout-create", Method: "POST", Path: "/checkout/create", Description: "Create checkout"},
			{Name: "webhook-stripe", Method: "POST", Path: "/webhooks/stripe", Description: "Send Stripe webhook payload"},
			{Name: "subscription-verify", Method: "GET", Path: "/subscription/verify", Description: "Verify subscription"},
			{Name: "subscription-cancel", Method: "POST", Path: "/subscription/cancel", Description: "Cancel subscription (admin)"},
		}),
	}
}
