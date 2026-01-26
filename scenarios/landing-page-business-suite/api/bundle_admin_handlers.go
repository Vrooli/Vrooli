package main

import (
	"context"
	"net/http"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
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

// handleAdminStripeImportPreview returns a preview of products/prices available for import from Stripe.
func handleAdminStripeImportPreview(stripe *StripeService, planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		planStore := planService.GetPlanStore()
		if planStore == nil {
			writeJSONError(w, http.StatusInternalServerError, "plan store not available", ApiErrorTypeServerError)
			return
		}

		preview, err := stripe.ListStripeProductsWithPrices(r.Context(), planStore)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error(), ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, preview)
	}
}

// ImportPlanSelection represents a single price selection for import.
type ImportPlanSelection struct {
	PriceID string `json:"price_id"`
	Action  string `json:"action"` // "import", "overwrite", "skip"
}

// StripeImportRequest contains the selections for importing prices from Stripe.
type StripeImportRequest struct {
	Selections []ImportPlanSelection `json:"selections"`
}

// StripeImportResult contains the results of the import operation.
type StripeImportResult struct {
	Imported   int      `json:"imported"`
	Overwritten int      `json:"overwritten"`
	Skipped    int      `json:"skipped"`
	Errors     []string `json:"errors,omitempty"`
}

// handleAdminStripeImport imports selected prices from Stripe into the plan store.
func handleAdminStripeImport(stripe *StripeService, planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req StripeImportRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		if len(req.Selections) == 0 {
			writeJSONError(w, http.StatusBadRequest, "no selections provided", ApiErrorTypeValidation)
			return
		}

		planStore := planService.GetPlanStore()
		if planStore == nil {
			writeJSONError(w, http.StatusInternalServerError, "plan store not available", ApiErrorTypeServerError)
			return
		}

		result := StripeImportResult{}
		ctx := r.Context()

		for _, selection := range req.Selections {
			priceID := strings.TrimSpace(selection.PriceID)
			if priceID == "" {
				result.Errors = append(result.Errors, "empty price ID in selection")
				continue
			}

			switch selection.Action {
			case "skip":
				result.Skipped++
				continue
			case "import", "overwrite":
				// Fetch price details from Stripe
				priceDetails, err := stripe.FetchStripePriceDetails(ctx, priceID)
				if err != nil {
					result.Errors = append(result.Errors, "failed to fetch price "+priceID+": "+err.Error())
					continue
				}

				// Convert to PlanOption
				plan := stripePriceImportToPlanOption(priceDetails)

				// Check if exists
				existing, _ := planStore.GetPlanByPriceID(priceID)
				if existing != nil {
					if selection.Action == "overwrite" {
						// Update existing plan with Stripe data
						input := UpdateBundlePriceInput{
							PlanName: &plan.PlanName,
						}
						if _, err := planStore.UpdatePlan(priceID, input); err != nil {
							result.Errors = append(result.Errors, "failed to update "+priceID+": "+err.Error())
							continue
						}
						result.Overwritten++
					} else {
						// Skip existing
						result.Skipped++
					}
				} else {
					// Add new plan
					if err := planStore.AddPlan(plan); err != nil {
						result.Errors = append(result.Errors, "failed to add "+priceID+": "+err.Error())
						continue
					}
					result.Imported++
				}
			default:
				result.Errors = append(result.Errors, "unknown action: "+selection.Action)
			}
		}

		writeJSONSuccessData(w, result)
	}
}

// stripePriceImportToPlanOption converts a StripePriceImport to a PlanOption.
func stripePriceImportToPlanOption(price *StripePriceImport) *PlanOption {
	interval := mapBillingInterval(price.Interval)

	// Derive plan tier from product name or default to "pro"
	planTier := derivePlanTierFromName(price.ProductName)

	return &PlanOption{
		StripePriceId:   price.PriceID,
		PlanName:        price.ProductName,
		PlanTier:        planTier,
		BillingInterval: interval,
		AmountCents:     price.AmountCents,
		Currency:        price.Currency,
		DisplayEnabled:  price.Active,
		DisplayWeight:   10,
		Kind:            landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUBSCRIPTION,
	}
}

// derivePlanTierFromName attempts to derive a plan tier from a product/price name.
func derivePlanTierFromName(name string) string {
	lower := strings.ToLower(name)
	tiers := []string{"free", "solo", "pro", "studio", "business"}
	for _, tier := range tiers {
		if strings.Contains(lower, tier) {
			return tier
		}
	}
	return "pro" // Default tier
}

// createBundlePriceRequest contains fields for creating a new plan.
type createBundlePriceRequest struct {
	StripePriceID          string   `json:"stripe_price_id"`
	PlanName               string   `json:"plan_name"`
	PlanTier               string   `json:"plan_tier"`
	BillingInterval        string   `json:"billing_interval"`
	AmountCents            *int64   `json:"amount_cents"`
	Currency               *string  `json:"currency"`
	DisplayWeight          *int32   `json:"display_weight"`
	DisplayEnabled         *bool    `json:"display_enabled"`
	MonthlyIncludedCredits *int64   `json:"monthly_included_credits"`
	Subtitle               *string  `json:"subtitle"`
	Badge                  *string  `json:"badge"`
	CtaLabel               *string  `json:"cta_label"`
	Highlight              *bool    `json:"highlight"`
	Features               []string `json:"features"`
}

// handleAdminCreateBundlePrice creates a new plan in the plan store.
func handleAdminCreateBundlePrice(planService *PlanService, stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleKey, ok := getPathParam(r, "bundle_key")
		if !ok || bundleKey == "" {
			writeJSONError(w, http.StatusBadRequest, "Bundle key is required", ApiErrorTypeValidation)
			return
		}

		var req createBundlePriceRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		// Validate required fields
		if req.StripePriceID == "" {
			writeJSONError(w, http.StatusBadRequest, "stripe_price_id is required", ApiErrorTypeValidation)
			return
		}
		if req.PlanName == "" {
			writeJSONError(w, http.StatusBadRequest, "plan_name is required", ApiErrorTypeValidation)
			return
		}
		if req.PlanTier == "" {
			writeJSONError(w, http.StatusBadRequest, "plan_tier is required", ApiErrorTypeValidation)
			return
		}
		if req.BillingInterval == "" {
			writeJSONError(w, http.StatusBadRequest, "billing_interval is required", ApiErrorTypeValidation)
			return
		}

		// Optionally verify the price exists in Stripe
		var amountCents int64
		var currency string = "usd"

		if strings.HasPrefix(req.StripePriceID, "price_") {
			priceDetails, err := stripe.FetchStripePriceDetails(context.Background(), req.StripePriceID)
			if err != nil {
				// Price not found in Stripe, but we can still create it locally
				logStructured("stripe_price_not_verified", map[string]interface{}{
					"price_id": req.StripePriceID,
					"error":    err.Error(),
				})
			} else {
				amountCents = priceDetails.AmountCents
				currency = priceDetails.Currency
			}
		}

		// Override with request values if provided
		if req.AmountCents != nil {
			amountCents = *req.AmountCents
		}
		if req.Currency != nil && *req.Currency != "" {
			currency = *req.Currency
		}

		displayWeight := int32(10)
		if req.DisplayWeight != nil {
			displayWeight = *req.DisplayWeight
		}

		displayEnabled := true
		if req.DisplayEnabled != nil {
			displayEnabled = *req.DisplayEnabled
		}

		var monthlyCredits int64
		if req.MonthlyIncludedCredits != nil {
			monthlyCredits = *req.MonthlyIncludedCredits
		}

		// Build metadata
		metadata := make(map[string]*commonv1.JsonValue)
		if req.Subtitle != nil && strings.TrimSpace(*req.Subtitle) != "" {
			metadata["subtitle"] = newStringJsonValue(strings.TrimSpace(*req.Subtitle))
		}
		if req.Badge != nil && strings.TrimSpace(*req.Badge) != "" {
			metadata["badge"] = newStringJsonValue(strings.TrimSpace(*req.Badge))
		}
		if req.CtaLabel != nil && strings.TrimSpace(*req.CtaLabel) != "" {
			metadata["cta_label"] = newStringJsonValue(strings.TrimSpace(*req.CtaLabel))
		}
		if req.Highlight != nil && *req.Highlight {
			metadata["highlight"] = newBoolJsonValue(true)
		}
		if len(req.Features) > 0 {
			listValues := make([]*commonv1.JsonValue, 0, len(req.Features))
			for _, f := range req.Features {
				if trimmed := strings.TrimSpace(f); trimmed != "" {
					listValues = append(listValues, newStringJsonValue(trimmed))
				}
			}
			if len(listValues) > 0 {
				metadata["features"] = newListJsonValue(listValues)
			}
		}

		plan := &PlanOption{
			StripePriceId:          strings.TrimSpace(req.StripePriceID),
			PlanName:               strings.TrimSpace(req.PlanName),
			PlanTier:               strings.ToLower(strings.TrimSpace(req.PlanTier)),
			BillingInterval:        mapBillingInterval(req.BillingInterval),
			AmountCents:            amountCents,
			Currency:               currency,
			DisplayWeight:          displayWeight,
			DisplayEnabled:         displayEnabled,
			MonthlyIncludedCredits: monthlyCredits,
			Kind:                   landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUBSCRIPTION,
			BundleKey:              bundleKey,
		}

		if len(metadata) > 0 {
			plan.Metadata = metadata
		}

		planStore := planService.GetPlanStore()
		if planStore == nil {
			writeJSONError(w, http.StatusInternalServerError, "plan store not available", ApiErrorTypeServerError)
			return
		}

		if err := planStore.AddPlan(plan); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, plan)
	}
}

// handleAdminDeleteBundlePrice deletes a plan from the plan store.
func handleAdminDeleteBundlePrice(planService *PlanService) http.HandlerFunc {
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

		planStore := planService.GetPlanStore()
		if planStore == nil {
			writeJSONError(w, http.StatusInternalServerError, "plan store not available", ApiErrorTypeServerError)
			return
		}

		if err := planStore.DeletePlan(priceID); err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error(), ApiErrorTypeNotFound)
			return
		}

		writeJSONSuccess(w, "Plan deleted successfully")
	}
}
