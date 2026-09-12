// Package plan exposes pricing/plan metadata: the public pricing overview
// (monthly/yearly display-enabled plans for the configured bundle) and the
// admin bundle catalog (all bundles and prices, editable). Its domain types are
// the shared protobuf pricing messages, so the handlers in handlers/payments and
// handlers/bundles are thin adapters over this Service.
package plan

import (
	"context"
	"database/sql"
	"fmt"
	"landing-page-react-vite-api/internal/jsonval"
	"os"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// Domain type aliases onto the shared protobuf pricing messages.
type (
	BundleProduct   = landingv1.Bundle
	PlanOption      = landingv1.PlanOption
	PricingOverview = landingv1.PricingOverview
)

// Service exposes pricing/plan helpers over the bundle tables.
type Service struct {
	db            *sql.DB
	defaultBundle string
	displayEnv    string
}

type bundleProductRecord struct {
	ID     int64
	Bundle *BundleProduct
}

// NewService constructs the plan Service. The default bundle key and display
// environment are read from BUNDLE_KEY / BUNDLE_ENVIRONMENT.
func NewService(db *sql.DB) *Service {
	return &Service{
		db:            db,
		defaultBundle: trimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite"),
		displayEnv:    trimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production"),
	}
}

func trimOrDefault(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

// BundleKey returns the configured default bundle key.
func (s *Service) BundleKey() string { return s.defaultBundle }

// GetPricingOverview loads product + display-enabled prices for the default
// bundle, split into weighted monthly/yearly plan lists.
func (s *Service) GetPricingOverview() (*PricingOverview, error) {
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
		case landingv1.BillingInterval_BILLING_INTERVAL_MONTH:
			monthly = append(monthly, proto.Clone(price).(*PlanOption))
		case landingv1.BillingInterval_BILLING_INTERVAL_YEAR:
			yearly = append(yearly, proto.Clone(price).(*PlanOption))
		}
	}
	sortPlans(monthly)
	sortPlans(yearly)

	return &PricingOverview{
		Bundle:    product.Bundle,
		Monthly:   monthly,
		Yearly:    yearly,
		UpdatedAt: timestamppb.Now(),
	}, nil
}

func sortPlans(plans []*PlanOption) {
	sort.SliceStable(plans, func(i, j int) bool {
		if plans[i].DisplayWeight == plans[j].DisplayWeight {
			return plans[i].PlanRank < plans[j].PlanRank
		}
		return plans[i].DisplayWeight > plans[j].DisplayWeight
	})
}

// GetPlanByPriceID fetches a plan option for a Stripe price identifier.
func (s *Service) GetPlanByPriceID(priceID string) (*PlanOption, error) {
	if priceID == "" {
		return nil, fmt.Errorf("price id is required")
	}
	option, err := scanPlanOption(s.db.QueryRow(planSelect+` WHERE bp.stripe_price_id = $1`, priceID))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("price %s not found", priceID)
	}
	if err != nil {
		return nil, err
	}
	return option, nil
}

// GetBundleProduct returns the configured bundle product metadata.
func (s *Service) GetBundleProduct() (*BundleProduct, error) {
	rec, err := s.loadBundleProduct(s.defaultBundle)
	if err != nil {
		return nil, err
	}
	return rec.Bundle, nil
}

const productSelect = `
	SELECT id, bundle_key, bundle_name, stripe_product_id, credits_per_usd,
	       display_credits_multiplier, display_credits_label, environment, metadata
	FROM bundle_products`

const planSelect = `
	SELECT bp.stripe_price_id, bp.plan_name, bp.plan_tier, bp.billing_interval,
	       bp.amount_cents, bp.currency, bp.intro_enabled, bp.intro_type,
	       bp.intro_amount_cents, bp.intro_periods, bp.intro_price_lookup_key,
	       bp.monthly_included_credits, bp.one_time_bonus_credits, bp.plan_rank,
	       bp.bonus_type, bp.kind, bp.is_variable_amount, bp.display_enabled, b.bundle_key,
	       bp.metadata, bp.display_weight
	FROM bundle_prices bp
	JOIN bundle_products b ON bp.product_id = b.id`

func (s *Service) loadBundleProduct(bundleKey string) (*bundleProductRecord, error) {
	row := s.db.QueryRow(productSelect+` WHERE bundle_key = $1 AND environment = $2 LIMIT 1`, bundleKey, s.displayEnv)
	rec, err := scanBundleProduct(row)
	if err != nil {
		return nil, fmt.Errorf("bundle %s not found: %w", bundleKey, err)
	}
	return rec, nil
}

func scanBundleProduct(row interface{ Scan(...any) error }) (*bundleProductRecord, error) {
	var id int64
	product := &BundleProduct{}
	var metadataBytes []byte
	if err := row.Scan(&id, &product.BundleKey, &product.Name, &product.StripeProductId,
		&product.CreditsPerUsd, &product.DisplayCreditsMultiplier, &product.DisplayCreditsLabel,
		&product.Environment, &metadataBytes); err != nil {
		return nil, err
	}
	product.Metadata = jsonval.FromJSONB(metadataBytes)
	return &bundleProductRecord{ID: id, Bundle: product}, nil
}

func (s *Service) loadBundlePrices(productID int64) ([]*PlanOption, error) {
	rows, err := s.db.Query(planSelect+` WHERE bp.product_id = $1 ORDER BY bp.display_weight DESC, bp.plan_rank ASC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*PlanOption
	for rows.Next() {
		option, err := scanPlanOption(rows)
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, rows.Err()
}

func scanPlanOption(row interface{ Scan(...any) error }) (*PlanOption, error) {
	option := &PlanOption{}
	var metadataBytes []byte
	var rawKind, rawInterval, rawIntroType string
	var introAmount sql.NullInt64
	if err := row.Scan(
		&option.StripePriceId, &option.PlanName, &option.PlanTier, &rawInterval,
		&option.AmountCents, &option.Currency, &option.IntroEnabled, &rawIntroType,
		&introAmount, &option.IntroPeriods, &option.IntroPriceLookupKey,
		&option.MonthlyIncludedCredits, &option.OneTimeBonusCredits, &option.PlanRank,
		&option.BonusType, &rawKind, &option.IsVariableAmount, &option.DisplayEnabled,
		&option.BundleKey, &metadataBytes, &option.DisplayWeight,
	); err != nil {
		return nil, err
	}
	option.BillingInterval = BillingIntervalFromString(rawInterval)
	option.IntroType = IntroTypeFromString(rawIntroType)
	if introAmount.Valid {
		option.IntroAmountCents = proto.Int64(introAmount.Int64)
	}
	option.Metadata = jsonval.FromJSONB(metadataBytes)
	option.Kind = mapPlanKind(rawKind)
	return option, nil
}

// CatalogEntry groups a bundle with all of its prices (visible + hidden).
type CatalogEntry struct {
	Bundle *BundleProduct
	Prices []*PlanOption
}

// ListBundleCatalog returns bundles for the configured environment so the admin
// UI can toggle prices without raw SQL edits.
func (s *Service) ListBundleCatalog(ctx context.Context) ([]CatalogEntry, error) {
	rows, err := s.db.QueryContext(ctx, productSelect+` WHERE environment = $1 ORDER BY bundle_key ASC`, s.displayEnv)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*bundleProductRecord
	for rows.Next() {
		rec, err := scanBundleProduct(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	entries := make([]CatalogEntry, 0, len(records))
	for _, rec := range records {
		prices, err := s.loadBundlePrices(rec.ID)
		if err != nil {
			return nil, err
		}
		entries = append(entries, CatalogEntry{Bundle: rec.Bundle, Prices: prices})
	}
	return entries, nil
}

// UpdateBundlePriceInput contains editable display metadata for a price row.
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
func (s *Service) UpdateBundlePrice(ctx context.Context, bundleKey, priceID string, input UpdateBundlePriceInput) (*PlanOption, error) {
	if priceID == "" || bundleKey == "" {
		return nil, fmt.Errorf("bundle key and price id are required")
	}

	var pricePrimaryID int64
	var metadataBytes []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT bp.id, bp.metadata
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.stripe_price_id = $1 AND b.bundle_key = $2 AND b.environment = $3`,
		priceID, bundleKey, s.displayEnv).Scan(&pricePrimaryID, &metadataBytes)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("price %s not found for bundle %s", priceID, bundleKey)
	}
	if err != nil {
		return nil, err
	}

	metadata := jsonval.FromJSONB(metadataBytes)
	if metadata == nil {
		metadata = map[string]*commonv1.JsonValue{}
	}

	setString := func(key string, value *string) {
		if value == nil {
			return
		}
		if trimmed := strings.TrimSpace(*value); trimmed != "" {
			metadata[key] = jsonval.String(trimmed)
		} else {
			delete(metadata, key)
		}
	}

	if input.Features != nil {
		var sanitized []string
		for _, feature := range *input.Features {
			if trimmed := strings.TrimSpace(feature); trimmed != "" {
				sanitized = append(sanitized, trimmed)
			}
		}
		if len(sanitized) == 0 {
			delete(metadata, "features")
		} else {
			metadata["features"] = jsonval.StringList(sanitized)
		}
	}

	setString("subtitle", input.Subtitle)
	setString("badge", input.Badge)
	setString("cta_label", input.CtaLabel)
	if input.Highlight != nil {
		if *input.Highlight {
			metadata["highlight"] = jsonval.Bool(true)
		} else {
			delete(metadata, "highlight")
		}
	}

	metadataJSON, err := jsonval.ToJSONB(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal price metadata: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE bundle_prices
		SET plan_name = COALESCE($1, plan_name),
		    display_weight = COALESCE($2, display_weight),
		    display_enabled = COALESCE($3, display_enabled),
		    metadata = $4, updated_at = NOW()
		WHERE id = $5`,
		input.PlanName, input.DisplayWeight, input.DisplayEnabled, metadataJSON, pricePrimaryID); err != nil {
		return nil, err
	}
	return s.GetPlanByPriceID(priceID)
}

func mapPlanKind(kind string) landingv1.PlanKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "subscription":
		return landingv1.PlanKind_PLAN_KIND_SUBSCRIPTION
	case "credits_topup", "credits-topup", "credits":
		return landingv1.PlanKind_PLAN_KIND_CREDITS_TOPUP
	case "supporter_contribution", "supporter-contribution", "supporter":
		return landingv1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION
	default:
		return landingv1.PlanKind_PLAN_KIND_UNSPECIFIED
	}
}

// PlanKindString renders a PlanKind back to its stored string form.
func PlanKindString(kind landingv1.PlanKind) string {
	switch kind {
	case landingv1.PlanKind_PLAN_KIND_CREDITS_TOPUP:
		return "credits_topup"
	case landingv1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION:
		return "supporter_contribution"
	default:
		return "subscription"
	}
}

// BillingIntervalFromString maps a stored billing_interval to its enum.
func BillingIntervalFromString(s string) landingv1.BillingInterval {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "month":
		return landingv1.BillingInterval_BILLING_INTERVAL_MONTH
	case "year":
		return landingv1.BillingInterval_BILLING_INTERVAL_YEAR
	case "one_time", "one-time":
		return landingv1.BillingInterval_BILLING_INTERVAL_ONE_TIME
	default:
		return landingv1.BillingInterval_BILLING_INTERVAL_UNSPECIFIED
	}
}

// BillingIntervalString renders a BillingInterval back to its stored form.
func BillingIntervalString(bi landingv1.BillingInterval) string {
	switch bi {
	case landingv1.BillingInterval_BILLING_INTERVAL_MONTH:
		return "month"
	case landingv1.BillingInterval_BILLING_INTERVAL_YEAR:
		return "year"
	case landingv1.BillingInterval_BILLING_INTERVAL_ONE_TIME:
		return "one_time"
	default:
		return ""
	}
}

// IntroTypeFromString maps a stored intro_type to its enum.
func IntroTypeFromString(s string) landingv1.IntroPricingType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "flat_amount", "flat-amount":
		return landingv1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT
	case "percentage":
		return landingv1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE
	default:
		return landingv1.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED
	}
}
