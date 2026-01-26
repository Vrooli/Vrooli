package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
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
	mu         sync.RWMutex
	bundle     *BundleProduct
	plans      []*PlanOption
	plansPath  string
	displayEnv string
	bundleKey  string
	updatedAt  time.Time
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
	Bundle    bundleFileFormat `json:"bundle"`
	Plans     []planFileFormat `json:"plans"`
	UpdatedAt string           `json:"updated_at,omitempty"`
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
	bundleKey := stringsTrimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production")
	return &PlanStore{
		plans:      make([]*PlanOption, 0),
		plansPath:  plansPath,
		displayEnv: env,
		bundleKey:  bundleKey,
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
		bundleKey = stringsTrimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite")
	}
	env := opts.DisplayEnv
	if env == "" {
		env = stringsTrimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production")
	}
	return &PlanStore{
		plans:      make([]*PlanOption, 0),
		plansPath:  opts.PlansPath,
		displayEnv: env,
		bundleKey:  bundleKey,
	}
}

// BundleKey returns the configured bundle key.
func (ps *PlanStore) BundleKey() string {
	return ps.bundleKey
}

// LoadAll loads plan configuration from JSON file into memory.
func (ps *PlanStore) LoadAll() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.plansPath == "" {
		logStructured("plans_path_not_set", map[string]interface{}{
			"fallback": "empty plans",
		})
		return nil
	}

	data, err := os.ReadFile(ps.plansPath)
	if err != nil {
		if os.IsNotExist(err) {
			logStructured("plans_file_not_found", map[string]interface{}{
				"path":     ps.plansPath,
				"fallback": "empty plans",
			})
			return nil
		}
		return err
	}

	var fileData plansFileFormat
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("parse plans JSON: %w", err)
	}

	// Convert bundle
	bundle := &BundleProduct{
		BundleKey:                fileData.Bundle.BundleKey,
		Name:                     fileData.Bundle.Name,
		StripeProductId:          fileData.Bundle.StripeProductID,
		CreditsPerUsd:            fileData.Bundle.CreditsPerUSD,
		DisplayCreditsMultiplier: fileData.Bundle.DisplayCreditsMultiplier,
		DisplayCreditsLabel:      fileData.Bundle.DisplayCreditsLabel,
		Environment:              fileData.Bundle.Environment,
	}

	if fileData.Bundle.Metadata != nil {
		bundle.Metadata = convertMetadataToProto(fileData.Bundle.Metadata)
	}

	if err := normalizeBundleProduct(bundle, ps.bundleKey, ps.displayEnv); err != nil {
		return fmt.Errorf("invalid bundle config: %w", err)
	}

	// Convert plans
	plans := make([]*PlanOption, 0, len(fileData.Plans))
	seenPriceIDs := make(map[string]struct{}, len(fileData.Plans))
	for _, planFile := range fileData.Plans {
		plan := &PlanOption{
			StripePriceId:          planFile.StripePriceID,
			PlanName:               planFile.PlanName,
			PlanTier:               planFile.PlanTier,
			BillingInterval:        mapBillingInterval(planFile.BillingInterval),
			AmountCents:            planFile.AmountCents,
			Currency:               planFile.Currency,
			DisplayWeight:          planFile.DisplayWeight,
			DisplayEnabled:         planFile.DisplayEnabled,
			MonthlyIncludedCredits: planFile.MonthlyIncludedCredits,
			OneTimeBonusCredits:    planFile.OneTimeBonusCredits,
			PlanRank:               planFile.PlanRank,
			BonusType:              planFile.BonusType,
			Kind:                   mapPlanKind(planFile.Kind),
			IntroEnabled:           planFile.IntroEnabled,
			IntroPeriods:           planFile.IntroPeriods,
			IntroPriceLookupKey:    planFile.IntroPriceLookupKey,
			IsVariableAmount:       planFile.IsVariableAmount,
			BundleKey:              ps.bundleKey,
		}

		if planFile.IntroAmountCents != nil {
			plan.IntroAmountCents = proto.Int64(*planFile.IntroAmountCents)
		}

		if planFile.IntroType != "" {
			plan.IntroType = mapIntroPricingTypeFromString(planFile.IntroType)
		}

		if planFile.Metadata != nil {
			plan.Metadata = convertMetadataToProto(planFile.Metadata)
		}

		if err := normalizePlanOption(plan, ps.bundleKey); err != nil {
			return fmt.Errorf("invalid plan %s: %w", plan.StripePriceId, err)
		}
		if _, exists := seenPriceIDs[plan.StripePriceId]; exists {
			return fmt.Errorf("duplicate stripe_price_id detected: %s", plan.StripePriceId)
		}
		seenPriceIDs[plan.StripePriceId] = struct{}{}

		plans = append(plans, plan)
	}

	if fileData.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, fileData.UpdatedAt); err == nil {
			ps.updatedAt = t
		}
	}

	ps.bundle = bundle
	ps.plans = plans

	logStructured("plans_loaded", map[string]interface{}{
		"path":       ps.plansPath,
		"plan_count": len(ps.plans),
		"bundle_key": ps.bundle.BundleKey,
	})

	return nil
}

// SavePlans writes plan configuration back to the JSON file.
func (ps *PlanStore) SavePlans() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.savePlansLocked()
}

func (ps *PlanStore) savePlansLocked() error {
	if ps.plansPath == "" {
		return fmt.Errorf("plans path not configured")
	}
	if err := ps.validatePlanCatalogLocked(); err != nil {
		return err
	}

	fileData := plansFileFormat{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Convert bundle
	if ps.bundle != nil {
		fileData.Bundle = bundleFileFormat{
			BundleKey:                ps.bundle.BundleKey,
			Name:                     ps.bundle.Name,
			StripeProductID:          ps.bundle.StripeProductId,
			CreditsPerUSD:            ps.bundle.CreditsPerUsd,
			DisplayCreditsMultiplier: ps.bundle.DisplayCreditsMultiplier,
			DisplayCreditsLabel:      ps.bundle.DisplayCreditsLabel,
			Environment:              ps.bundle.Environment,
		}
		if ps.bundle.Metadata != nil {
			fileData.Bundle.Metadata = convertProtoMetadataToMap(ps.bundle.Metadata)
		}
	}

	// Convert plans
	fileData.Plans = make([]planFileFormat, 0, len(ps.plans))
	for _, plan := range ps.plans {
		planFile := planFileFormat{
			StripePriceID:          plan.StripePriceId,
			PlanName:               plan.PlanName,
			PlanTier:               plan.PlanTier,
			BillingInterval:        billingIntervalLabel(plan.BillingInterval),
			AmountCents:            plan.AmountCents,
			Currency:               plan.Currency,
			DisplayWeight:          plan.DisplayWeight,
			DisplayEnabled:         plan.DisplayEnabled,
			MonthlyIncludedCredits: plan.MonthlyIncludedCredits,
			OneTimeBonusCredits:    plan.OneTimeBonusCredits,
			PlanRank:               plan.PlanRank,
			BonusType:              plan.BonusType,
			Kind:                   planKindString(plan.Kind),
			IntroEnabled:           plan.IntroEnabled,
			IntroPeriods:           plan.IntroPeriods,
			IntroPriceLookupKey:    plan.IntroPriceLookupKey,
			IsVariableAmount:       plan.IsVariableAmount,
		}

		if plan.IntroAmountCents != nil {
			val := *plan.IntroAmountCents
			planFile.IntroAmountCents = &val
		}

		if plan.IntroType != landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED {
			planFile.IntroType = introPricingTypeString(plan.IntroType)
		}

		if plan.Metadata != nil {
			planFile.Metadata = convertProtoMetadataToMap(plan.Metadata)
		}

		fileData.Plans = append(fileData.Plans, planFile)
	}

	// Ensure directory exists
	dir := filepath.Dir(ps.plansPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create plans directory: %w", err)
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plans: %w", err)
	}

	if err := writeFileAtomic(ps.plansPath, data, 0o644); err != nil {
		return fmt.Errorf("write plans file: %w", err)
	}

	ps.updatedAt = time.Now()

	logStructured("plans_saved", map[string]interface{}{
		"path":       ps.plansPath,
		"plan_count": len(ps.plans),
	})

	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, ".plans-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempName := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempName)
	}()

	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tempFile.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace plans file: %w", err)
	}
	return nil
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
		case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH:
			monthly = append(monthly, proto.Clone(plan).(*PlanOption))
		case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_YEAR:
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

// UpdatePlan applies partial updates to a plan.
func (ps *PlanStore) UpdatePlan(priceID string, input UpdateBundlePriceInput) (*PlanOption, error) {
	return ps.UpdatePlanWithStripeDetails(priceID, input, nil)
}

// UpdatePlanWithStripeDetails applies partial updates to a plan and optionally syncs Stripe fields.
func (ps *PlanStore) UpdatePlanWithStripeDetails(priceID string, input UpdateBundlePriceInput, stripeDetails *StripePriceImport) (*PlanOption, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	derivedTier := ""
	if stripeDetails != nil {
		if tier, ok := derivePlanTierFromStripe(stripeDetails); ok {
			derivedTier = tier
		}
	}

	updatedPlan, err := ps.updatePlanWithStripeDetailsLocked(priceID, input, stripeDetails, derivedTier)
	if err != nil {
		return nil, err
	}

	if err := ps.savePlansLocked(); err != nil {
		return nil, err
	}

	return proto.Clone(updatedPlan).(*PlanOption), nil
}

func (ps *PlanStore) updatePlanWithStripeDetailsLocked(priceID string, input UpdateBundlePriceInput, stripeDetails *StripePriceImport, derivedTier string) (*PlanOption, error) {
	if priceID == "" {
		return nil, fmt.Errorf("price id is required")
	}

	if stripeDetails != nil {
		if err := ps.ensureStripePriceMatchesBundleLocked(stripeDetails); err != nil {
			return nil, err
		}
	}

	targetIdx := -1
	for i, plan := range ps.plans {
		if plan.StripePriceId == priceID {
			targetIdx = i
			break
		}
	}

	if targetIdx < 0 {
		return nil, fmt.Errorf("price %s not found", priceID)
	}

	currentPlan := ps.plans[targetIdx]
	updatedPlan := proto.Clone(currentPlan).(*PlanOption)

	// Apply updates
	if input.StripePriceID != nil {
		trimmed, err := normalizeStripePriceID(*input.StripePriceID)
		if err != nil {
			return nil, err
		}
		if trimmed != currentPlan.StripePriceId {
			if stripeDetails == nil {
				return nil, fmt.Errorf("stripe price changes require a verified Stripe price")
			}
			if strings.TrimSpace(stripeDetails.PriceID) != trimmed {
				return nil, fmt.Errorf("stripe price verification mismatch for %s", trimmed)
			}
			for _, existing := range ps.plans {
				if existing.StripePriceId == trimmed {
					return nil, fmt.Errorf("plan with price ID %s already exists", trimmed)
				}
			}
			updatedPlan.StripePriceId = trimmed
		}
	}
	if input.PlanName != nil {
		name := strings.TrimSpace(*input.PlanName)
		if name == "" {
			return nil, fmt.Errorf("plan_name is required")
		}
		updatedPlan.PlanName = name
	}
	if input.DisplayWeight != nil {
		if *input.DisplayWeight < 0 {
			return nil, fmt.Errorf("display_weight must be >= 0")
		}
		if *input.DisplayWeight > math.MaxInt32 {
			return nil, fmt.Errorf("display_weight must be <= %d", math.MaxInt32)
		}
		updatedPlan.DisplayWeight = int32(*input.DisplayWeight)
	}
	if input.DisplayEnabled != nil {
		updatedPlan.DisplayEnabled = *input.DisplayEnabled
	}

	if stripeDetails != nil {
		interval := mapBillingInterval(stripeDetails.Interval)
		if err := validateBillingInterval(interval); err != nil {
			return nil, fmt.Errorf("invalid stripe billing interval: %w", err)
		}
		currency, err := normalizeCurrency(stripeDetails.Currency)
		if err != nil {
			return nil, fmt.Errorf("invalid stripe currency: %w", err)
		}
		if stripeDetails.AmountCents < 0 {
			return nil, fmt.Errorf("stripe amount_cents must be >= 0")
		}

		updatedPlan.AmountCents = stripeDetails.AmountCents
		updatedPlan.Currency = currency
		updatedPlan.BillingInterval = interval

		if input.DisplayEnabled == nil && !stripeDetails.Active {
			updatedPlan.DisplayEnabled = false
		}
	}

	if strings.TrimSpace(derivedTier) != "" {
		normalizedTier, err := normalizePlanTier(derivedTier)
		if err != nil {
			return nil, err
		}
		if normalizedTier != updatedPlan.PlanTier {
			updatedPlan.PlanTier = normalizedTier
			updatedPlan.PlanRank = planRankForTier(normalizedTier)
			updatedPlan.Kind = planKindForTier(normalizedTier)
		}
	}

	// Update metadata
	if updatedPlan.Metadata == nil {
		updatedPlan.Metadata = map[string]*commonv1.JsonValue{}
	}

	updateMetadataString := func(key string, value *string) {
		if value == nil {
			return
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			delete(updatedPlan.Metadata, key)
			return
		}
		updatedPlan.Metadata[key] = newStringJsonValue(trimmed)
	}

	updateMetadataString("subtitle", input.Subtitle)
	updateMetadataString("badge", input.Badge)
	updateMetadataString("cta_label", input.CtaLabel)

	if input.Highlight != nil {
		if *input.Highlight {
			updatedPlan.Metadata["highlight"] = newBoolJsonValue(true)
		} else {
			delete(updatedPlan.Metadata, "highlight")
		}
	}

	if input.Features != nil {
		var sanitized []string
		for _, feature := range *input.Features {
			trimmed := strings.TrimSpace(feature)
			if trimmed != "" {
				sanitized = append(sanitized, trimmed)
			}
		}
		if len(sanitized) == 0 {
			delete(updatedPlan.Metadata, "features")
		} else {
			listValues := make([]*commonv1.JsonValue, 0, len(sanitized))
			for _, feature := range sanitized {
				listValues = append(listValues, newStringJsonValue(feature))
			}
			updatedPlan.Metadata["features"] = newListJsonValue(listValues)
		}
	}

	updatedPlan.BundleKey = ps.bundleKey
	if updatedPlan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_UNSPECIFIED {
		updatedPlan.Kind = planKindForTier(updatedPlan.PlanTier)
	}

	normalizedTier, err := normalizePlanTier(updatedPlan.PlanTier)
	if err != nil {
		return nil, err
	}
	updatedPlan.PlanTier = normalizedTier

	if _, err := normalizePlanName(updatedPlan.PlanName); err != nil {
		return nil, err
	}
	if err := validateBillingInterval(updatedPlan.BillingInterval); err != nil {
		return nil, err
	}
	normalizedCurrency, err := normalizeCurrency(updatedPlan.Currency)
	if err != nil {
		return nil, err
	}
	updatedPlan.Currency = normalizedCurrency
	if updatedPlan.AmountCents < 0 {
		return nil, fmt.Errorf("amount_cents must be >= 0")
	}
	if updatedPlan.MonthlyIncludedCredits < 0 {
		return nil, fmt.Errorf("monthly_included_credits must be >= 0")
	}
	if updatedPlan.OneTimeBonusCredits < 0 {
		return nil, fmt.Errorf("one_time_bonus_credits must be >= 0")
	}
	if updatedPlan.PlanRank < 0 {
		return nil, fmt.Errorf("plan_rank must be >= 0")
	}
	if err := validatePlanTierConstraints(updatedPlan); err != nil {
		return nil, err
	}

	if updatedPlan.Metadata != nil && len(updatedPlan.Metadata) == 0 {
		updatedPlan.Metadata = nil
	}

	ps.plans[targetIdx] = updatedPlan
	return updatedPlan, nil
}

func (ps *PlanStore) ensureStripePriceMatchesBundleLocked(stripeDetails *StripePriceImport) error {
	if stripeDetails == nil {
		return nil
	}
	if ps.bundle == nil {
		return fmt.Errorf("bundle not configured")
	}

	bundleProductID := strings.TrimSpace(ps.bundle.StripeProductId)
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

func mapIntroPricingTypeFromString(s string) landing_page_react_vite_v1.IntroPricingType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "percentage", "percent", "pct":
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE
	case "flat_amount", "flat-amount", "flat", "amount":
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT
	default:
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED
	}
}

func introPricingTypeString(t landing_page_react_vite_v1.IntroPricingType) string {
	switch t {
	case landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE:
		return "percentage"
	case landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT:
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
	if plan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_UNSPECIFIED {
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
		if plan.BillingInterval != landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_ONE_TIME {
			return fmt.Errorf("%s plans must use one_time billing_interval", plan.PlanTier)
		}
	}

	return nil
}
