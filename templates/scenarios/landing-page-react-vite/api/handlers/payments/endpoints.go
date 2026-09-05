package payments

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the payments module's surface for codegen: the seven
// LandingPagePaymentsService Connect RPCs plus the raw Stripe webhook receiver,
// which is declared as a REST exception (raw body + signature header).
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "payments_create_checkout_session", Path: landingconnect.LandingPagePaymentsServiceCreateCheckoutSessionProcedure, Method: "POST",
		Summary: "Create checkout session", Description: "Creates a simulated Stripe checkout session for a price (public).", Category: "payments",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"price_id": "string", "customer_email": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"session": "CheckoutSession"}},
	},
	{
		ID: "payments_verify_subscription", Path: landingconnect.LandingPagePaymentsServiceVerifySubscriptionProcedure, Method: "POST",
		Summary: "Verify subscription", Description: "Returns the cached subscription status for a user identity (public).", Category: "payments",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"user_identity": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"status": "SubscriptionStatus"}},
	},
	{
		ID: "payments_cancel_subscription", Path: landingconnect.LandingPagePaymentsServiceCancelSubscriptionProcedure, Method: "POST",
		Summary: "Cancel subscription", Description: "Cancels the user's active subscription at period end (admin).", Category: "payments",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"user_identity": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"*": "cancellation outcome"}},
	},
	{
		ID: "payments_get_pricing", Path: landingconnect.LandingPagePaymentsServiceGetPricingProcedure, Method: "POST",
		Summary: "Get pricing", Description: "Returns the public pricing overview for the configured bundle.", Category: "payments",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"pricing": "PricingOverview"}},
	},
	{
		ID: "payments_get_stripe_settings", Path: landingconnect.LandingPagePaymentsServiceGetStripeSettingsProcedure, Method: "POST",
		Summary: "Get Stripe settings", Description: "Returns persisted Stripe settings and the active config snapshot (admin).", Category: "payments",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"settings": "StripeSettings", "snapshot": "StripeConfigSnapshot"}},
	},
	{
		ID: "payments_update_stripe_settings", Path: landingconnect.LandingPagePaymentsServiceUpdateStripeSettingsProcedure, Method: "POST",
		Summary: "Update Stripe settings", Description: "Persists Stripe credentials and refreshes the runtime config (admin).", Category: "payments",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"*": "optional stripe credential fields"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"settings": "StripeSettings", "snapshot": "StripeConfigSnapshot"}},
	},
	{
		ID: "payments_get_billing_portal", Path: landingconnect.LandingPagePaymentsServiceGetBillingPortalProcedure, Method: "POST",
		Summary: "Get billing portal", Description: "Returns the hosted billing portal URL for self-service management.", Category: "payments",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"url": "string"}},
	},
	{
		ID: "payments_webhook", Path: webhookPath, Method: "POST",
		Summary: "Stripe webhook receiver", Description: "Receives simulated Stripe webhook events with a signed raw body.", Category: "payments",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"*": "raw Stripe event JSON"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"status": "string"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonWebhookReceiver,
			Note:   "Stripe webhook contract: raw request body + Stripe-Signature header HMAC verification; not a Connect payload.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "json", Conformance: "none"},
				Response: module.RESTPayload{Transport: "json", Conformance: "none"},
			},
		},
	},
}
