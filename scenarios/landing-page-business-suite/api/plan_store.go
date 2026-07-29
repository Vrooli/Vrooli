package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"landing-page-business-suite-api/internal/envx"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// PlanStoreReader provides read-only access to plan configuration.
type PlanStoreReader interface {
	GetBundle() *BundleProduct
	GetPlans() []*PlanOption
	GetPlanByPriceID(priceID string) (*PlanOption, error)
	GetPricingOverview() (*PricingOverview, error)
	ListBundleCatalog(ctx context.Context) ([]BundleCatalogEntry, error)
}

// PlanStoreWriter provides write access to plan configuration.
type PlanStoreWriter interface {
	LoadAll() error
	SavePlans() error
	AddPlan(plan *PlanOption) (*PlanOption, error)
	UpdatePlan(priceID string, input UpdateBundlePriceInput) (*PlanOption, error)
	DeletePlan(priceID string) error
	SetPlans(bundle *BundleProduct, plans []*PlanOption) error
}

// PlanStorer combines read and write access to plan configuration.
type PlanStorer interface {
	PlanStoreReader
	PlanStoreWriter
}

// Compile-time check that PlanStore implements PlanStorer
var _ PlanStorer = (*PlanStore)(nil)

// PlanStore provides in-memory caching of plan configuration
// loaded from JSON files. It serves as the single source of truth for plans,
// replacing the previous database-backed plan storage.
type PlanStore struct {
	mu             sync.RWMutex
	bundle         *BundleProduct
	plans          []*PlanOption
	couponMappings map[string]string // priceID -> couponID
	plansPath      string
	displayEnv     string
	bundleKey      string
	updatedAt      time.Time
}

var (
	errStripeImportNoSelections                 = errors.New("no selections provided")
	errStripeImportNoValidSelections            = errors.New("no valid selections provided")
	errStripeImportMissingFetcher               = errors.New("stripe price fetcher required")
	errStripeImportBundleMissing                = errors.New("bundle not configured")
	errStripeImportBundleProductMissing         = errors.New("bundle_product_id is required")
	errStripeImportInvalidMode                  = errors.New("import mode must be merge or replace")
	errStripeImportProductSwitchRequiresReplace = errors.New("bundle product change requires replace mode")
)

// plansFileFormat represents the JSON file structure for plans.
type plansFileFormat struct {
	Bundle         bundleFileFormat  `json:"bundle"`
	Plans          []planFileFormat  `json:"plans"`
	CouponMappings map[string]string `json:"coupon_mappings,omitempty"` // priceID -> couponID
	UpdatedAt      string            `json:"updated_at,omitempty"`
}

type bundleFileFormat struct {
	BundleKey                string                 `json:"bundle_key"`
	Name                     string                 `json:"name"`
	StripeProductID          string                 `json:"stripe_product_id"`
	CreditsPerUSD            int64                  `json:"credits_per_usd"`
	DisplayCreditsMultiplier float64                `json:"display_credits_multiplier"`
	DisplayCreditsLabel      string                 `json:"display_credits_label"`
	Environment              string                 `json:"environment"`
	Metadata                 map[string]interface{} `json:"metadata,omitempty"`
}

type planFileFormat struct {
	StripePriceID          string                 `json:"stripe_price_id"`
	PlanName               string                 `json:"plan_name"`
	PlanTier               string                 `json:"plan_tier"`
	BillingInterval        string                 `json:"billing_interval"`
	AmountCents            int64                  `json:"amount_cents"`
	Currency               string                 `json:"currency"`
	DisplayWeight          int32                  `json:"display_weight"`
	DisplayEnabled         bool                   `json:"display_enabled"`
	MonthlyIncludedCredits int64                  `json:"monthly_included_credits"`
	OneTimeBonusCredits    int64                  `json:"one_time_bonus_credits"`
	PlanRank               int32                  `json:"plan_rank"`
	BonusType              string                 `json:"bonus_type"`
	Kind                   string                 `json:"kind"`
	IntroEnabled           bool                   `json:"intro_enabled"`
	IntroType              string                 `json:"intro_type,omitempty"`
	IntroAmountCents       *int64                 `json:"intro_amount_cents,omitempty"`
	IntroPeriods           int32                  `json:"intro_periods,omitempty"`
	IntroPriceLookupKey    string                 `json:"intro_price_lookup_key,omitempty"`
	IsVariableAmount       bool                   `json:"is_variable_amount"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

// NewPlanStore creates a new PlanStore with path to the JSON file.
func NewPlanStore(plansPath string) *PlanStore {
	bundleKey := stringsTrimOrDefault(envx.Get("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(envx.Get("BUNDLE_ENVIRONMENT"), "production")
	return &PlanStore{
		plans:          make([]*PlanOption, 0),
		couponMappings: make(map[string]string),
		plansPath:      plansPath,
		displayEnv:     env,
		bundleKey:      bundleKey,
	}
}

// NewPlanStoreWithOptions creates a plan store with explicit configuration.
type PlanStoreOptions struct {
	PlansPath  string
	BundleKey  string
	DisplayEnv string
}

func NewPlanStoreWithOptions(opts PlanStoreOptions) *PlanStore {
	bundleKey := opts.BundleKey
	if bundleKey == "" {
		bundleKey = stringsTrimOrDefault(envx.Get("BUNDLE_KEY"), "business_suite")
	}
	env := opts.DisplayEnv
	if env == "" {
		env = stringsTrimOrDefault(envx.Get("BUNDLE_ENVIRONMENT"), "production")
	}
	return &PlanStore{
		plans:          make([]*PlanOption, 0),
		couponMappings: make(map[string]string),
		plansPath:      opts.PlansPath,
		displayEnv:     env,
		bundleKey:      bundleKey,
	}
}

// BundleKey returns the configured bundle key.
func (ps *PlanStore) BundleKey() string {
	return ps.bundleKey
}

func (ps *PlanStore) validatePlanCatalogLocked() error {
	if ps.bundle == nil {
		return fmt.Errorf("bundle not configured")
	}
	if err := normalizeBundleProduct(ps.bundle, ps.bundleKey, ps.displayEnv); err != nil {
		return fmt.Errorf("invalid bundle: %w", err)
	}

	seenPriceIDs := make(map[string]struct{}, len(ps.plans))
	for _, plan := range ps.plans {
		if plan == nil {
			return fmt.Errorf("plan is required")
		}
		if err := normalizePlanOption(plan, ps.bundleKey); err != nil {
			return fmt.Errorf("invalid plan %s: %w", strings.TrimSpace(plan.StripePriceId), err)
		}
		if _, exists := seenPriceIDs[plan.StripePriceId]; exists {
			return fmt.Errorf("duplicate stripe_price_id detected: %s", plan.StripePriceId)
		}
		seenPriceIDs[plan.StripePriceId] = struct{}{}
	}
	return nil
}

// GetBundle returns the bundle configuration.
func (ps *PlanStore) GetBundle() *BundleProduct {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.bundle == nil {
		return nil
	}
	return proto.Clone(ps.bundle).(*BundleProduct)
}

// GetPlans returns all plans.
func (ps *PlanStore) GetPlans() []*PlanOption {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make([]*PlanOption, 0, len(ps.plans))
	for _, plan := range ps.plans {
		result = append(result, proto.Clone(plan).(*PlanOption))
	}
	return result
}

// GetPlanByPriceID fetches a plan by its Stripe price ID.
func (ps *PlanStore) GetPlanByPriceID(priceID string) (*PlanOption, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	priceID = strings.TrimSpace(priceID)
	if priceID == "" {
		return nil, fmt.Errorf("price id is required")
	}

	for _, plan := range ps.plans {
		if plan.StripePriceId == priceID {
			return proto.Clone(plan).(*PlanOption), nil
		}
	}

	return nil, fmt.Errorf("price %s not found", priceID)
}

// GetCouponMappings returns all coupon-to-plan mappings.
func (ps *PlanStore) GetCouponMappings() map[string]string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make(map[string]string, len(ps.couponMappings))
	for priceID, couponID := range ps.couponMappings {
		result[priceID] = couponID
	}
	return result
}

// GetCouponForPlan returns the coupon ID assigned to a specific price, or empty string if none.
func (ps *PlanStore) GetCouponForPlan(priceID string) string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	priceID = strings.TrimSpace(priceID)
	if priceID == "" {
		return ""
	}
	return ps.couponMappings[priceID]
}

// SetCouponForPlan assigns a coupon to a specific price. Returns error if price doesn't exist.
func (ps *PlanStore) SetCouponForPlan(priceID, couponID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	priceID = strings.TrimSpace(priceID)
	couponID = strings.TrimSpace(couponID)

	if priceID == "" {
		return fmt.Errorf("price id is required")
	}
	if couponID == "" {
		return fmt.Errorf("coupon id is required")
	}

	// Verify the price exists
	found := false
	for _, plan := range ps.plans {
		if plan.StripePriceId == priceID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("price %s not found", priceID)
	}

	if ps.couponMappings == nil {
		ps.couponMappings = make(map[string]string)
	}
	ps.couponMappings[priceID] = couponID

	return ps.savePlansLocked()
}

// RemoveCouponFromPlan removes the coupon assignment from a specific price.
func (ps *PlanStore) RemoveCouponFromPlan(priceID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	priceID = strings.TrimSpace(priceID)
	if priceID == "" {
		return fmt.Errorf("price id is required")
	}

	if ps.couponMappings == nil {
		return nil // Nothing to remove
	}

	delete(ps.couponMappings, priceID)
	return ps.savePlansLocked()
}

// GetPricingOverview returns a complete pricing overview for the frontend.
func (ps *PlanStore) GetPricingOverview() (*PricingOverview, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.bundle == nil {
		return nil, fmt.Errorf("bundle not configured")
	}

	monthly := make([]*PlanOption, 0)
	yearly := make([]*PlanOption, 0)

	for _, plan := range ps.plans {
		if !plan.DisplayEnabled && strings.ToLower(strings.TrimSpace(plan.PlanTier)) != "free" {
			continue
		}

		switch plan.BillingInterval {
		case shared.BillingInterval_BILLING_INTERVAL_MONTH:
			monthly = append(monthly, proto.Clone(plan).(*PlanOption))
		case shared.BillingInterval_BILLING_INTERVAL_YEAR:
			yearly = append(yearly, proto.Clone(plan).(*PlanOption))
		}
	}

	// Sort by display weight (descending) then plan rank (ascending)
	sortPlans := func(plans []*PlanOption) {
		sort.SliceStable(plans, func(i, j int) bool {
			if plans[i].DisplayWeight == plans[j].DisplayWeight {
				return plans[i].PlanRank < plans[j].PlanRank
			}
			return plans[i].DisplayWeight > plans[j].DisplayWeight
		})
	}

	sortPlans(monthly)
	sortPlans(yearly)

	return &PricingOverview{
		Bundle:    proto.Clone(ps.bundle).(*BundleProduct),
		Monthly:   monthly,
		Yearly:    yearly,
		UpdatedAt: timestamppb.Now(),
	}, nil
}

// ListBundleCatalog returns all bundles with their prices for admin UI.
func (ps *PlanStore) ListBundleCatalog(ctx context.Context) ([]BundleCatalogEntry, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.bundle == nil {
		return []BundleCatalogEntry{}, nil
	}

	prices := make([]*PlanOption, 0, len(ps.plans))
	for _, plan := range ps.plans {
		prices = append(prices, proto.Clone(plan).(*PlanOption))
	}

	// Sort by display weight and plan rank
	sort.SliceStable(prices, func(i, j int) bool {
		if prices[i].DisplayWeight == prices[j].DisplayWeight {
			return prices[i].PlanRank < prices[j].PlanRank
		}
		return prices[i].DisplayWeight > prices[j].DisplayWeight
	})

	return []BundleCatalogEntry{
		{
			Bundle: proto.Clone(ps.bundle).(*BundleProduct),
			Prices: prices,
		},
	}, nil
}

// AddPlan adds a new plan to the store.
func (ps *PlanStore) AddPlan(plan *PlanOption) (*PlanOption, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if plan == nil {
		return nil, fmt.Errorf("plan is required")
	}

	cloned := proto.Clone(plan).(*PlanOption)
	if err := normalizePlanOption(cloned, ps.bundleKey); err != nil {
		return nil, err
	}

	if ps.bundle == nil {
		return nil, fmt.Errorf("bundle not configured")
	}

	// Check for duplicate
	for _, existing := range ps.plans {
		if existing.StripePriceId == cloned.StripePriceId {
			return nil, fmt.Errorf("plan with price ID %s already exists", cloned.StripePriceId)
		}
	}

	ps.plans = append(ps.plans, cloned)
	if err := ps.savePlansLocked(); err != nil {
		return nil, err
	}
	return proto.Clone(cloned).(*PlanOption), nil
}

func normalizeStripeImportSelections(selections []ImportPlanSelection) ([]ImportPlanSelection, []string, error) {
	if len(selections) == 0 {
		return nil, nil, errStripeImportNoSelections
	}

	normalizedSelections := make([]ImportPlanSelection, 0, len(selections))
	seenSelections := make(map[string]struct{}, len(selections))
	var errorsList []string

	for _, selection := range selections {
		priceID := strings.TrimSpace(selection.PriceID)
		if priceID == "" {
			errorsList = append(errorsList, "empty price ID in selection")
			continue
		}
		if _, err := normalizeStripePriceID(priceID); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("invalid price id %s: %s", priceID, err.Error()))
			continue
		}
		action := strings.ToLower(strings.TrimSpace(selection.Action))
		switch action {
		case "import", "overwrite", "skip":
		default:
			errorsList = append(errorsList, "unknown action: "+selection.Action)
			continue
		}
		if _, exists := seenSelections[priceID]; exists {
			errorsList = append(errorsList, "duplicate selection for price "+priceID)
			continue
		}
		seenSelections[priceID] = struct{}{}
		normalizedSelections = append(normalizedSelections, ImportPlanSelection{
			PriceID: priceID,
			Action:  action,
		})
	}

	if len(normalizedSelections) == 0 {
		return nil, errorsList, errStripeImportNoValidSelections
	}

	return normalizedSelections, errorsList, nil
}

// ApplyStripeImportSelections batches Stripe import updates into the plan store with a single save.
func (ps *PlanStore) ApplyStripeImportSelections(ctx context.Context, selections []ImportPlanSelection, fetcher StripePriceFetcher) (*StripeImportResult, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	result := &StripeImportResult{}
	if fetcher == nil {
		return result, errStripeImportMissingFetcher
	}
	if ps.bundle == nil {
		return result, errStripeImportBundleMissing
	}

	normalizedSelections, errorsList, err := normalizeStripeImportSelections(selections)
	if len(errorsList) > 0 {
		result.Errors = append(result.Errors, errorsList...)
	}
	if err != nil {
		return result, err
	}

	planIndex := make(map[string]int, len(ps.plans))
	for i, plan := range ps.plans {
		planIndex[plan.StripePriceId] = i
	}

	changed := false

	for _, selection := range normalizedSelections {
		switch selection.Action {
		case "skip":
			result.Skipped++
			continue
		case "import", "overwrite":
			priceDetails, err := fetcher(ctx, selection.PriceID)
			if err != nil {
				result.Errors = append(result.Errors, "failed to fetch price "+selection.PriceID+": "+err.Error())
				continue
			}
			if priceDetails == nil || strings.TrimSpace(priceDetails.PriceID) == "" {
				result.Errors = append(result.Errors, "stripe price details missing for "+selection.PriceID)
				continue
			}
			if priceDetails.PriceID != selection.PriceID {
				result.Errors = append(result.Errors, fmt.Sprintf("stripe price mismatch for %s (got %s)", selection.PriceID, priceDetails.PriceID))
				continue
			}
			if err := ps.ensureStripePriceMatchesBundleLocked(priceDetails); err != nil {
				result.Errors = append(result.Errors, "stripe price "+selection.PriceID+" rejected: "+err.Error())
				continue
			}

			if idx, exists := planIndex[selection.PriceID]; exists {
				if selection.Action == "overwrite" {
					name := planNameFromStripeImport(priceDetails)
					displayEnabled := priceDetails.Active
					input := UpdateBundlePriceInput{
						PlanName:       &name,
						DisplayEnabled: &displayEnabled,
					}
					derivedTier, ok := derivePlanTierFromStripe(priceDetails)
					if !ok {
						derivedTier = ""
					}
					if _, err := ps.updatePlanWithStripeDetailsLocked(selection.PriceID, input, priceDetails, derivedTier); err != nil {
						result.Errors = append(result.Errors, "failed to update "+selection.PriceID+": "+err.Error())
						continue
					}
					changed = true
					result.Overwritten++
				} else {
					result.Skipped++
				}
				planIndex[selection.PriceID] = idx
				continue
			}

			plan := stripePriceImportToPlanOption(priceDetails)
			if err := normalizePlanOption(plan, ps.bundleKey); err != nil {
				result.Errors = append(result.Errors, "failed to normalize "+selection.PriceID+": "+err.Error())
				continue
			}
			ps.plans = append(ps.plans, plan)
			planIndex[plan.StripePriceId] = len(ps.plans) - 1
			changed = true
			result.Imported++
		}
	}

	if changed {
		if err := ps.savePlansLocked(); err != nil {
			return result, err
		}
	}

	return result, nil
}

// DeletePlan removes a plan by its Stripe price ID.
func (ps *PlanStore) DeletePlan(priceID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	priceID = strings.TrimSpace(priceID)
	if priceID == "" {
		return fmt.Errorf("price id is required")
	}

	idx := -1
	for i, plan := range ps.plans {
		if plan.StripePriceId == priceID {
			idx = i
			break
		}
	}

	if idx < 0 {
		return fmt.Errorf("price %s not found", priceID)
	}

	ps.plans = append(ps.plans[:idx], ps.plans[idx+1:]...)
	return ps.savePlansLocked()
}

// SetPlans replaces all plans with the provided list (used for Stripe import).
func (ps *PlanStore) SetPlans(bundle *BundleProduct, plans []*PlanOption) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if bundle != nil {
		clonedBundle := proto.Clone(bundle).(*BundleProduct)
		if err := normalizeBundleProduct(clonedBundle, ps.bundleKey, ps.displayEnv); err != nil {
			return err
		}
		ps.bundle = clonedBundle
	}

	nextPlans := make([]*PlanOption, 0, len(plans))
	seenPriceIDs := make(map[string]struct{}, len(plans))
	for _, plan := range plans {
		cloned := proto.Clone(plan).(*PlanOption)
		if err := normalizePlanOption(cloned, ps.bundleKey); err != nil {
			return err
		}
		if _, exists := seenPriceIDs[cloned.StripePriceId]; exists {
			return fmt.Errorf("duplicate stripe_price_id detected: %s", cloned.StripePriceId)
		}
		seenPriceIDs[cloned.StripePriceId] = struct{}{}
		nextPlans = append(nextPlans, cloned)
	}

	ps.plans = nextPlans
	return ps.savePlansLocked()
}

// Helper functions for metadata conversion

func convertMetadataToProto(m map[string]interface{}) map[string]*commonv1.JsonValue {
	if m == nil {
		return nil
	}
	result := make(map[string]*commonv1.JsonValue, len(m))
	for key, value := range m {
		if jv := toJsonValue(value); jv != nil {
			result[key] = jv
		}
	}
	return result
}

func convertProtoMetadataToMap(m map[string]*commonv1.JsonValue) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = jsonValueToAny(v)
	}
	return result
}

func mapIntroPricingTypeFromString(s string) shared.IntroPricingType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "percentage", "percent", "pct":
		return shared.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE
	case "flat_amount", "flat-amount", "flat", "amount":
		return shared.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT
	default:
		return shared.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED
	}
}

func introPricingTypeString(t shared.IntroPricingType) string {
	switch t {
	case shared.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE:
		return "percentage"
	case shared.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT:
		return "flat_amount"
	default:
		return ""
	}
}

// resolvePlansPath finds the plans.json file.
func resolvePlansPath() string {
	candidates := []string{
		filepath.Join("..", ".vrooli", "plans.json"),
		filepath.Join(".", ".vrooli", "plans.json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", ".vrooli", "plans.json")
}

func normalizeBundleProduct(bundle *BundleProduct, bundleKeyFallback, envFallback string) error {
	if bundle == nil {
		return fmt.Errorf("bundle is required")
	}

	bundle.BundleKey = strings.TrimSpace(bundle.BundleKey)
	if bundle.BundleKey == "" {
		bundle.BundleKey = strings.TrimSpace(bundleKeyFallback)
	}
	if bundle.BundleKey == "" {
		return fmt.Errorf("bundle_key is required")
	}
	if bundleKeyFallback != "" && bundle.BundleKey != bundleKeyFallback {
		return fmt.Errorf("bundle_key mismatch: expected %s", bundleKeyFallback)
	}

	bundle.Name = strings.TrimSpace(bundle.Name)
	if bundle.Name == "" {
		return fmt.Errorf("bundle name is required")
	}

	bundle.StripeProductId = strings.TrimSpace(bundle.StripeProductId)
	if bundle.StripeProductId == "" {
		return fmt.Errorf("stripe_product_id is required")
	}

	if bundle.CreditsPerUsd <= 0 {
		return fmt.Errorf("credits_per_usd must be > 0")
	}

	if bundle.DisplayCreditsMultiplier <= 0 {
		bundle.DisplayCreditsMultiplier = 1
	}
	if strings.TrimSpace(bundle.DisplayCreditsLabel) == "" {
		bundle.DisplayCreditsLabel = "credits"
	}

	if strings.TrimSpace(bundle.Environment) == "" {
		bundle.Environment = strings.TrimSpace(envFallback)
		if bundle.Environment == "" {
			bundle.Environment = "production"
		}
	}

	return nil
}

func normalizePlanOption(plan *PlanOption, bundleKey string) error {
	if plan == nil {
		return fmt.Errorf("plan is required")
	}

	priceID, err := normalizeStripePriceID(plan.StripePriceId)
	if err != nil {
		return err
	}
	plan.StripePriceId = priceID

	name, err := normalizePlanName(plan.PlanName)
	if err != nil {
		return err
	}
	plan.PlanName = name

	tier, err := normalizePlanTier(plan.PlanTier)
	if err != nil {
		return err
	}
	plan.PlanTier = tier

	if err := validateBillingInterval(plan.BillingInterval); err != nil {
		return err
	}

	currency, err := normalizeCurrency(plan.Currency)
	if err != nil {
		return err
	}
	plan.Currency = currency

	if plan.AmountCents < 0 {
		return fmt.Errorf("amount_cents must be >= 0")
	}
	if plan.MonthlyIncludedCredits < 0 {
		return fmt.Errorf("monthly_included_credits must be >= 0")
	}
	if plan.OneTimeBonusCredits < 0 {
		return fmt.Errorf("one_time_bonus_credits must be >= 0")
	}
	if plan.PlanRank < 0 {
		return fmt.Errorf("plan_rank must be >= 0")
	}
	if plan.DisplayWeight < 0 {
		return fmt.Errorf("display_weight must be >= 0")
	}

	expectedKind := planKindForTier(plan.PlanTier)
	if plan.Kind == shared.PlanKind_PLAN_KIND_UNSPECIFIED {
		plan.Kind = expectedKind
	} else if plan.Kind != expectedKind {
		return fmt.Errorf("plan_kind %s does not match plan_tier %s", planKindString(plan.Kind), plan.PlanTier)
	}

	plan.BundleKey = bundleKey
	if err := validatePlanTierConstraints(plan); err != nil {
		return err
	}

	return nil
}

func validatePlanTierConstraints(plan *PlanOption) error {
	if plan == nil {
		return nil
	}

	switch plan.PlanTier {
	case "free":
		if plan.AmountCents != 0 {
			return fmt.Errorf("free plan amount_cents must be 0")
		}
	case "credits", "donation":
		if plan.BillingInterval != shared.BillingInterval_BILLING_INTERVAL_ONE_TIME {
			return fmt.Errorf("%s plans must use one_time billing_interval", plan.PlanTier)
		}
	}

	return nil
}
