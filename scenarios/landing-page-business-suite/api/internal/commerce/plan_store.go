package commerce

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"landing-page-business-suite-api/internal/envx"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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

// seam: PlanStorer combines read and write access to plan configuration.
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
	log            func(event string, fields map[string]interface{})
}

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

// PlansFileFormat and its component aliases are intentionally exposed only
// within the API's internal package boundary. They support black-box callers
// and test fixtures that must serialize the catalog without duplicating its
// on-disk contract.
type (
	PlansFileFormat  = plansFileFormat
	BundleFileFormat = bundleFileFormat
	PlanFileFormat   = planFileFormat
)

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
	// Log is an optional composition-root supplied observability seam.
	Log func(event string, fields map[string]interface{})
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
		log:            opts.Log,
	}
}

func (ps *PlanStore) logEvent(event string, fields map[string]interface{}) {
	if ps.log != nil {
		ps.log(event, fields)
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
	if err := NormalizeBundle(ps.bundle, ps.bundleKey, ps.displayEnv); err != nil {
		return fmt.Errorf("invalid bundle: %w", err)
	}

	seenPriceIDs := make(map[string]struct{}, len(ps.plans))
	for _, plan := range ps.plans {
		if plan == nil {
			return fmt.Errorf("plan is required")
		}
		if err := NormalizePlanOption(plan, ps.bundleKey); err != nil {
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

func (ps *PlanStore) GetPlanByExternalProductID(productID string) (*PlanOption, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("external product id is required")
	}
	for _, plan := range ps.plans {
		for _, key := range []string{"external_product_id", "product_id"} {
			if value, ok := plan.Metadata[key]; ok && value.GetStringValue() == productID {
				return proto.Clone(plan).(*PlanOption), nil
			}
		}
	}
	return nil, fmt.Errorf("external product %s not found", productID)
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
	return BuildPricingOverview(ps.bundle, ps.plans)
}

// ListBundleCatalog returns all bundles with their prices for admin UI.
func (ps *PlanStore) ListBundleCatalog(ctx context.Context) ([]BundleCatalogEntry, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return BuildBundleCatalog(ps.bundle, ps.plans), nil
}

// AddPlan adds a new plan to the store.
func (ps *PlanStore) AddPlan(plan *PlanOption) (*PlanOption, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if plan == nil {
		return nil, fmt.Errorf("plan is required")
	}

	cloned := proto.Clone(plan).(*PlanOption)
	if err := NormalizePlanOption(cloned, ps.bundleKey); err != nil {
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

// ApplyStripeImportSelections batches Stripe import updates into the plan store with a single save.
func (ps *PlanStore) ApplyStripeImportSelections(ctx context.Context, selections []ImportPlanSelection, fetcher StripePriceFetcher) (*StripeImportResult, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	result := &StripeImportResult{}
	if fetcher == nil {
		return result, ErrStripeImportMissingFetcher
	}
	if ps.bundle == nil {
		return result, ErrStripeImportBundleMissing
	}

	normalizedSelections, errorsList, err := NormalizeStripeImportSelections(selections)
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
					name := PlanNameFromStripeImport(priceDetails)
					displayEnabled := priceDetails.Active
					input := UpdateBundlePriceInput{
						PlanName:       &name,
						DisplayEnabled: &displayEnabled,
					}
					derivedTier, ok := DerivePlanTierFromStripe(priceDetails)
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

			plan := StripePriceImportToPlanOption(priceDetails)
			if err := NormalizePlanOption(plan, ps.bundleKey); err != nil {
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
		if err := NormalizeBundle(clonedBundle, ps.bundleKey, ps.displayEnv); err != nil {
			return err
		}
		ps.bundle = clonedBundle
	}

	nextPlans := make([]*PlanOption, 0, len(plans))
	seenPriceIDs := make(map[string]struct{}, len(plans))
	for _, plan := range plans {
		cloned := proto.Clone(plan).(*PlanOption)
		if err := NormalizePlanOption(cloned, ps.bundleKey); err != nil {
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

// ConvertProtoMetadataToMap converts protobuf metadata into JSON-compatible values.
func ConvertProtoMetadataToMap(m map[string]*commonv1.JsonValue) map[string]interface{} {
	return convertProtoMetadataToMap(m)
}

// ResolvePlansPath finds the plans.json file.
func ResolvePlansPath() string {
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
