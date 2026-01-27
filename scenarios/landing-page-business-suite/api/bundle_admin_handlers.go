package main

import (
	"errors"
	"net/http"
	"strings"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

type bundleCatalogResponse struct {
	Bundles []bundleCatalogEntryResponse `json:"bundles"`
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

type bundleCatalogEntryResponse struct {
	Bundle bundleProductResponse `json:"bundle"`
	Prices []planOptionResponse  `json:"prices"`
}

type bundleProductResponse struct {
	BundleKey                string                 `json:"bundle_key"`
	Name                     string                 `json:"name"`
	StripeProductID          string                 `json:"stripe_product_id"`
	CreditsPerUSD            int64                  `json:"credits_per_usd"`
	DisplayCreditsMultiplier float64                `json:"display_credits_multiplier"`
	DisplayCreditsLabel      string                 `json:"display_credits_label"`
	Environment              string                 `json:"environment,omitempty"`
	Metadata                 map[string]interface{} `json:"metadata,omitempty"`
}

type planOptionResponse struct {
	PlanName               string                 `json:"plan_name"`
	PlanTier               string                 `json:"plan_tier"`
	BillingInterval        string                 `json:"billing_interval"`
	AmountCents            int64                  `json:"amount_cents"`
	Currency               string                 `json:"currency"`
	IntroEnabled           bool                   `json:"intro_enabled"`
	IntroType              *string                `json:"intro_type,omitempty"`
	IntroAmountCents       *int64                 `json:"intro_amount_cents,omitempty"`
	IntroPeriods           *int32                 `json:"intro_periods,omitempty"`
	IntroPriceLookupKey    *string                `json:"intro_price_lookup_key,omitempty"`
	StripePriceID          string                 `json:"stripe_price_id"`
	MonthlyIncludedCredits int64                  `json:"monthly_included_credits"`
	OneTimeBonusCredits    int64                  `json:"one_time_bonus_credits"`
	PlanRank               *int32                 `json:"plan_rank,omitempty"`
	BonusType              *string                `json:"bonus_type,omitempty"`
	Kind                   *string                `json:"kind,omitempty"`
	IsVariableAmount       bool                   `json:"is_variable_amount"`
	DisplayEnabled         bool                   `json:"display_enabled"`
	BundleKey              string                 `json:"bundle_key,omitempty"`
	DisplayWeight          int32                  `json:"display_weight"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

func buildBundleCatalogResponse(entries []BundleCatalogEntry) (bundleCatalogResponse, error) {
	bundles := make([]bundleCatalogEntryResponse, 0, len(entries))
	for _, entry := range entries {
		if entry.Bundle == nil {
			return bundleCatalogResponse{}, errors.New("bundle catalog entry missing bundle")
		}
		bundle := bundleProductResponseFromProto(entry.Bundle)
		prices := make([]planOptionResponse, 0, len(entry.Prices))
		for _, price := range entry.Prices {
			if price == nil {
				continue
			}
			prices = append(prices, planOptionResponseFromProto(price))
		}
		bundles = append(bundles, bundleCatalogEntryResponse{
			Bundle: bundle,
			Prices: prices,
		})
	}
	return bundleCatalogResponse{Bundles: bundles}, nil
}

func bundleProductResponseFromProto(bundle *BundleProduct) bundleProductResponse {
	if bundle == nil {
		return bundleProductResponse{}
	}
	response := bundleProductResponse{
		BundleKey:                bundle.BundleKey,
		Name:                     bundle.Name,
		StripeProductID:          bundle.StripeProductId,
		CreditsPerUSD:            bundle.CreditsPerUsd,
		DisplayCreditsMultiplier: bundle.DisplayCreditsMultiplier,
		DisplayCreditsLabel:      bundle.DisplayCreditsLabel,
		Environment:              bundle.Environment,
	}
	if len(bundle.Metadata) > 0 {
		response.Metadata = convertProtoMetadataToMap(bundle.Metadata)
	}
	return response
}

func planOptionResponseFromProto(plan *PlanOption) planOptionResponse {
	if plan == nil {
		return planOptionResponse{}
	}

	interval := billingIntervalLabel(plan.BillingInterval)
	if interval == "unspecified" {
		interval = "one_time"
	}

	var introType *string
	if value := introPricingTypeString(plan.IntroType); value != "" {
		introType = &value
	}

	var introPeriods *int32
	if plan.IntroPeriods > 0 {
		value := plan.IntroPeriods
		introPeriods = &value
	}

	var planRank *int32
	if plan.PlanRank > 0 {
		value := plan.PlanRank
		planRank = &value
	}

	var bonusType *string
	if strings.TrimSpace(plan.BonusType) != "" {
		value := strings.TrimSpace(plan.BonusType)
		bonusType = &value
	}

	var kind *string
	if plan.Kind != landing_page_react_vite_v1.PlanKind_PLAN_KIND_UNSPECIFIED {
		value := planKindString(plan.Kind)
		kind = &value
	}

	response := planOptionResponse{
		PlanName:               plan.PlanName,
		PlanTier:               plan.PlanTier,
		BillingInterval:        interval,
		AmountCents:            plan.AmountCents,
		Currency:               plan.Currency,
		IntroEnabled:           plan.IntroEnabled,
		IntroType:              introType,
		IntroAmountCents:       plan.IntroAmountCents,
		IntroPeriods:           introPeriods,
		IntroPriceLookupKey:    optionalString(plan.IntroPriceLookupKey),
		StripePriceID:          plan.StripePriceId,
		MonthlyIncludedCredits: plan.MonthlyIncludedCredits,
		OneTimeBonusCredits:    plan.OneTimeBonusCredits,
		PlanRank:               planRank,
		BonusType:              bonusType,
		Kind:                   kind,
		IsVariableAmount:       plan.IsVariableAmount,
		DisplayEnabled:         plan.DisplayEnabled,
		BundleKey:              plan.BundleKey,
		DisplayWeight:          plan.DisplayWeight,
	}

	if len(plan.Metadata) > 0 {
		response.Metadata = convertProtoMetadataToMap(plan.Metadata)
	}

	return response
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func handleAdminBundleCatalog(planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundles, err := planService.ListBundleCatalog(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to load bundle catalog", ApiErrorTypeServerError)
			return
		}

		response, err := buildBundleCatalogResponse(bundles)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode bundle catalog", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, response)
	}
}

func handleAdminUpdateBundlePrice(planService *PlanService, stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleKey, ok := getPathParam(r, "bundle_key")
		if !ok || bundleKey == "" {
			writeJSONError(w, http.StatusBadRequest, "Bundle key is required", ApiErrorTypeValidation)
			return
		}
		if bundleKey != planService.BundleKey() {
			writeJSONError(w, http.StatusBadRequest, "Bundle key does not match active bundle", ApiErrorTypeValidation)
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

		var fetcher StripePriceFetcher
		if stripe != nil {
			fetcher = stripe.FetchStripePriceDetails
		}
		updated, err := planService.UpdateBundlePriceWithStripe(r.Context(), bundleKey, priceID, input, fetcher)
		if err != nil {
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, planOptionResponseFromProto(updated))
	}
}

func handleAdminVerifyStripePrice(stripe *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if stripe == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "Stripe service unavailable", ApiErrorTypeServerError)
			return
		}
		key := strings.TrimSpace(getQueryParam(r, "key"))
		if key == "" {
			writeJSONError(w, http.StatusBadRequest, "price key required", ApiErrorTypeValidation)
			return
		}

		info, err := stripe.VerifyStripePrice(key)
		if err != nil {
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, info)
	}
}

// handleAdminStripeImportPreview returns a preview of products/prices available for import from Stripe.
func handleAdminStripeImportPreview(stripe *StripeService, planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if stripe == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "Stripe service unavailable", ApiErrorTypeServerError)
			return
		}
		planStore := planService.GetPlanStore()
		if planStore == nil {
			writeJSONError(w, http.StatusInternalServerError, "plan store not available", ApiErrorTypeServerError)
			return
		}

		preview, err := stripe.ListStripeProductsWithPrices(r.Context(), planStore)
		if err != nil {
			if status, errType, message, ok := classifyStripeError(err); ok {
				writeJSONError(w, status, message, errType)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to load Stripe products", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, preview)
	}
}

// StripeImportRequest contains the selections for importing prices from Stripe.
type StripeImportRequest struct {
	BundleProductID string                `json:"bundle_product_id"`
	Mode            StripeImportMode      `json:"mode,omitempty"`
	Selections      []ImportPlanSelection `json:"selections"`
}

// handleAdminStripeImport imports selected prices from Stripe into the plan store.
func handleAdminStripeImport(stripe *StripeService, planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req StripeImportRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		bundleProductID := strings.TrimSpace(req.BundleProductID)
		if bundleProductID == "" {
			writeJSONError(w, http.StatusBadRequest, "bundle_product_id is required", ApiErrorTypeValidation)
			return
		}

		mode := req.Mode
		if strings.TrimSpace(string(mode)) == "" {
			mode = StripeImportModeMerge
		}

		var fetcher StripePriceFetcher
		if stripe != nil {
			fetcher = stripe.FetchStripePriceDetails
		}

		result, err := planService.ImportStripePricesForProduct(r.Context(), req.Selections, bundleProductID, mode, fetcher)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, errStripeImportNoSelections) ||
				errors.Is(err, errStripeImportNoValidSelections) ||
				errors.Is(err, errStripeImportMissingFetcher) ||
				errors.Is(err, errStripeImportBundleMissing) ||
				errors.Is(err, errStripeImportBundleProductMissing) ||
				errors.Is(err, errStripeImportInvalidMode) ||
				errors.Is(err, errStripeImportProductSwitchRequiresReplace) {
				status = http.StatusBadRequest
			}
			errorType := ApiErrorTypeValidation
			if status == http.StatusInternalServerError {
				errorType = ApiErrorTypeServerError
			}
			writeJSONError(w, status, err.Error(), errorType)
			return
		}

		writeJSONSuccessData(w, result)
	}
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
		if bundleKey != planService.BundleKey() {
			writeJSONError(w, http.StatusBadRequest, "Bundle key does not match active bundle", ApiErrorTypeValidation)
			return
		}

		var req createBundlePriceRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		planStore := planService.GetPlanStore()
		if planStore == nil || planStore.GetBundle() == nil {
			writeJSONError(w, http.StatusInternalServerError, "plan store not available", ApiErrorTypeServerError)
			return
		}

		input := CreateBundlePriceInput{
			StripePriceID:          req.StripePriceID,
			PlanName:               req.PlanName,
			PlanTier:               req.PlanTier,
			BillingInterval:        req.BillingInterval,
			AmountCents:            req.AmountCents,
			Currency:               req.Currency,
			DisplayWeight:          req.DisplayWeight,
			DisplayEnabled:         req.DisplayEnabled,
			MonthlyIncludedCredits: req.MonthlyIncludedCredits,
			Subtitle:               req.Subtitle,
			Badge:                  req.Badge,
			CtaLabel:               req.CtaLabel,
			Highlight:              req.Highlight,
			Features:               req.Features,
		}

		var fetcher StripePriceFetcher
		if stripe != nil {
			fetcher = stripe.FetchStripePriceDetails
		}

		plan, err := planService.CreateBundlePrice(r.Context(), bundleKey, input, fetcher)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, planOptionResponseFromProto(plan))
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
		if bundleKey != planService.BundleKey() {
			writeJSONError(w, http.StatusBadRequest, "Bundle key does not match active bundle", ApiErrorTypeValidation)
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
