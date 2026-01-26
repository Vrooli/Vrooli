package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	AddPlan(plan *PlanOption) error
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
	mu          sync.RWMutex
	bundle      *BundleProduct
	plans       []*PlanOption
	plansPath   string
	displayEnv  string
	bundleKey   string
	updatedAt   time.Time
}

// plansFileFormat represents the JSON file structure for plans.
type plansFileFormat struct {
	Bundle    bundleFileFormat  `json:"bundle"`
	Plans     []planFileFormat  `json:"plans"`
	UpdatedAt string            `json:"updated_at,omitempty"`
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
	ps.bundle = &BundleProduct{
		BundleKey:                fileData.Bundle.BundleKey,
		Name:                     fileData.Bundle.Name,
		StripeProductId:          fileData.Bundle.StripeProductID,
		CreditsPerUsd:            fileData.Bundle.CreditsPerUSD,
		DisplayCreditsMultiplier: fileData.Bundle.DisplayCreditsMultiplier,
		DisplayCreditsLabel:      fileData.Bundle.DisplayCreditsLabel,
		Environment:              fileData.Bundle.Environment,
	}

	if fileData.Bundle.Metadata != nil {
		ps.bundle.Metadata = convertMetadataToProto(fileData.Bundle.Metadata)
	}

	// Convert plans
	ps.plans = make([]*PlanOption, 0, len(fileData.Plans))
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

		ps.plans = append(ps.plans, plan)
	}

	if fileData.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, fileData.UpdatedAt); err == nil {
			ps.updatedAt = t
		}
	}

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

	if err := os.WriteFile(ps.plansPath, data, 0o644); err != nil {
		return fmt.Errorf("write plans file: %w", err)
	}

	ps.updatedAt = time.Now()

	logStructured("plans_saved", map[string]interface{}{
		"path":       ps.plansPath,
		"plan_count": len(ps.plans),
	})

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
func (ps *PlanStore) AddPlan(plan *PlanOption) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if plan.StripePriceId == "" {
		return fmt.Errorf("stripe_price_id is required")
	}

	// Check for duplicate
	for _, existing := range ps.plans {
		if existing.StripePriceId == plan.StripePriceId {
			return fmt.Errorf("plan with price ID %s already exists", plan.StripePriceId)
		}
	}

	// Set bundle key
	plan.BundleKey = ps.bundleKey

	ps.plans = append(ps.plans, proto.Clone(plan).(*PlanOption))
	return ps.savePlansLocked()
}

// UpdatePlan applies partial updates to a plan.
func (ps *PlanStore) UpdatePlan(priceID string, input UpdateBundlePriceInput) (*PlanOption, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if priceID == "" {
		return nil, fmt.Errorf("price id is required")
	}

	var targetPlan *PlanOption
	for _, plan := range ps.plans {
		if plan.StripePriceId == priceID {
			targetPlan = plan
			break
		}
	}

	if targetPlan == nil {
		return nil, fmt.Errorf("price %s not found", priceID)
	}

	// Apply updates
	if input.StripePriceID != nil {
		targetPlan.StripePriceId = strings.TrimSpace(*input.StripePriceID)
	}
	if input.PlanName != nil {
		targetPlan.PlanName = strings.TrimSpace(*input.PlanName)
	}
	if input.DisplayWeight != nil {
		targetPlan.DisplayWeight = int32(*input.DisplayWeight)
	}
	if input.DisplayEnabled != nil {
		targetPlan.DisplayEnabled = *input.DisplayEnabled
	}

	// Update metadata
	if targetPlan.Metadata == nil {
		targetPlan.Metadata = map[string]*commonv1.JsonValue{}
	}

	updateMetadataString := func(key string, value *string) {
		if value == nil {
			return
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			delete(targetPlan.Metadata, key)
			return
		}
		targetPlan.Metadata[key] = newStringJsonValue(trimmed)
	}

	updateMetadataString("subtitle", input.Subtitle)
	updateMetadataString("badge", input.Badge)
	updateMetadataString("cta_label", input.CtaLabel)

	if input.Highlight != nil {
		if *input.Highlight {
			targetPlan.Metadata["highlight"] = newBoolJsonValue(true)
		} else {
			delete(targetPlan.Metadata, "highlight")
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
			delete(targetPlan.Metadata, "features")
		} else {
			listValues := make([]*commonv1.JsonValue, 0, len(sanitized))
			for _, feature := range sanitized {
				listValues = append(listValues, newStringJsonValue(feature))
			}
			targetPlan.Metadata["features"] = newListJsonValue(listValues)
		}
	}

	if err := ps.savePlansLocked(); err != nil {
		return nil, err
	}

	return proto.Clone(targetPlan).(*PlanOption), nil
}

// DeletePlan removes a plan by its Stripe price ID.
func (ps *PlanStore) DeletePlan(priceID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

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
		ps.bundle = proto.Clone(bundle).(*BundleProduct)
	}

	ps.plans = make([]*PlanOption, 0, len(plans))
	for _, plan := range plans {
		cloned := proto.Clone(plan).(*PlanOption)
		cloned.BundleKey = ps.bundleKey
		ps.plans = append(ps.plans, cloned)
	}

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
