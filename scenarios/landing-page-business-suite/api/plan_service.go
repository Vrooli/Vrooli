package main

import (
	"context"
	"fmt"
	"strings"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
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
	BundleProduct   = shared.Bundle
	PlanOption      = shared.PlanOption
	PricingOverview = shared.PricingOverview
)

type (
	StripePriceFetcher     = commerce.StripePriceFetcher
	ImportPlanSelection    = commerce.ImportPlanSelection
	StripeImportMode       = commerce.StripeImportMode
	StripeImportResult     = commerce.StripeImportResult
	BundleCatalogEntry     = commerce.BundleCatalogEntry
	UpdateBundlePriceInput = commerce.UpdateBundlePriceInput
	CreateBundlePriceInput = commerce.CreateBundlePriceInput
)

const (
	StripeImportModeMerge   = commerce.StripeImportModeMerge
	StripeImportModeReplace = commerce.StripeImportModeReplace
)

var (
	errStripeImportNoSelections                 = commerce.ErrStripeImportNoSelections
	errStripeImportNoValidSelections            = commerce.ErrStripeImportNoValidSelections
	errStripeImportMissingFetcher               = commerce.ErrStripeImportMissingFetcher
	errStripeImportBundleMissing                = commerce.ErrStripeImportBundleMissing
	errStripeImportBundleProductMissing         = commerce.ErrStripeImportBundleProductMissing
	errStripeImportInvalidMode                  = commerce.ErrStripeImportInvalidMode
	errStripeImportProductSwitchRequiresReplace = commerce.ErrStripeImportProductSwitchRequiresReplace
)

// NewPlanService creates a PlanService with a PlanStore. Plans are file-backed;
// the ignored legacy argument keeps existing callers source-compatible while
// ensuring this service never captures a database pool.
func NewPlanService(_ any) *PlanService {
	bundle := stringsTrimOrDefault(envx.Get("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(envx.Get("BUNDLE_ENVIRONMENT"), "production")

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
	bundle := stringsTrimOrDefault(envx.Get("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(envx.Get("BUNDLE_ENVIRONMENT"), "production")
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
		bundle = stringsTrimOrDefault(envx.Get("BUNDLE_KEY"), "business_suite")
	}
	env := opts.DisplayEnv
	if env == "" {
		env = stringsTrimOrDefault(envx.Get("BUNDLE_ENVIRONMENT"), "production")
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

// ListBundleCatalog returns bundles for the configured environment so the admin UI
// can toggle prices without raw SQL edits.
func (s *PlanService) ListBundleCatalog(ctx context.Context) ([]BundleCatalogEntry, error) {
	return s.planStore.ListBundleCatalog(ctx)
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

	priceID, err := commerce.NormalizeStripePriceID(input.StripePriceID)
	if err != nil {
		return nil, err
	}
	planName, err := commerce.NormalizePlanName(input.PlanName)
	if err != nil {
		return nil, err
	}
	planTier, err := commerce.NormalizePlanTier(input.PlanTier)
	if err != nil {
		return nil, err
	}

	billingInterval := commerce.MapBillingInterval(input.BillingInterval)
	if err := commerce.ValidateBillingInterval(billingInterval); err != nil {
		return nil, err
	}

	stripeDetails := fetchStripePriceForCreation(ctx, priceID, fetcher)
	pricing, err := resolveCreatedPlanPricing(input, priceID, billingInterval, stripeDetails, s.planStore.GetBundle())
	if err != nil {
		return nil, err
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
		AmountCents:            pricing.amountCents,
		Currency:               pricing.currency,
		DisplayWeight:          pricing.displayWeight,
		DisplayEnabled:         pricing.displayEnabled,
		MonthlyIncludedCredits: pricing.monthlyCredits,
		PlanRank:               commerce.PlanRankForTier(planTier),
		Kind:                   commerce.PlanKindForTier(planTier),
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

	var stripeDetails *commerce.StripePriceImport
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

	normalizedSelections, _, err := commerce.NormalizeStripeImportSelections(selections)
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
