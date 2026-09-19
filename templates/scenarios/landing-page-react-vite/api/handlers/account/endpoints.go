package account

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the account module's Connect-RPC surface for codegen.
// All three RPCs read the caller identity from the X-User-Email header.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "account_my_subscription", Path: landingconnect.AccountServiceGetMySubscriptionProcedure, Method: "POST",
		Summary: "Get my subscription", Description: "Returns the caller's cached subscription status (identity via X-User-Email).", Category: "account",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"status": "SubscriptionStatus"}},
	},
	{
		ID: "account_my_credits", Path: landingconnect.AccountServiceGetMyCreditsProcedure, Method: "POST",
		Summary: "Get my credits", Description: "Returns the caller's credit balance and display labelling (identity via X-User-Email).", Category: "account",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"balance": "CreditsBalance", "display_credits_label": "string", "display_credits_multiplier": "number"}},
	},
	{
		ID: "account_entitlements", Path: landingconnect.AccountServiceGetEntitlementsProcedure, Method: "POST",
		Summary: "Get entitlements", Description: "Returns the caller's computed entitlements (status, tier, features, credits).", Category: "account",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"*": "entitlements payload"}},
	},
}
