package main

import (
	"encoding/json"
	"io"
	"net/http"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// handleCheckoutCreate creates a new Stripe checkout session
// [REQ:STRIPE-ROUTES] POST /api/v1/checkout/create
func handleCheckoutCreate(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PriceID       string `json:"price_id"`
			CustomerEmail string `json:"customer_email"`
			SuccessURL    string `json:"success_url"`
			CancelURL     string `json:"cancel_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if body.PriceID == "" || body.CustomerEmail == "" {
			http.Error(w, "Missing required fields: price_id, customer_email", http.StatusBadRequest)
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
			logStructured("Failed to create checkout session", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Get Stripe signature header
		signature := r.Header.Get("Stripe-Signature")
		if signature == "" {
			http.Error(w, "Missing Stripe-Signature header", http.StatusBadRequest)
			return
		}

		// Process webhook with signature verification
		if err := service.HandleWebhook(body, signature); err != nil {
			logStructured("Failed to process webhook", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return 200 to acknowledge receipt
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
		})
	}
}

// handleSubscriptionVerify checks subscription status for a user
// [REQ:SUB-VERIFY] GET /api/v1/subscription/verify
func handleSubscriptionVerify(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := r.URL.Query().Get("user")
		if userIdentity == "" {
			http.Error(w, "Missing required parameter: user", http.StatusBadRequest)
			return
		}

		result, err := service.VerifySubscription(userIdentity)
		if err != nil {
			logStructured("Failed to verify subscription", map[string]interface{}{
				"user":  userIdentity,
				"error": err.Error(),
			})
			http.Error(w, "Failed to verify subscription", http.StatusInternalServerError)
			return
		}

		writeJSON(w, &landing_page_react_vite_v1.VerifySubscriptionResponse{Status: result})
	}
}

// handleSubscriptionCancel cancels an active subscription
// [REQ:SUB-CANCEL] POST /api/v1/subscription/cancel
func handleSubscriptionCancel(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			UserIdentity string `json:"user_identity"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if body.UserIdentity == "" {
			http.Error(w, "Missing required field: user_identity", http.StatusBadRequest)
			return
		}

		req := landing_page_react_vite_v1.CancelSubscriptionRequest{UserIdentity: body.UserIdentity}

		result, err := service.CancelSubscription(req.UserIdentity)
		if err != nil {
			logStructured("Failed to cancel subscription", map[string]interface{}{
				"user":  req.UserIdentity,
				"error": err.Error(),
			})
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, result)
	}
}
