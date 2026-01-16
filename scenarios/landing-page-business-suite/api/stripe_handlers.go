package main

import (
	"encoding/json"
	"io"
	"net/http"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// handleCheckoutCreate creates a new Stripe checkout session
// [REQ:STRIPE-ROUTES] POST /api/v1/checkout/create
// [REQ:SIGNAL-FEEDBACK] Logs checkout session creation for payment flow observability
func handleCheckoutCreate(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PriceID       string `json:"price_id"`
			CustomerEmail string `json:"customer_email"`
			SuccessURL    string `json:"success_url"`
			CancelURL     string `json:"cancel_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		// Validate required fields
		if body.PriceID == "" || body.CustomerEmail == "" {
			logStructured("checkout_validation_failed", map[string]interface{}{
				"reason":     "missing_required_fields",
				"has_price":  body.PriceID != "",
				"has_email":  body.CustomerEmail != "",
			})
			writeJSONError(w, http.StatusBadRequest, "Missing required fields: price_id, customer_email", ApiErrorTypeValidation)
			return
		}

		// Set default URLs if not provided
		if body.SuccessURL == "" {
			body.SuccessURL = "/success"
		}
		if body.CancelURL == "" {
			body.CancelURL = "/cancel"
		}

		req := landing_page_react_vite_v1.CreateCheckoutSessionRequest{
			PriceId:       body.PriceID,
			CustomerEmail: body.CustomerEmail,
			SuccessUrl:    body.SuccessURL,
			CancelUrl:     body.CancelURL,
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			logStructuredError("checkout_session_create_failed", map[string]interface{}{
				"error":    err.Error(),
				"price_id": req.PriceId,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to create checkout session. Please try again.", ApiErrorTypeServerError)
			return
		}

		// Log successful checkout session creation for payment flow tracking
		logStructured("checkout_session_created", map[string]interface{}{
			"price_id":   req.PriceId,
			"session_id": session.GetSessionId(),
		})

		writeJSON(w, &landing_page_react_vite_v1.CreateCheckoutSessionResponse{Session: session})
	}
}

// handleStripeWebhook processes Stripe webhook events
// [REQ:STRIPE-ROUTES] POST /api/v1/webhooks/stripe
// [REQ:STRIPE-SIG] Verifies webhook signature
func handleStripeWebhook(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read raw body for signature verification
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// Webhook errors should not expose details to Stripe - use generic message
			logStructuredError("webhook_body_read_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to read request body", ApiErrorTypeValidation)
			return
		}
		defer r.Body.Close()

		// Get Stripe signature header
		signature := r.Header.Get("Stripe-Signature")
		if signature == "" {
			logStructuredError("webhook_signature_missing", nil)
			writeJSONError(w, http.StatusBadRequest, "Missing Stripe-Signature header", ApiErrorTypeValidation)
			return
		}

		// Process webhook with signature verification
		if err := service.HandleWebhook(body, signature); err != nil {
			logStructuredError("webhook_processing_failed", map[string]interface{}{
				"error": err.Error(),
			})
			// Return generic error to avoid leaking internal details to webhook caller
			writeJSONError(w, http.StatusBadRequest, "Webhook processing failed", ApiErrorTypeServerError)
			return
		}

		// Return 200 to acknowledge receipt
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
		}); err != nil {
			logStructuredError("webhook_response_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleSubscriptionVerify checks subscription status for a user
// [REQ:SUB-VERIFY] GET /api/v1/subscription/verify
func handleSubscriptionVerify(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := r.URL.Query().Get("user")
		if userIdentity == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing required parameter: user", ApiErrorTypeValidation)
			return
		}

		result, err := service.VerifySubscription(userIdentity)
		if err != nil {
			logStructuredError("subscription_verify_failed", map[string]interface{}{
				"user":  userIdentity,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to verify subscription. Please try again.", ApiErrorTypeServerError)
			return
		}

		writeJSON(w, &landing_page_react_vite_v1.VerifySubscriptionResponse{Status: result})
	}
}

// handleSubscriptionCancel cancels an active subscription
// [REQ:SUB-CANCEL] POST /api/v1/subscription/cancel
// [REQ:SIGNAL-FEEDBACK] Logs subscription cancellation for billing audit trail
func handleSubscriptionCancel(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			UserIdentity string `json:"user_identity"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if body.UserIdentity == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing required field: user_identity", ApiErrorTypeValidation)
			return
		}

		req := landing_page_react_vite_v1.CancelSubscriptionRequest{UserIdentity: body.UserIdentity}

		result, err := service.CancelSubscription(req.UserIdentity)
		if err != nil {
			logStructuredError("subscription_cancel_failed", map[string]interface{}{
				"user":  req.UserIdentity,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to cancel subscription. Please try again.", ApiErrorTypeServerError)
			return
		}

		// Log successful subscription cancellation for billing audit trail
		logStructured("subscription_cancelled", map[string]interface{}{
			"user": req.UserIdentity,
		})

		writeJSON(w, result)
	}
}
