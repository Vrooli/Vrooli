package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// PlanService exposes helper utilities for pricing/plan metadata.
type PlanService struct {
	db            *sql.DB
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
	Updated time.Time
}

func NewPlanService(db *sql.DB) *PlanService {
	bundle := stringsTrimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production")
	return &PlanService{db: db, defaultBundle: bundle, displayEnv: env}
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

// GetPricingOverview loads the product and price rows for the default bundle.
func (s *PlanService) GetPricingOverview() (*PricingOverview, error) {
	product, err := s.loadBundleProduct(s.defaultBundle)
	if err != nil {
		return nil, err
	}

	prices, err := s.loadBundlePrices(product.ID)
	if err != nil {
		return nil, err
	}

	// Initialize as empty slices (not nil) so JSON serialization includes them as [] instead of omitting
	monthly := make([]*PlanOption, 0)
	yearly := make([]*PlanOption, 0)

	// Log raw prices for debugging
	for i, price := range prices {
		logStructured("price_loaded", map[string]interface{}{
			"level":            "debug",
			"index":            i,
			"plan_name":        price.PlanName,
			"billing_interval": price.BillingInterval.String(),
			"display_enabled":  price.DisplayEnabled,
			"tier":             price.PlanTier,
		})
	}

	for _, price := range prices {
		if !price.GetDisplayEnabled() && strings.ToLower(strings.TrimSpace(price.PlanTier)) != "free" {
			continue
		}
		switch price.BillingInterval {
		case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH:
			monthly = append(monthly, proto.Clone(price).(*PlanOption))
		case landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_YEAR:
			yearly = append(yearly, proto.Clone(price).(*PlanOption))
		default:
			logStructured("unmatched_billing_interval", map[string]interface{}{
				"level":     "warn",
				"plan_name": price.PlanName,
				"interval":  price.BillingInterval.String(),
				"tier":      price.PlanTier,
			})
		}
	}

	sort.SliceStable(monthly, func(i, j int) bool {
		if monthly[i].DisplayWeight == monthly[j].DisplayWeight {
			return monthly[i].PlanRank < monthly[j].PlanRank
		}
		return monthly[i].DisplayWeight > monthly[j].DisplayWeight
	})
	sort.SliceStable(yearly, func(i, j int) bool {
		if yearly[i].DisplayWeight == yearly[j].DisplayWeight {
			return yearly[i].PlanRank < yearly[j].PlanRank
		}
		return yearly[i].DisplayWeight > yearly[j].DisplayWeight
	})

	logStructured("pricing_overview_built", map[string]interface{}{
		"level":         "info",
		"monthly_count": len(monthly),
		"yearly_count":  len(yearly),
		"total_prices":  len(prices),
	})

	return &PricingOverview{
		Bundle:    product.Bundle,
		Monthly:   monthly,
		Yearly:    yearly,
		UpdatedAt: timestamppb.Now(),
	}, nil
}

// GetPlanByPriceID fetches a plan option for a Stripe price identifier.
func (s *PlanService) GetPlanByPriceID(priceID string) (*PlanOption, error) {
	if priceID == "" {
		return nil, fmt.Errorf("price id is required")
	}

	query := `
		SELECT bp.stripe_price_id, bp.plan_name, bp.plan_tier, bp.billing_interval,
		       bp.amount_cents, bp.currency, bp.intro_enabled, bp.intro_type,
		       bp.intro_amount_cents, bp.intro_periods, bp.intro_price_lookup_key,
		       bp.monthly_included_credits, bp.one_time_bonus_credits, bp.plan_rank,
		       bp.bonus_type, bp.kind, bp.is_variable_amount, bp.display_enabled, b.bundle_key,
		       bp.metadata, bp.display_weight
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.stripe_price_id = $1
	`

	row := s.db.QueryRow(query, priceID)
	option, err := s.scanPlanOption(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("price %s not found", priceID)
	}
	if err != nil {
		return nil, err
	}

	return option, nil
}

func (s *PlanService) loadBundleProduct(bundleKey string) (*bundleProductRecord, error) {
	query := `
		SELECT id, bundle_key, bundle_name, stripe_product_id, credits_per_usd,
		       display_credits_multiplier, display_credits_label, environment, metadata
		FROM bundle_products
		WHERE bundle_key = $1 AND environment = $2
		LIMIT 1
	`
	row := s.db.QueryRow(query, bundleKey, s.displayEnv)

	var id int64
	product := &BundleProduct{}
	var metadataBytes []byte
	if err := row.Scan(
		&id,
		&product.BundleKey,
		&product.Name,
		&product.StripeProductId,
		&product.CreditsPerUsd,
		&product.DisplayCreditsMultiplier,
		&product.DisplayCreditsLabel,
		&product.Environment,
		&metadataBytes,
	); err != nil {
		return nil, fmt.Errorf("bundle %s not found: %w", bundleKey, err)
	}

	if len(metadataBytes) > 0 {
		if meta := parseMetadata(metadataBytes); meta != nil {
			product.Metadata = meta
		}
	}

	return &bundleProductRecord{ID: id, Bundle: product}, nil
}

func (s *PlanService) loadBundlePrices(productID int64) ([]*PlanOption, error) {
	query := `
		SELECT bp.id, bp.stripe_price_id, bp.plan_name, bp.plan_tier, bp.billing_interval,
		       bp.amount_cents, bp.currency, bp.intro_enabled, bp.intro_type, bp.intro_amount_cents,
		       bp.intro_periods, bp.intro_price_lookup_key, bp.monthly_included_credits,
		       bp.one_time_bonus_credits, bp.plan_rank, bp.bonus_type, bp.kind, bp.is_variable_amount,
		       bp.display_enabled, b.bundle_key, bp.metadata, bp.display_weight
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.product_id = $1
		ORDER BY bp.display_weight DESC, bp.plan_rank ASC
	`

	logStructured("loadBundlePrices_query", map[string]interface{}{
		"level":      "debug",
		"product_id": productID,
	})

	rows, err := s.db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*PlanOption
	rowCount := 0
	for rows.Next() {
		rowCount++
		var option PlanOption
		var pricePrimaryID int64
		var metadataBytes []byte
		var introAmount sql.NullInt64
		var rawKind string
		var rawIntroType sql.NullString
		var rawIntroLookup sql.NullString
		var rawInterval string
		var stripePriceID sql.NullString
		if err := rows.Scan(
			&pricePrimaryID,
			&stripePriceID,
			&option.PlanName,
			&option.PlanTier,
			&rawInterval,
			&option.AmountCents,
			&option.Currency,
			&option.IntroEnabled,
			&rawIntroType,
			&introAmount,
			&option.IntroPeriods,
			&rawIntroLookup,
			&option.MonthlyIncludedCredits,
			&option.OneTimeBonusCredits,
			&option.PlanRank,
			&option.BonusType,
			&rawKind,
			&option.IsVariableAmount,
			&option.DisplayEnabled,
			&option.BundleKey,
			&metadataBytes,
			&option.DisplayWeight,
		); err != nil {
			return nil, err
		}

		if stripePriceID.Valid {
			option.StripePriceId = strings.TrimSpace(stripePriceID.String)
		} else {
			option.StripePriceId = ""
		}

		if introAmount.Valid {
			val := introAmount.Int64
			option.IntroAmountCents = proto.Int64(val)
		}
		if rawIntroLookup.Valid {
			option.IntroPriceLookupKey = rawIntroLookup.String
		}

		if len(metadataBytes) > 0 {
			if meta := parseMetadata(metadataBytes); meta != nil {
				option.Metadata = meta
			}
		}

		if option.Metadata == nil {
			option.Metadata = map[string]*commonv1.JsonValue{}
		}

		// When stripe_price_id is empty (free/CTA-only), attach the DB primary key so admin/UI can round-trip.
		if strings.TrimSpace(option.StripePriceId) == "" {
			option.Metadata["__price_pk"] = newStringJsonValue(fmt.Sprintf("%d", pricePrimaryID))
		}

		option.IntroType = mapIntroPricingType(rawIntroType)
		option.Kind = mapPlanKind(rawKind)
		option.BillingInterval = mapBillingInterval(rawInterval)

		// Debug: log what we're reading from DB and what we're converting to
		logStructured("price_interval_conversion", map[string]interface{}{
			"level":        "debug",
			"plan_name":    option.PlanName,
			"raw_interval": rawInterval,
			"mapped_enum":  option.BillingInterval.String(),
		})

		copied := option
		options = append(options, &copied)
	}

	logStructured("loadBundlePrices_done", map[string]interface{}{
		"level":         "debug",
		"rows_scanned":  rowCount,
		"options_count": len(options),
	})

	return options, nil
}

// GetBundleProduct returns the configured bundle product metadata.
func (s *PlanService) GetBundleProduct() (*BundleProduct, error) {
	rec, err := s.loadBundleProduct(s.defaultBundle)
	if err != nil {
		return nil, err
	}
	return rec.Bundle, nil
}

// BundleCatalogEntry groups a bundle with all of its prices (visible + hidden).
type BundleCatalogEntry struct {
	Bundle *BundleProduct `json:"bundle"`
	Prices []*PlanOption  `json:"prices"`
}

// ListBundleCatalog returns bundles for the configured environment so the admin UI
// can toggle prices without raw SQL edits.
func (s *PlanService) ListBundleCatalog(ctx context.Context) ([]BundleCatalogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, bundle_key, bundle_name, stripe_product_id, credits_per_usd,
		       display_credits_multiplier, display_credits_label, environment, metadata
		FROM bundle_products
		WHERE environment = $1
		ORDER BY bundle_key ASC
	`, s.displayEnv)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []BundleCatalogEntry
	for rows.Next() {
		var id int64
		product := &BundleProduct{}
		var metadataBytes []byte
		if err := rows.Scan(
			&id,
			&product.BundleKey,
			&product.Name,
			&product.StripeProductId,
			&product.CreditsPerUsd,
			&product.DisplayCreditsMultiplier,
			&product.DisplayCreditsLabel,
			&product.Environment,
			&metadataBytes,
		); err != nil {
			return nil, err
		}

		if len(metadataBytes) > 0 {
			if meta := parseMetadata(metadataBytes); meta != nil {
				product.Metadata = meta
			}
		}

		prices, err := s.loadBundlePrices(id)
		if err != nil {
			return nil, err
		}

		entries = append(entries, BundleCatalogEntry{
			Bundle: product,
			Prices: prices,
		})
	}

	return entries, nil
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
		return nil, fmt.Errorf("bundle key and price id are required")
	}

	stripeOverride := sql.NullString{Valid: false}
	if input.StripePriceID != nil {
		stripeOverride = sql.NullString{String: strings.TrimSpace(*input.StripePriceID), Valid: true}
	}

	var pricePrimaryID int64
	var metadataBytes []byte
	query := `
		SELECT bp.id, bp.metadata
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.stripe_price_id = $1 AND b.bundle_key = $2 AND b.environment = $3
	`
	if err := s.db.QueryRowContext(ctx, query, priceID, bundleKey, s.displayEnv).Scan(&pricePrimaryID, &metadataBytes); err != nil {
		if err == sql.ErrNoRows {
			if numericID, parseErr := strconv.ParseInt(priceID, 10, 64); parseErr == nil {
				altQuery := `
					SELECT bp.id, bp.metadata
					FROM bundle_prices bp
					JOIN bundle_products b ON bp.product_id = b.id
					WHERE bp.id = $1 AND b.bundle_key = $2 AND b.environment = $3
				`
				if err := s.db.QueryRowContext(ctx, altQuery, numericID, bundleKey, s.displayEnv).Scan(&pricePrimaryID, &metadataBytes); err != nil {
					if err == sql.ErrNoRows {
						return nil, fmt.Errorf("price %s not found for bundle %s", priceID, bundleKey)
					}
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("price %s not found for bundle %s", priceID, bundleKey)
			}
		}
		return nil, err
	}

	metadata := parseMetadata(metadataBytes)
	if metadata == nil {
		metadata = map[string]*commonv1.JsonValue{}
	}

	setMetadataString := func(key string, value *string) {
		if value == nil {
			return
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			delete(metadata, key)
			return
		}
		metadata[key] = newStringJsonValue(trimmed)
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
			delete(metadata, "features")
		} else {
			listValues := make([]*commonv1.JsonValue, 0, len(sanitized))
			for _, feature := range sanitized {
				listValues = append(listValues, newStringJsonValue(feature))
			}
			metadata["features"] = newListJsonValue(listValues)
		}
	}

	setMetadataString("subtitle", input.Subtitle)
	setMetadataString("badge", input.Badge)
	setMetadataString("cta_label", input.CtaLabel)
	if input.Highlight != nil {
		if *input.Highlight {
			metadata["highlight"] = newBoolJsonValue(true)
		} else {
			delete(metadata, "highlight")
		}
	}

	metadataJSON, err := json.Marshal(jsonValueToMap(metadata))
	if err != nil {
		return nil, fmt.Errorf("marshal price metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE bundle_prices
		SET stripe_price_id = CASE WHEN $1::text IS NOT NULL THEN NULLIF($1::text, '') ELSE stripe_price_id END,
		    plan_name = COALESCE($2, plan_name),
		    display_weight = COALESCE($3, display_weight),
		    display_enabled = COALESCE($4, display_enabled),
		    metadata = $5,
		    updated_at = NOW()
		WHERE id = $6
	`, stripeOverride, input.PlanName, input.DisplayWeight, input.DisplayEnabled, metadataJSON, pricePrimaryID)
	if err != nil {
		return nil, err
	}

	return s.getPlanByInternalID(pricePrimaryID)
}

// getPlanByInternalID fetches a plan option by primary key (used when stripe_price_id is cleared).
func (s *PlanService) getPlanByInternalID(id int64) (*PlanOption, error) {
	query := `
		SELECT bp.stripe_price_id, bp.plan_name, bp.plan_tier, bp.billing_interval,
		       bp.amount_cents, bp.currency, bp.intro_enabled, bp.intro_type,
		       bp.intro_amount_cents, bp.intro_periods, bp.intro_price_lookup_key,
		       bp.monthly_included_credits, bp.one_time_bonus_credits, bp.plan_rank,
		       bp.bonus_type, bp.kind, bp.is_variable_amount, bp.display_enabled, b.bundle_key,
		       bp.metadata, bp.display_weight
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.id = $1
	`
	row := s.db.QueryRow(query, id)
	return s.scanPlanOption(row)
}

func (s *PlanService) scanPlanOption(row *sql.Row) (*PlanOption, error) {
	option := &PlanOption{}
	var metadataBytes []byte
	var rawKind string
	var rawInterval string
	var rawIntroType sql.NullString
	var rawIntroLookup sql.NullString
	var introAmount sql.NullInt64
	var stripePriceID sql.NullString
	if err := row.Scan(
		&stripePriceID,
		&option.PlanName,
		&option.PlanTier,
		&rawInterval,
		&option.AmountCents,
		&option.Currency,
		&option.IntroEnabled,
		&rawIntroType,
		&introAmount,
		&option.IntroPeriods,
		&rawIntroLookup,
		&option.MonthlyIncludedCredits,
		&option.OneTimeBonusCredits,
		&option.PlanRank,
		&option.BonusType,
		&rawKind,
		&option.IsVariableAmount,
		&option.DisplayEnabled,
		&option.BundleKey,
		&metadataBytes,
		&option.DisplayWeight,
	); err != nil {
		return nil, err
	}

	if stripePriceID.Valid {
		option.StripePriceId = strings.TrimSpace(stripePriceID.String)
	} else {
		option.StripePriceId = ""
	}

	option.Kind = mapPlanKind(rawKind)
	option.BillingInterval = mapBillingInterval(rawInterval)
	option.IntroType = mapIntroPricingType(rawIntroType)
	if introAmount.Valid {
		option.IntroAmountCents = proto.Int64(introAmount.Int64)
	}
	if rawIntroLookup.Valid {
		option.IntroPriceLookupKey = rawIntroLookup.String
	}

	if len(metadataBytes) > 0 {
		if meta := parseMetadata(metadataBytes); meta != nil {
			option.Metadata = meta
		}
	}

	return option, nil
}

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
