package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type bundleCatalogResponse struct {
	Bundles []BundleCatalogEntry `json:"bundles"`
}

type updateBundlePriceRequest struct {
	StripePriceID  *string   `json:"stripe_price_id"`
	PlanName       *string   `json:"plan_name"`
	DisplayWeight  *int      `json:"display_weight"`
	DisplayEnabled *bool     `json:"display_enabled"`
	Subtitle       *string   `json:"subtitle"`
	Badge          *string   `json:"badge"`
	CtaLabel       *string   `json:"cta_label"`
	Highlight      *bool     `json:"highlight"`
	Features       *[]string `json:"features"`
}

func handleAdminBundleCatalog(planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundles, err := planService.ListBundleCatalog(r.Context())
		if err != nil {
			http.Error(w, "Failed to load bundle catalog", http.StatusInternalServerError)
			return
		}

		writeJSON(w, bundleCatalogResponse{Bundles: bundles})
	}
}

func handleAdminUpdateBundlePrice(planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bundleKey := vars["bundle_key"]
		priceID := vars["price_id"]
		if bundleKey == "" || priceID == "" {
			http.Error(w, "Bundle key and price id are required", http.StatusBadRequest)
			return
		}

		var req updateBundlePriceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		updated, err := planService.UpdateBundlePrice(r.Context(), bundleKey, priceID, UpdateBundlePriceInput{
			StripePriceID:  req.StripePriceID,
			PlanName:       req.PlanName,
			DisplayWeight:  req.DisplayWeight,
			DisplayEnabled: req.DisplayEnabled,
			Subtitle:       req.Subtitle,
			Badge:          req.Badge,
			CtaLabel:       req.CtaLabel,
			Highlight:      req.Highlight,
			Features:       req.Features,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, updated)
	}
}

func handleAdminVerifyStripePrice(stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.URL.Query().Get("key"))
		if key == "" {
			http.Error(w, "price key required", http.StatusBadRequest)
			return
		}

		info, err := stripe.VerifyStripePrice(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, info)
	}
}
