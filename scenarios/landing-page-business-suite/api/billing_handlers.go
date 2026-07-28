package main

import (
	"context"
	"net/http"

	billinghttp "landing-page-business-suite-api/handlers/billing"
)

func billingDependencies(service *StripeService) billinghttp.Dependencies {
	return billinghttp.Dependencies{
		ValidateEmail:       ValidateEmailForHandler,
		NormalizeRedirect:   NormalizeRedirectURLForHandler,
		ValidateOptionalURL: ValidateURLOptional,
		CreateCheckout:      service.CreateCheckoutSession,
		CreatePortal: func(ctx context.Context, user, returnURL string) (any, error) {
			return service.CreateBillingPortalSession(ctx, user, returnURL)
		},
		UserEmail:     getUserEmail,
		ClassifyError: classifyStripeError,
		WriteJSON:     writeJSON,
		WriteError:    writeJSONError,
		Log:           logStructuredError,
	}
}

func billingConnectDependencies(service *StripeService) billinghttp.ConnectDependencies {
	return billinghttp.ConnectDependencies{
		Payments:            service,
		ValidateEmail:       ValidateEmail,
		NormalizeRedirect:   NormalizeRedirectURL,
		ValidateOptionalURL: ValidateURLOptional,
		UserEmail:           getUserEmail,
	}
}

// createCheckoutSessionHandler remains a composition seam for focused tests;
// handlers/billing owns the request validation and response behavior.
func createCheckoutSessionHandler(service *StripeService, logKey, errorMessage string, requireEmail bool) http.HandlerFunc {
	return billinghttp.Checkout(billingDependencies(service), logKey, errorMessage, requireEmail)
}

func handleBillingCreateCheckoutSession(service *StripeService) http.HandlerFunc {
	return billinghttp.Checkout(billingDependencies(service), "billing_checkout_session_failed", "Failed to create checkout session. Please try again.", false)
}

func handleBillingCreateCreditsSession(service *StripeService) http.HandlerFunc {
	return billinghttp.Checkout(billingDependencies(service), "billing_credits_session_failed", "Failed to create credits checkout. Please try again.", true)
}

func handleBillingPortalURL(service *StripeService) http.HandlerFunc {
	return billinghttp.Portal(billingDependencies(service))
}
