package main

import (
	"encoding/json"
	"net/http"

	lprvv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func handleBillingCreateCheckoutSession(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req lprvv1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, &lprvv1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingCreateCreditsSession(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req lprvv1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, &lprvv1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingPortalURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, &lprvv1.BillingPortalResponse{
			Url: "https://dashboard.stripe.com/test/customers",
		})
	}
}
