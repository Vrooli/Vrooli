package main

import (
	"net/http"

	billinghttp "landing-page-business-suite-api/handlers/commerce"
)

// Root tests characterize the Stripe callback while production routes bind the
// commerce handler directly.
func handleStripeWebhook(service *StripeService) http.HandlerFunc {
	return billinghttp.Webhook(billingWebhookDependencies(service))
}
