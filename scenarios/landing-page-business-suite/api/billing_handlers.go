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
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, &landing_page_react_vite_v1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingCreateCreditsSession(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req landing_page_react_vite_v1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		session, err := service.CreateCheckoutSession(req.PriceId, req.SuccessUrl, req.CancelUrl, req.CustomerEmail)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, &landing_page_react_vite_v1.CreateCheckoutSessionResponse{Session: session})
	}
}

func handleBillingPortalURL(service *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := resolveUserIdentity(r)
		if user == "" {
			http.Error(w, "user identity required", http.StatusBadRequest)
			return
		}
		returnURL := r.URL.Query().Get("return_url")
		resp, err := service.CreateBillingPortalSession(r.Context(), user, returnURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, resp)
	}
}
