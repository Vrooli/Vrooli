package main

import (
	"encoding/json"
	"net/http"
	"strings"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

// createCheckoutSessionHandler creates a parameterized checkout session handler.
// This consolidates common logic between subscription and credits checkout flows.
func createCheckoutSessionHandler(service *StripeService, logKey, errorMsg string, requireEmail bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req landing_page_business_suite_v1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		req.PriceId = strings.TrimSpace(req.PriceId)
		if req.PriceId == "" {
			writeJSONError(w, http.StatusBadRequest, "price_id is required", ApiErrorTypeValidation)
			return
		}

		req.CustomerEmail = strings.TrimSpace(req.CustomerEmail)
		if requireEmail {
			normalized, ok := ValidateEmailForHandler(w, req.CustomerEmail)
			if !ok {
				return
			}
			req.CustomerEmail = normalized
		} else if req.CustomerEmail != "" {
			normalized, ok := ValidateEmailForHandler(w, req.CustomerEmail)
			if !ok {
				return
			}
			req.CustomerEmail = normalized
		}

		successURL, ok := NormalizeRedirectURLForHandler(w, req.SuccessUrl, "success_url")
		if !ok {
			return
		}
		cancelURL, ok := NormalizeRedirectURLForHandler(w, req.CancelUrl, "cancel_url")
		if !ok {
			return
		}
		req.SuccessUrl = successURL
		req.CancelUrl = cancelURL

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			logStructuredError(logKey, map[string]interface{}{
				"error":    err.Error(),
				"price_id": req.PriceId,
			})
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			// Stripe errors could be config issues (retryable) or validation (not retryable)
			// Default to server_error since Stripe integration issues are typically transient
			writeJSONError(w, http.StatusBadRequest, errorMsg, ApiErrorTypeServerError)
			return
		}
		writeJSON(w, &landing_page_business_suite_v1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingCreateCheckoutSession(service *StripeService) http.HandlerFunc {
	return createCheckoutSessionHandler(
		service,
		"billing_checkout_session_failed",
		"Failed to create checkout session. Please try again.",
		false,
	)
}

func handleBillingCreateCreditsSession(service *StripeService) http.HandlerFunc {
	return createCheckoutSessionHandler(
		service,
		"billing_credits_session_failed",
		"Failed to create credits checkout. Please try again.",
		true,
	)
}

func handleBillingPortalURL(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUserEmail(r.Context())
		if user == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}
		returnURL := strings.TrimSpace(r.URL.Query().Get("return_url"))
		if returnURL != "" {
			normalized, err := ValidateURLOptional(returnURL)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "Invalid return_url format", ApiErrorTypeValidation)
				return
			}
			returnURL = normalized
		}
		resp, err := service.CreateBillingPortalSession(r.Context(), user, returnURL)
		if err != nil {
			logStructuredError("billing_portal_session_failed", map[string]interface{}{
				"error": err.Error(),
				"user":  user,
			})
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			writeJSONError(w, http.StatusBadRequest, "Failed to create billing portal session. Please try again.", ApiErrorTypeServerError)
			return
		}
		writeJSON(w, resp)
	}
}
