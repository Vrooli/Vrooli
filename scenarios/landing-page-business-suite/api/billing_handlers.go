package main

import (
	"encoding/json"
	"net/http"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// createCheckoutSessionHandler creates a parameterized checkout session handler.
// This consolidates common logic between subscription and credits checkout flows.
func createCheckoutSessionHandler(service *StripeService, logKey, errorMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req landing_page_react_vite_v1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			logStructuredError(logKey, map[string]interface{}{
				"error":    err.Error(),
				"price_id": req.PriceId,
			})
			// Stripe errors could be config issues (retryable) or validation (not retryable)
			// Default to server_error since Stripe integration issues are typically transient
			writeJSONError(w, http.StatusBadRequest, errorMsg, ApiErrorTypeServerError)
			return
		}
		writeJSON(w, &landing_page_react_vite_v1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingCreateCheckoutSession(service *StripeService) http.HandlerFunc {
	return createCheckoutSessionHandler(
		service,
		"billing_checkout_session_failed",
		"Failed to create checkout session. Please try again.",
	)
}

func handleBillingCreateCreditsSession(service *StripeService) http.HandlerFunc {
	return createCheckoutSessionHandler(
		service,
		"billing_credits_session_failed",
		"Failed to create credits checkout. Please try again.",
	)
}

func handleBillingPortalURL(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUserEmail(r.Context())
		if user == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}
		returnURL := r.URL.Query().Get("return_url")
		resp, err := service.CreateBillingPortalSession(r.Context(), user, returnURL)
		if err != nil {
			logStructuredError("billing_portal_session_failed", map[string]interface{}{
				"error": err.Error(),
				"user":  user,
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to create billing portal session. Please try again.", ApiErrorTypeServerError)
			return
		}
		writeJSON(w, resp)
	}
}
