package main

import (
	"encoding/json"
	"net/http"
)

type checkoutRequest struct {
	PriceID       string `json:"price_id"`
	CustomerEmail string `json:"customer_email"`
	SuccessURL    string `json:"success_url"`
	CancelURL     string `json:"cancel_url"`
}

func handleBillingCreateCheckoutSession(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req checkoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceID, req.SuccessURL, req.CancelURL, req.CustomerEmail)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, session)
	}
}

func handleBillingCreateCreditsSession(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req checkoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceID, req.SuccessURL, req.CancelURL, req.CustomerEmail)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, session)
	}
}

func handleBillingPortalURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Placeholder implementation – in production this would create a Stripe Billing Portal session
		writeJSON(w, map[string]string{
			"url": "https://dashboard.stripe.com/test/customers",
		})
	}
}
