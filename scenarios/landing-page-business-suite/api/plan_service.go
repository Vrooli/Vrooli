package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
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

type bundleProductRecord struct {
	ID      int64
	Bundle  *BundleProduct
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

// UpdateBundlePrice applies display overrides for a Stripe price row.
func (s *PlanService) UpdateBundlePrice(ctx context.Context, bundleKey, priceID string, input UpdateBundlePriceInput) (*PlanOption, error) {
	if priceID == "" || bundleKey == "" {
		return nil, nil
	}
	return s.planStore.UpdatePlan(priceID, input)
}

// AddPlan adds a new plan to the store.
func (s *PlanService) AddPlan(plan *PlanOption) error {
	return s.planStore.AddPlan(plan)
}

// DeletePlan removes a plan by its Stripe price ID.
func (s *PlanService) DeletePlan(priceID string) error {
	return s.planStore.DeletePlan(priceID)
}

// ReloadPlans reloads plans from the JSON file.
func (s *PlanService) ReloadPlans() error {
	return s.planStore.LoadAll()
}

// Helper functions for metadata conversion and enum mapping

func parseMetadata(metadataBytes []byte) map[string]*commonv1.JsonValue {
	if len(metadataBytes) == 0 {
		return nil
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(metadataBytes, &meta); err != nil {
		logStructured("plan metadata unmarshal failed", map[string]interface{}{
			"level": "warn",
			"error": err.Error(),
		})
		return nil
	}

	result := make(map[string]*commonv1.JsonValue, len(meta))
	for key, value := range meta {
		if jv := toJsonValue(value); jv != nil {
			result[key] = jv
		}
	}

	return result
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

// jsonValueToMap converts a map of JsonValue to a map of any for JSON marshaling.
func jsonValueToMap(m map[string]*commonv1.JsonValue) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = jsonValueToAny(v)
	}
	return result
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

func mapIntroPricingType(raw sql.NullString) landing_page_react_vite_v1.IntroPricingType {
	if !raw.Valid {
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED
	}

	value := strings.TrimSpace(strings.ToLower(raw.String))
	switch value {
	case "", "unspecified", "none":
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED
	case "percentage", "percent", "pct":
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE
	case "flat_amount", "flat-amount", "flat", "amount":
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT
	default:
		if parsed, err := strconv.Atoi(value); err == nil {
			switch landing_page_react_vite_v1.IntroPricingType(parsed) {
			case landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE:
				return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE
			case landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT:
				return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT
			}
		}
		return landing_page_react_vite_v1.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED
	}
}

// clonePlanOption creates a deep copy of a PlanOption.
func clonePlanOption(p *PlanOption) *PlanOption {
	if p == nil {
		return nil
	}
	return proto.Clone(p).(*PlanOption)
}
