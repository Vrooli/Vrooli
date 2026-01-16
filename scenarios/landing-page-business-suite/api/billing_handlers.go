package main

import (
	"encoding/json"
	"net/http"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func handleBillingCreateCheckoutSession(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req landing_page_react_vite_v1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			logStructuredError("billing_checkout_session_failed", map[string]interface{}{
				"error":    err.Error(),
				"price_id": req.PriceId,
			})
			// Stripe errors could be config issues (retryable) or validation (not retryable)
			// Default to server_error since Stripe integration issues are typically transient
			writeJSONError(w, http.StatusBadRequest, "Failed to create checkout session. Please try again.", ApiErrorTypeServerError)
			return
		}
		writeJSON(w, &landing_page_react_vite_v1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingCreateCreditsSession(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req landing_page_react_vite_v1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			logStructuredError("billing_credits_session_failed", map[string]interface{}{
				"error":    err.Error(),
				"price_id": req.PriceId,
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to create credits checkout. Please try again.", ApiErrorTypeServerError)
			return
		}
		writeJSON(w, &landing_page_react_vite_v1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingPortalURL(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := resolveUserIdentity(r)
		if user == "" {
			writeJSONError(w, http.StatusBadRequest, "User identity required", ApiErrorTypeValidation)
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
