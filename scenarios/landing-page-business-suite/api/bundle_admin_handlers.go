package main

import (
	"net/http"
	"strings"
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
			writeJSONError(w, http.StatusInternalServerError, "Failed to load bundle catalog", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, bundleCatalogResponse{Bundles: bundles})
	}
}

func handleAdminUpdateBundlePrice(planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleKey, ok := getPathParam(r, "bundle_key")
		if !ok || bundleKey == "" {
			writeJSONError(w, http.StatusBadRequest, "Bundle key is required", ApiErrorTypeValidation)
			return
		}
		priceID, ok := getPathParam(r, "price_id")
		if !ok || priceID == "" {
			writeJSONError(w, http.StatusBadRequest, "Price id is required", ApiErrorTypeValidation)
			return
		}

		var req updateBundlePriceRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		input := UpdateBundlePriceInput(req)
		updated, err := planService.UpdateBundlePrice(r.Context(), bundleKey, priceID, input)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, updated)
	}
}

func handleAdminVerifyStripePrice(stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(getQueryParam(r, "key"))
		if key == "" {
			writeJSONError(w, http.StatusBadRequest, "price key required", ApiErrorTypeValidation)
			return
		}

		info, err := stripe.VerifyStripePrice(key)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, info)
	}
}
