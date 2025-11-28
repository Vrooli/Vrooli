package main

import (
	"encoding/json"
	"io"
	"net/http"
)

// handleCheckoutCreate creates a new Stripe checkout session
// [REQ:STRIPE-ROUTES] POST /api/v1/checkout/create
func handleCheckoutCreate(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PriceID       string `json:"price_id"`
			CustomerEmail string `json:"customer_email"`
			SuccessURL    string `json:"success_url"`
			CancelURL     string `json:"cancel_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.PriceID == "" || req.CustomerEmail == "" {
			http.Error(w, "Missing required fields: price_id, customer_email", http.StatusBadRequest)
			return
		}

		// Set default URLs if not provided
		if req.SuccessURL == "" {
			req.SuccessURL = "/success"
		}
		if req.CancelURL == "" {
			req.CancelURL = "/cancel"
		}

		session, err := service.CreateCheckoutSession(req.PriceID, req.SuccessURL, req.CancelURL, req.CustomerEmail)
		if err != nil {
			logStructured("Failed to create checkout session", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// handleSubscriptionCancel cancels an active subscription
// [REQ:SUB-CANCEL] POST /api/v1/subscription/cancel
func handleSubscriptionCancel(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserIdentity string `json:"user_identity"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.UserIdentity == "" {
			http.Error(w, "Missing required field: user_identity", http.StatusBadRequest)
			return
		}

		result, err := service.CancelSubscription(req.UserIdentity)
		if err != nil {
			logStructured("Failed to cancel subscription", map[string]interface{}{
				"user":  req.UserIdentity,
				"error": err.Error(),
			})
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
