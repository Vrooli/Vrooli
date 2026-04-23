package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"unicode"

	"google.golang.org/protobuf/types/known/structpb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// PlanService exposes helper utilities for pricing/plan metadata.
// This service delegates to PlanStore for file-based storage.
type PlanService struct {
	planStore     *PlanStore
	defaultBundle string
	displayEnv    string
}

type (
	// BundleProduct is a thin alias to the shared protobuf bundle for readability.
	BundleProduct   = landing_page_react_vite_v1.Bundle
	PlanOption      = landing_page_react_vite_v1.PlanOption
	PricingOverview = landing_page_react_vite_v1.PricingOverview
)

// StripePriceFetcher resolves Stripe price details for plan updates.
type StripePriceFetcher func(ctx context.Context, priceID string) (*StripePriceImport, error)

// ImportPlanSelection represents a single price selection for import.
type ImportPlanSelection struct {
	PriceID string `json:"price_id"`
	Action  string `json:"action"` // "import", "overwrite", "skip"
}

// StripeImportMode controls how Stripe imports reconcile existing plans.
type StripeImportMode string

const (
	StripeImportModeMerge   StripeImportMode = "merge"
	StripeImportModeReplace StripeImportMode = "replace"
)

// StripeImportResult contains the results of the import operation.
type StripeImportResult struct {
	Imported    int      `json:"imported"`
	Overwritten int      `json:"overwritten"`
	Skipped     int      `json:"skipped"`
	Errors      []string `json:"errors,omitempty"`
}

// NewPlanService creates a PlanService with a PlanStore.
// The db parameter is kept for backward compatibility but is no longer used for plans.
func NewPlanService(db *sql.DB) *PlanService {
	bundle := stringsTrimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production")

	// Create and load the plan store
	plansPath := resolvePlansPath()
	planStore := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  bundle,
		DisplayEnv: env,
	})
	if err := planStore.LoadAll(); err != nil {
		logStructuredError("plan_store_load_failed", map[string]interface{}{
			"error": err.Error(),
			"path":  plansPath,
		})
	}

	return &PlanService{planStore: planStore, defaultBundle: bundle, displayEnv: env}
}

// NewPlanServiceWithPlanStore creates a PlanService with an explicit PlanStore.
func NewPlanServiceWithPlanStore(planStore *PlanStore) *PlanService {
	bundle := stringsTrimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production")
	return &PlanService{planStore: planStore, defaultBundle: bundle, displayEnv: env}
}

// PlanServiceOptions allows explicit configuration for testing.
type PlanServiceOptions struct {
	PlanStore     *PlanStore
	DefaultBundle string
	DisplayEnv    string
}

// NewPlanServiceWithOptions creates a plan service with explicit configuration.
func NewPlanServiceWithOptions(opts PlanServiceOptions) *PlanService {
	bundle := opts.DefaultBundle
	if bundle == "" {
		bundle = stringsTrimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite")
	}
	env := opts.DisplayEnv
	if env == "" {
		env = stringsTrimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production")
	}
	planStore := opts.PlanStore
	if planStore == nil {
		plansPath := resolvePlansPath()
		planStore = NewPlanStoreWithOptions(PlanStoreOptions{
			PlansPath:  plansPath,
			BundleKey:  bundle,
			DisplayEnv: env,
		})
		_ = planStore.LoadAll()
	}
	return &PlanService{planStore: planStore, defaultBundle: bundle, displayEnv: env}
}

func stringsTrimOrDefault(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

// BundleKey returns the configured bundle key used for plan lookups.
func (s *PlanService) BundleKey() string {
	return s.defaultBundle
}

// GetPlanStore returns the underlying PlanStore for direct access.
func (s *PlanService) GetPlanStore() *PlanStore {
	return s.planStore
}

// GetPricingOverview loads the product and price rows for the default bundle.
func (s *PlanService) GetPricingOverview() (*PricingOverview, error) {
	return s.planStore.GetPricingOverview()
}

// GetPlanByPriceID fetches a plan option for a Stripe price identifier.
func (s *PlanService) GetPlanByPriceID(priceID string) (*PlanOption, error) {
	return s.planStore.GetPlanByPriceID(priceID)
}

// GetBundleProduct returns the configured bundle product metadata.
func (s *PlanService) GetBundleProduct() (*BundleProduct, error) {
	bundle := s.planStore.GetBundle()
	if bundle == nil {
		return nil, nil
	}
	return bundle, nil
}

// BundleCatalogEntry groups a bundle with all of its prices (visible + hidden).
type BundleCatalogEntry struct {
	Bundle *BundleProduct `json:"bundle"`
	Prices []*PlanOption  `json:"prices"`
}

// ListBundleCatalog returns bundles for the configured environment so the admin UI
// can toggle prices without raw SQL edits.
func (s *PlanService) ListBundleCatalog(ctx context.Context) ([]BundleCatalogEntry, error) {
	return s.planStore.ListBundleCatalog(ctx)
}

// UpdateBundlePriceInput contains editable fields for price display metadata.
type UpdateBundlePriceInput struct {
	StripePriceID  *string
	PlanName       *string
	DisplayWeight  *int
	DisplayEnabled *bool
	Subtitle       *string
	Badge          *string
	CtaLabel       *string
	Highlight      *bool
	Features       *[]string
}

// CreateBundlePriceInput contains fields for creating a new plan in the plan store.
type CreateBundlePriceInput struct {
	StripePriceID          string
	PlanName               string
	PlanTier               string
	BillingInterval        string
	AmountCents            *int64
	Currency               *string
	DisplayWeight          *int32
	DisplayEnabled         *bool
	MonthlyIncludedCredits *int64
	Subtitle               *string
	Badge                  *string
	CtaLabel               *string
	Highlight              *bool
	Features               []string
}

// UpdateBundlePrice applies display overrides for a Stripe price row.
func (s *PlanService) UpdateBundlePrice(ctx context.Context, bundleKey, priceID string, input UpdateBundlePriceInput) (*PlanOption, error) {
	if priceID == "" || bundleKey == "" {
		return nil, nil
	}
	return s.planStore.UpdatePlan(priceID, input)
}

// CreateBundlePrice creates a new plan in the plan store.
func (s *PlanService) CreateBundlePrice(ctx context.Context, bundleKey string, input CreateBundlePriceInput, fetcher StripePriceFetcher) (*PlanOption, error) {
	if strings.TrimSpace(bundleKey) == "" {
		return nil, fmt.Errorf("bundle_key is required")
	}
	if bundleKey != s.defaultBundle {
		return nil, fmt.Errorf("bundle key does not match active bundle")
	}
	if s.planStore == nil {
		return nil, fmt.Errorf("plan store not available")
	}

	priceID, err := normalizeStripePriceID(input.StripePriceID)
	if err != nil {
		return nil, err
	}
	planName, err := normalizePlanName(input.PlanName)
	if err != nil {
		return nil, err
	}
	planTier, err := normalizePlanTier(input.PlanTier)
	if err != nil {
		return nil, err
	}

	billingInterval := mapBillingInterval(input.BillingInterval)
	if err := validateBillingInterval(billingInterval); err != nil {
		return nil, err
	}

	var stripeDetails *StripePriceImport
	if fetcher != nil {
		details, err := fetcher(ctx, priceID)
		if err != nil {
			logStructured("stripe_price_not_verified", map[string]interface{}{
				"price_id": priceID,
				"error":    err.Error(),
			})
		} else {
			stripeDetails = details
		}
	}

	if stripeDetails != nil {
		if strings.TrimSpace(stripeDetails.PriceID) != priceID {
			return nil, fmt.Errorf("stripe price verification mismatch for %s", priceID)
		}
		bundle := s.planStore.GetBundle()
		if err := ensureStripePriceMatchesBundle(bundle, stripeDetails); err != nil {
			return nil, err
		}

		stripeInterval := mapBillingInterval(stripeDetails.Interval)
		if err := validateBillingInterval(stripeInterval); err != nil {
			return nil, fmt.Errorf("stripe price has unsupported billing interval")
		}
		if billingInterval != stripeInterval {
			return nil, fmt.Errorf("billing_interval does not match Stripe price")
		}
	}

	var amountCents int64
	if stripeDetails != nil {
		amountCents = stripeDetails.AmountCents
		if input.AmountCents != nil && *input.AmountCents != amountCents {
			return nil, fmt.Errorf("amount_cents does not match Stripe price")
		}
	} else {
		if input.AmountCents == nil {
			return nil, fmt.Errorf("amount_cents is required when Stripe price cannot be verified")
		}
		amountCents = *input.AmountCents
	}
	if amountCents < 0 {
		return nil, fmt.Errorf("amount_cents must be >= 0")
	}

	var currency string
	if stripeDetails != nil {
		currency = stripeDetails.Currency
		if input.Currency != nil && strings.TrimSpace(*input.Currency) != "" {
			if !strings.EqualFold(currency, *input.Currency) {
				return nil, fmt.Errorf("currency does not match Stripe price")
			}
		}
	} else if input.Currency != nil && strings.TrimSpace(*input.Currency) != "" {
		currency = *input.Currency
	} else {
		currency = "usd"
	}

	normalizedCurrency, err := normalizeCurrency(currency)
	if err != nil {
		return nil, err
	}
	currency = normalizedCurrency

	displayWeight := int32(10)
	if input.DisplayWeight != nil {
		displayWeight = *input.DisplayWeight
	}
	if displayWeight < 0 {
		return nil, fmt.Errorf("display_weight must be >= 0")
	}

	displayEnabled := true
	if input.DisplayEnabled != nil {
		displayEnabled = *input.DisplayEnabled
	} else if stripeDetails != nil {
		displayEnabled = stripeDetails.Active
	}

	monthlyCredits := int64(0)
	if input.MonthlyIncludedCredits != nil {
		monthlyCredits = *input.MonthlyIncludedCredits
	}
	if monthlyCredits < 0 {
		return nil, fmt.Errorf("monthly_included_credits must be >= 0")
	}

	metadata := buildPlanMetadata(
		input.Subtitle,
		input.Badge,
		input.CtaLabel,
		input.Highlight,
		input.Features,
	)

	plan := &PlanOption{
		StripePriceId:          priceID,
		PlanName:               planName,
		PlanTier:               planTier,
		BillingInterval:        billingInterval,
		AmountCents:            amountCents,
		Currency:               currency,
		DisplayWeight:          displayWeight,
		DisplayEnabled:         displayEnabled,
		MonthlyIncludedCredits: monthlyCredits,
		PlanRank:               planRankForTier(planTier),
		Kind:                   planKindForTier(planTier),
		BundleKey:              bundleKey,
	}

	if len(metadata) > 0 {
		plan.Metadata = metadata
	}

	created, err := s.planStore.AddPlan(plan)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// UpdateBundlePriceWithStripe updates a plan and optionally syncs Stripe price details when the price ID changes.
func (s *PlanService) UpdateBundlePriceWithStripe(ctx context.Context, bundleKey, priceID string, input UpdateBundlePriceInput, fetcher StripePriceFetcher) (*PlanOption, error) {
	if priceID == "" || bundleKey == "" {
		return nil, fmt.Errorf("bundle_key and price_id are required")
	}
	if bundleKey != s.defaultBundle {
		return nil, fmt.Errorf("bundle key does not match active bundle")
	}
	if s.planStore == nil {
		return nil, fmt.Errorf("plan store not available")
	}

	var stripeDetails *StripePriceImport
	if input.StripePriceID != nil {
		nextID := strings.TrimSpace(*input.StripePriceID)
		if nextID != "" {
			existing, err := s.planStore.GetPlanByPriceID(priceID)
			if err != nil {
				return nil, err
			}
			if existing != nil && nextID != existing.StripePriceId {
				if fetcher == nil {
					return nil, fmt.Errorf("stripe price changes require a Stripe verifier")
				}
				details, err := fetcher(ctx, nextID)
				if err != nil {
					return nil, err
				}
				stripeDetails = details
			}
		}
	}

	return s.planStore.UpdatePlanWithStripeDetails(priceID, input, stripeDetails)
}

// ImportStripePrices applies Stripe import selections and saves the plan catalog.
func (s *PlanService) ImportStripePrices(ctx context.Context, selections []ImportPlanSelection, fetcher StripePriceFetcher) (*StripeImportResult, error) {
	if s.planStore == nil {
		return nil, fmt.Errorf("plan store not available")
	}
	bundle := s.planStore.GetBundle()
	if bundle == nil {
		return nil, errStripeImportBundleMissing
	}
	bundleProductID := strings.TrimSpace(bundle.StripeProductId)
	return s.ImportStripePricesForProduct(ctx, selections, bundleProductID, StripeImportModeMerge, fetcher)
}

// ImportStripePricesForProduct imports selected Stripe prices for the specified bundle product.
// Use replace mode when switching products to clear existing plans first.
func (s *PlanService) ImportStripePricesForProduct(ctx context.Context, selections []ImportPlanSelection, bundleProductID string, mode StripeImportMode, fetcher StripePriceFetcher) (*StripeImportResult, error) {
	if s.planStore == nil {
		return nil, fmt.Errorf("plan store not available")
	}

	bundle := s.planStore.GetBundle()
	if bundle == nil {
		return nil, errStripeImportBundleMissing
	}

	normalizedProductID := strings.TrimSpace(bundleProductID)
	if normalizedProductID == "" {
		return nil, errStripeImportBundleProductMissing
	}

	switch mode {
	case StripeImportModeMerge, StripeImportModeReplace:
	default:
		return nil, errStripeImportInvalidMode
	}

	currentProductID := strings.TrimSpace(bundle.StripeProductId)
	if mode == StripeImportModeMerge && currentProductID != "" && currentProductID != normalizedProductID {
		return nil, errStripeImportProductSwitchRequiresReplace
	}

	bundle.StripeProductId = normalizedProductID

	normalizedSelections, _, err := normalizeStripeImportSelections(selections)
	if err != nil {
		return nil, err
	}

	var nextPlans []*PlanOption
	if mode == StripeImportModeReplace {
		nextPlans = nil
	} else {
		nextPlans = s.planStore.GetPlans()
	}

	if err := s.planStore.SetPlans(bundle, nextPlans); err != nil {
		return nil, err
	}

	return s.planStore.ApplyStripeImportSelections(ctx, normalizedSelections, fetcher)
}

// AddPlan adds a new plan to the store.
func (s *PlanService) AddPlan(plan *PlanOption) (*PlanOption, error) {
	return s.planStore.AddPlan(plan)
}

// DeletePlan removes a plan by its Stripe price ID.
func (s *PlanService) DeletePlan(priceID string) error {
	return s.planStore.DeletePlan(priceID)
}

// GetCouponMappings returns all coupon-to-plan mappings.
func (s *PlanService) GetCouponMappings() map[string]string {
	return s.planStore.GetCouponMappings()
}

// GetCouponForPlan returns the coupon ID assigned to a specific price.
func (s *PlanService) GetCouponForPlan(priceID string) string {
	return s.planStore.GetCouponForPlan(priceID)
}

// SetCouponForPlan assigns a coupon to a specific price.
func (s *PlanService) SetCouponForPlan(priceID, couponID string) error {
	return s.planStore.SetCouponForPlan(priceID, couponID)
}

// RemoveCouponFromPlan removes the coupon assignment from a specific price.
func (s *PlanService) RemoveCouponFromPlan(priceID string) error {
	return s.planStore.RemoveCouponFromPlan(priceID)
}

// ReloadPlans reloads plans from the JSON file.
func (s *PlanService) ReloadPlans() error {
	return s.planStore.LoadAll()
}

// toJsonValue converts a Go value to a commonv1.JsonValue.
func toJsonValue(v any) *commonv1.JsonValue {
	switch val := v.(type) {
	case nil:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		// JSON numbers are parsed as float64; check if it's a whole number
		if val == float64(int64(val)) {
			return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := make(map[string]*commonv1.JsonValue, len(val))
		for key, value := range val {
			if nested := toJsonValue(value); nested != nil {
				obj[key] = nested
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{
			ObjectValue: &commonv1.JsonObject{Fields: obj},
		}}
	case []any:
		items := make([]*commonv1.JsonValue, 0, len(val))
		for _, item := range val {
			if nested := toJsonValue(item); nested != nil {
				items = append(items, nested)
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{
			ListValue: &commonv1.JsonList{Values: items},
		}}
	default:
		return nil
	}
}

// jsonValueToAny converts a JsonValue to a Go any type.
func jsonValueToAny(v *commonv1.JsonValue) any {
	if v == nil {
		return nil
	}
	switch kind := v.Kind.(type) {
	case *commonv1.JsonValue_NullValue:
		return nil
	case *commonv1.JsonValue_BoolValue:
		return kind.BoolValue
	case *commonv1.JsonValue_IntValue:
		return kind.IntValue
	case *commonv1.JsonValue_DoubleValue:
		return kind.DoubleValue
	case *commonv1.JsonValue_StringValue:
		return kind.StringValue
	case *commonv1.JsonValue_BytesValue:
		return kind.BytesValue
	case *commonv1.JsonValue_ObjectValue:
		if kind.ObjectValue == nil {
			return nil
		}
		result := make(map[string]any, len(kind.ObjectValue.Fields))
		for k, fv := range kind.ObjectValue.Fields {
			result[k] = jsonValueToAny(fv)
		}
		return result
	case *commonv1.JsonValue_ListValue:
		if kind.ListValue == nil {
			return nil
		}
		result := make([]any, 0, len(kind.ListValue.Values))
		for _, item := range kind.ListValue.Values {
			result = append(result, jsonValueToAny(item))
		}
		return result
	default:
		return nil
	}
}

// newStringJsonValue creates a JsonValue with a string.
func newStringJsonValue(s string) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: s}}
}

// newBoolJsonValue creates a JsonValue with a bool.
func newBoolJsonValue(b bool) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: b}}
}

// newListJsonValue creates a JsonValue with a list of JsonValues.
func newListJsonValue(values []*commonv1.JsonValue) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{
		ListValue: &commonv1.JsonList{Values: values},
	}}
}

func buildPlanMetadata(subtitle, badge, ctaLabel *string, highlight *bool, features []string) map[string]*commonv1.JsonValue {
	metadata := make(map[string]*commonv1.JsonValue)

	if subtitle != nil {
		if trimmed := strings.TrimSpace(*subtitle); trimmed != "" {
			metadata["subtitle"] = newStringJsonValue(trimmed)
		}
	}
	if badge != nil {
		if trimmed := strings.TrimSpace(*badge); trimmed != "" {
			metadata["badge"] = newStringJsonValue(trimmed)
		}
	}
	if ctaLabel != nil {
		if trimmed := strings.TrimSpace(*ctaLabel); trimmed != "" {
			metadata["cta_label"] = newStringJsonValue(trimmed)
		}
	}
	if highlight != nil && *highlight {
		metadata["highlight"] = newBoolJsonValue(true)
	}

	var sanitized []string
	for _, feature := range features {
		if trimmed := strings.TrimSpace(feature); trimmed != "" {
			sanitized = append(sanitized, trimmed)
		}
	}
	if len(sanitized) > 0 {
		listValues := make([]*commonv1.JsonValue, 0, len(sanitized))
		for _, feature := range sanitized {
			listValues = append(listValues, newStringJsonValue(feature))
		}
		metadata["features"] = newListJsonValue(listValues)
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func ensureStripePriceMatchesBundle(bundle *BundleProduct, stripeDetails *StripePriceImport) error {
	if stripeDetails == nil {
		return nil
	}
	if bundle == nil {
		return fmt.Errorf("bundle not configured")
	}

	bundleProductID := strings.TrimSpace(bundle.StripeProductId)
	if bundleProductID == "" {
		return fmt.Errorf("bundle stripe_product_id is required")
	}

	priceProductID := strings.TrimSpace(stripeDetails.ProductID)
	if priceProductID == "" {
		return fmt.Errorf("stripe price %s missing product_id", strings.TrimSpace(stripeDetails.PriceID))
	}

	if priceProductID != bundleProductID {
		return fmt.Errorf("stripe price %s belongs to product %s (expected %s)", stripeDetails.PriceID, priceProductID, bundleProductID)
	}

	return nil
}

func mapPlanKind(kind string) landing_page_react_vite_v1.PlanKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "subscription":
		return landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUBSCRIPTION
	case "credits_topup", "credits-topup", "credits":
		return landing_page_react_vite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP
	case "supporter_contribution", "supporter-contribution", "supporter":
		return landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION
	default:
		return landing_page_react_vite_v1.PlanKind_PLAN_KIND_UNSPECIFIED
	}
}

var allowedPlanTiers = map[string]int32{
	"free":     0,
	"solo":     1,
	"pro":      2,
	"studio":   3,
	"business": 4,
	"credits":  5,
	"donation": 6,
}

func normalizePlanTier(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", fmt.Errorf("plan_tier is required")
	}
	if _, ok := allowedPlanTiers[value]; !ok {
		return "", fmt.Errorf("unsupported plan_tier: %s", value)
	}
	return value, nil
}

func planKindForTier(tier string) landing_page_react_vite_v1.PlanKind {
	switch tier {
	case "credits":
		return landing_page_react_vite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP
	case "donation":
		return landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION
	default:
		return landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUBSCRIPTION
	}
}

func derivePlanTierFromStripe(price *StripePriceImport) (string, bool) {
	if price == nil {
		return "", false
	}
	candidates := []string{
		price.LookupKey,
		price.ProductName,
		price.PriceID,
	}

	for _, candidate := range candidates {
		if tier, ok := detectTierToken(candidate); ok {
			return tier, true
		}
	}

	return "", false
}

func detectTierToken(source string) (string, bool) {
	normalized := strings.ToLower(source)
	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if _, ok := allowedPlanTiers[token]; ok {
			return token, true
		}
	}
	return "", false
}

func planNameFromStripeImport(price *StripePriceImport) string {
	if price == nil {
		return ""
	}
	name := strings.TrimSpace(price.ProductName)
	if name == "" {
		name = strings.TrimSpace(price.LookupKey)
	}
	if name == "" {
		name = strings.TrimSpace(price.PriceID)
	}
	return name
}

// stripePriceImportToPlanOption converts a StripePriceImport to a PlanOption.
func stripePriceImportToPlanOption(price *StripePriceImport) *PlanOption {
	interval := mapBillingInterval(price.Interval)
	if interval == landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_UNSPECIFIED {
		interval = landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_ONE_TIME
	}

	planTier, ok := derivePlanTierFromStripe(price)
	if !ok {
		planTier = "pro"
	}
	planName := planNameFromStripeImport(price)

	return &PlanOption{
		StripePriceId:   price.PriceID,
		PlanName:        planName,
		PlanTier:        planTier,
		BillingInterval: interval,
		AmountCents:     price.AmountCents,
		Currency:        price.Currency,
		DisplayEnabled:  price.Active,
		DisplayWeight:   10,
		PlanRank:        planRankForTier(planTier),
		Kind:            planKindForTier(planTier),
	}
}

func normalizeCurrency(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", fmt.Errorf("currency is required")
	}
	if len(value) != 3 {
		return "", fmt.Errorf("currency must be a 3-letter ISO code")
	}
	for _, r := range value {
		if !unicode.IsLetter(r) {
			return "", fmt.Errorf("currency must only contain letters")
		}
	}
	return value, nil
}

func normalizePlanName(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("plan_name is required")
	}
	return value, nil
}

func normalizeStripePriceID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("stripe_price_id is required")
	}
	if !strings.HasPrefix(value, "price_") {
		return "", fmt.Errorf("stripe_price_id must start with price_")
	}
	return value, nil
}

func validateBillingInterval(interval landing_page_react_vite_v1.BillingInterval) error {
	if interval == landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_UNSPECIFIED {
		return fmt.Errorf("billing_interval is required")
	}
	return nil
}

func planRankForTier(tier string) int32 {
	if rank, ok := allowedPlanTiers[tier]; ok {
		return rank
	}
	return 0
}

//nolint:unused // helper retained for future optional fields
func nullableString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func planKindString(kind landing_page_react_vite_v1.PlanKind) string {
	switch kind {
	case landing_page_react_vite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP:
		return "credits_topup"
	case landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION:
		return "supporter_contribution"
	case landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUBSCRIPTION:
		return "subscription"
	default:
		return "subscription"
	}
}

func mapBillingInterval(raw string) landing_page_react_vite_v1.BillingInterval {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "month", "monthly", "m":
		return landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH
	case "year", "yearly", "y":
		return landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_YEAR
	case "one_time", "one-time", "one time", "onetime", "ot":
		return landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_ONE_TIME
	default:
		return landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_UNSPECIFIED
	}
}

func billingIntervalLabel(interval landing_page_react_vite_v1.BillingInterval) string {
	switch interval {
	case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH:
		return "month"
	case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_YEAR:
		return "year"
	case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_ONE_TIME:
		return "one_time"
	default:
		return "unspecified"
	}
}
