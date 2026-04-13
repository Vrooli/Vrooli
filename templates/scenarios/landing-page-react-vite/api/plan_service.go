package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	var monthly, yearly []*PlanOption
	for _, price := range prices {
		if !price.GetDisplayEnabled() {
			continue
		}
		switch price.BillingInterval {
		case "month":
			monthly = append(monthly, proto.Clone(price).(*PlanOption))
		case "year":
			yearly = append(yearly, proto.Clone(price).(*PlanOption))
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
	option := &PlanOption{}
	var metadataBytes []byte
	var rawKind string
	var introAmount sql.NullInt64
	err := row.Scan(
		&option.StripePriceId,
		&option.PlanName,
		&option.PlanTier,
		&option.BillingInterval,
		&option.AmountCents,
		&option.Currency,
		&option.IntroEnabled,
		&option.IntroType,
		&introAmount,
		&option.IntroPeriods,
		&option.IntroPriceLookupKey,
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
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("price %s not found", priceID)
	}
	if err != nil {
		return nil, err
	}

	if introAmount.Valid {
		val := introAmount.Int64
		option.IntroAmountCents = proto.Int64(val)
	}

	if len(metadataBytes) > 0 {
		if meta := parseMetadata(metadataBytes); meta != nil {
			option.Metadata = meta
		}
	}

	option.Kind = mapPlanKind(rawKind)

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
		SELECT bp.stripe_price_id, bp.plan_name, bp.plan_tier, bp.billing_interval,
		       bp.amount_cents, bp.currency, bp.intro_enabled, bp.intro_type, bp.intro_amount_cents,
		       bp.intro_periods, bp.intro_price_lookup_key, bp.monthly_included_credits,
		       bp.one_time_bonus_credits, bp.plan_rank, bp.bonus_type, bp.kind, bp.is_variable_amount,
		       bp.display_enabled, b.bundle_key, bp.metadata, bp.display_weight
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.product_id = $1
		ORDER BY bp.display_weight DESC, bp.plan_rank ASC
	`

	rows, err := s.db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*PlanOption
	for rows.Next() {
		var option PlanOption
		var metadataBytes []byte
		var introAmount sql.NullInt64
		var rawKind string
		if err := rows.Scan(
			&option.StripePriceId,
			&option.PlanName,
			&option.PlanTier,
			&option.BillingInterval,
			&option.AmountCents,
			&option.Currency,
			&option.IntroEnabled,
			&option.IntroType,
			&introAmount,
			&option.IntroPeriods,
			&option.IntroPriceLookupKey,
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

		if introAmount.Valid {
			val := introAmount.Int64
			option.IntroAmountCents = proto.Int64(val)
		}

		if len(metadataBytes) > 0 {
			if meta := parseMetadata(metadataBytes); meta != nil {
				option.Metadata = meta
			}
		}

		option.Kind = mapPlanKind(rawKind)

		copied := option
		options = append(options, &copied)
	}

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
			return nil, fmt.Errorf("price %s not found for bundle %s", priceID, bundleKey)
		}
		return nil, err
	}

	metadata := parseMetadata(metadataBytes)
	if metadata == nil {
		metadata = map[string]*structpb.Value{}
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
		metadata[key] = structpb.NewStringValue(trimmed)
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
			listValues := make([]*structpb.Value, 0, len(sanitized))
			for _, feature := range sanitized {
				listValues = append(listValues, structpb.NewStringValue(feature))
			}
			metadata["features"] = structpb.NewListValue(&structpb.ListValue{Values: listValues})
		}
	}

	setMetadataString("subtitle", input.Subtitle)
	setMetadataString("badge", input.Badge)
	setMetadataString("cta_label", input.CtaLabel)
	if input.Highlight != nil {
		if *input.Highlight {
			metadata["highlight"] = structpb.NewBoolValue(true)
		} else {
			delete(metadata, "highlight")
		}
	}

	metadataJSON, err := json.Marshal((&structpb.Struct{Fields: metadata}).AsMap())
	if err != nil {
		return nil, fmt.Errorf("marshal price metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE bundle_prices
		SET plan_name = COALESCE($1, plan_name),
		    display_weight = COALESCE($2, display_weight),
		    display_enabled = COALESCE($3, display_enabled),
		    metadata = $4,
		    updated_at = NOW()
		WHERE id = $5
	`, input.PlanName, input.DisplayWeight, input.DisplayEnabled, metadataJSON, pricePrimaryID)
	if err != nil {
		return nil, err
	}

	return s.GetPlanByPriceID(priceID)
}

func parseMetadata(metadataBytes []byte) map[string]*structpb.Value {
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

	structVal, err := structpb.NewStruct(meta)
	if err != nil {
		logStructured("plan metadata structpb conversion failed", map[string]interface{}{
			"level": "warn",
			"error": err.Error(),
		})
		return nil
	}

	return structVal.Fields
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
