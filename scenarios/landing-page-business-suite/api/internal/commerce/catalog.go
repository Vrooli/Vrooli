// Package commerce owns subscription, credit, and Stripe catalog business rules.
package commerce

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var allowedPlanTiers = map[string]int32{
	"free": 0, "solo": 1, "pro": 2, "studio": 3, "business": 4, "credits": 5, "donation": 6,
}

func EnsureStripePriceMatchesBundle(bundle *shared.Bundle, price *StripePriceImport) error {
	if price == nil {
		return nil
	}
	if bundle == nil {
		return fmt.Errorf("bundle not configured")
	}
	bundleProductID := strings.TrimSpace(bundle.StripeProductId)
	if bundleProductID == "" {
		return fmt.Errorf("bundle stripe_product_id is required")
	}
	priceProductID := strings.TrimSpace(price.ProductID)
	if priceProductID == "" {
		return fmt.Errorf("stripe price %s missing product_id", strings.TrimSpace(price.PriceID))
	}
	if priceProductID != bundleProductID {
		return fmt.Errorf("stripe price %s belongs to product %s (expected %s)", price.PriceID, priceProductID, bundleProductID)
	}
	return nil
}

func MapPlanKind(kind string) shared.PlanKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "subscription":
		return shared.PlanKind_PLAN_KIND_SUBSCRIPTION
	case "credits_topup", "credits-topup", "credits":
		return shared.PlanKind_PLAN_KIND_CREDITS_TOPUP
	case "supporter_contribution", "supporter-contribution", "supporter":
		return shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION
	default:
		return shared.PlanKind_PLAN_KIND_UNSPECIFIED
	}
}

func NormalizePlanTier(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", fmt.Errorf("plan_tier is required")
	}
	if _, ok := allowedPlanTiers[value]; !ok {
		return "", fmt.Errorf("unsupported plan_tier: %s", value)
	}
	return value, nil
}

func PlanKindForTier(tier string) shared.PlanKind {
	switch tier {
	case "credits":
		return shared.PlanKind_PLAN_KIND_CREDITS_TOPUP
	case "donation":
		return shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION
	default:
		return shared.PlanKind_PLAN_KIND_SUBSCRIPTION
	}
}

func DerivePlanTierFromStripe(price *StripePriceImport) (string, bool) {
	if price == nil {
		return "", false
	}
	for _, candidate := range []string{price.LookupKey, price.ProductName, price.PriceID} {
		if tier, ok := DetectTierToken(candidate); ok {
			return tier, true
		}
	}
	return "", false
}

func DetectTierToken(source string) (string, bool) {
	tokens := strings.FieldsFunc(strings.ToLower(source), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	for _, token := range tokens {
		if _, ok := allowedPlanTiers[token]; ok {
			return token, true
		}
	}
	return "", false
}

func PlanNameFromStripeImport(price *StripePriceImport) string {
	if price == nil {
		return ""
	}
	for _, candidate := range []string{price.ProductName, price.LookupKey, price.PriceID} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	return ""
}

func StripePriceImportToPlanOption(price *StripePriceImport) *shared.PlanOption {
	interval := MapBillingInterval(price.Interval)
	if interval == shared.BillingInterval_BILLING_INTERVAL_UNSPECIFIED {
		interval = shared.BillingInterval_BILLING_INTERVAL_ONE_TIME
	}
	tier, ok := DerivePlanTierFromStripe(price)
	if !ok {
		tier = "pro"
	}
	return &shared.PlanOption{StripePriceId: price.PriceID, PlanName: PlanNameFromStripeImport(price), PlanTier: tier, BillingInterval: interval, AmountCents: price.AmountCents, Currency: price.Currency, DisplayEnabled: price.Active, DisplayWeight: 10, PlanRank: PlanRankForTier(tier), Kind: PlanKindForTier(tier)}
}

func NormalizeCurrency(raw string) (string, error) {
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

func NormalizePlanName(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("plan_name is required")
	}
	return value, nil
}

func NormalizeStripePriceID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("stripe_price_id is required")
	}
	if !strings.HasPrefix(value, "price_") {
		return "", fmt.Errorf("stripe_price_id must start with price_")
	}
	return value, nil
}

func ValidateBillingInterval(interval shared.BillingInterval) error {
	if interval == shared.BillingInterval_BILLING_INTERVAL_UNSPECIFIED {
		return fmt.Errorf("billing_interval is required")
	}
	return nil
}

func PlanRankForTier(tier string) int32 { return allowedPlanTiers[tier] }

func PlanKindString(kind shared.PlanKind) string {
	switch kind {
	case shared.PlanKind_PLAN_KIND_CREDITS_TOPUP:
		return "credits_topup"
	case shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION:
		return "supporter_contribution"
	default:
		return "subscription"
	}
}

func MapBillingInterval(raw string) shared.BillingInterval {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "month", "monthly", "m":
		return shared.BillingInterval_BILLING_INTERVAL_MONTH
	case "year", "yearly", "y":
		return shared.BillingInterval_BILLING_INTERVAL_YEAR
	case "one_time", "one-time", "one time", "onetime", "ot":
		return shared.BillingInterval_BILLING_INTERVAL_ONE_TIME
	default:
		return shared.BillingInterval_BILLING_INTERVAL_UNSPECIFIED
	}
}

func BillingIntervalLabel(interval shared.BillingInterval) string {
	switch interval {
	case shared.BillingInterval_BILLING_INTERVAL_MONTH:
		return "month"
	case shared.BillingInterval_BILLING_INTERVAL_YEAR:
		return "year"
	case shared.BillingInterval_BILLING_INTERVAL_ONE_TIME:
		return "one_time"
	default:
		return "unspecified"
	}
}

func MapIntroPricingType(raw string) shared.IntroPricingType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "percentage", "percent", "pct":
		return shared.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE
	case "flat_amount", "flat-amount", "flat", "amount":
		return shared.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT
	default:
		return shared.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED
	}
}

func IntroPricingTypeString(pricingType shared.IntroPricingType) string {
	switch pricingType {
	case shared.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE:
		return "percentage"
	case shared.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT:
		return "flat_amount"
	default:
		return ""
	}
}

func ValidatePlanTierConstraints(plan *shared.PlanOption) error {
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

// NormalizeBundle canonicalizes and validates a bundle loaded from catalog storage.
func NormalizeBundle(bundle *shared.Bundle, bundleKeyFallback, environmentFallback string) error {
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
		bundle.Environment = strings.TrimSpace(environmentFallback)
		if bundle.Environment == "" {
			bundle.Environment = "production"
		}
	}
	return nil
}

// NormalizePlanOption canonicalizes and validates a plan for its owning bundle.
func NormalizePlanOption(plan *shared.PlanOption, bundleKey string) error {
	if plan == nil {
		return fmt.Errorf("plan is required")
	}
	priceID, err := NormalizeStripePriceID(plan.StripePriceId)
	if err != nil {
		return err
	}
	plan.StripePriceId = priceID
	name, err := NormalizePlanName(plan.PlanName)
	if err != nil {
		return err
	}
	plan.PlanName = name
	tier, err := NormalizePlanTier(plan.PlanTier)
	if err != nil {
		return err
	}
	plan.PlanTier = tier
	if err := ValidateBillingInterval(plan.BillingInterval); err != nil {
		return err
	}
	currency, err := NormalizeCurrency(plan.Currency)
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
	expectedKind := PlanKindForTier(plan.PlanTier)
	if plan.Kind == shared.PlanKind_PLAN_KIND_UNSPECIFIED {
		plan.Kind = expectedKind
	} else if plan.Kind != expectedKind {
		return fmt.Errorf("plan_kind %s does not match plan_tier %s", PlanKindString(plan.Kind), plan.PlanTier)
	}
	plan.BundleKey = bundleKey
	return ValidatePlanTierConstraints(plan)
}

// ApplyStripePriceDetails synchronizes verified Stripe attributes onto a plan.
func ApplyStripePriceDetails(plan *shared.PlanOption, input UpdateBundlePriceInput, details *StripePriceImport) error {
	if details == nil {
		return nil
	}
	interval := MapBillingInterval(details.Interval)
	if err := ValidateBillingInterval(interval); err != nil {
		return fmt.Errorf("invalid stripe billing interval: %w", err)
	}
	currency, err := NormalizeCurrency(details.Currency)
	if err != nil {
		return fmt.Errorf("invalid stripe currency: %w", err)
	}
	if details.AmountCents < 0 {
		return fmt.Errorf("stripe amount_cents must be >= 0")
	}
	plan.AmountCents, plan.Currency, plan.BillingInterval = details.AmountCents, currency, interval
	if input.DisplayEnabled == nil && !details.Active {
		plan.DisplayEnabled = false
	}
	return nil
}

// ApplyDerivedPlanTier applies Stripe-derived tier semantics when supplied.
func ApplyDerivedPlanTier(plan *shared.PlanOption, derivedTier string) error {
	if strings.TrimSpace(derivedTier) == "" {
		return nil
	}
	tier, err := NormalizePlanTier(derivedTier)
	if err != nil {
		return err
	}
	if tier != plan.PlanTier {
		plan.PlanTier, plan.PlanRank, plan.Kind = tier, PlanRankForTier(tier), PlanKindForTier(tier)
	}
	return nil
}

// ApplyPlanMetadata updates only operator-editable metadata fields.
func ApplyPlanMetadata(plan *shared.PlanOption, input UpdateBundlePriceInput) {
	if plan.Metadata == nil {
		plan.Metadata = map[string]*commonv1.JsonValue{}
	}
	setString := func(key string, value *string) {
		if value == nil {
			return
		}
		if trimmed := strings.TrimSpace(*value); trimmed == "" {
			delete(plan.Metadata, key)
		} else {
			plan.Metadata[key] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: trimmed}}
		}
	}
	setString("subtitle", input.Subtitle)
	setString("badge", input.Badge)
	setString("cta_label", input.CtaLabel)
	if input.Highlight != nil {
		if *input.Highlight {
			plan.Metadata["highlight"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: true}}
		} else {
			delete(plan.Metadata, "highlight")
		}
	}
	if input.Features != nil {
		values := make([]*commonv1.JsonValue, 0, len(*input.Features))
		for _, feature := range *input.Features {
			if trimmed := strings.TrimSpace(feature); trimmed != "" {
				values = append(values, &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: trimmed}})
			}
		}
		if len(values) == 0 {
			delete(plan.Metadata, "features")
		} else {
			plan.Metadata["features"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: &commonv1.JsonList{Values: values}}}
		}
	}
}

// ValidateUpdatedPlan validates a partially updated plan before store persistence.
func ValidateUpdatedPlan(plan *shared.PlanOption) error {
	if plan.Kind == shared.PlanKind_PLAN_KIND_UNSPECIFIED {
		plan.Kind = PlanKindForTier(plan.PlanTier)
	}
	tier, err := NormalizePlanTier(plan.PlanTier)
	if err != nil {
		return err
	}
	plan.PlanTier = tier
	if _, err := NormalizePlanName(plan.PlanName); err != nil {
		return err
	}
	if err := ValidateBillingInterval(plan.BillingInterval); err != nil {
		return err
	}
	currency, err := NormalizeCurrency(plan.Currency)
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
	return ValidatePlanTierConstraints(plan)
}

// BuildPricingOverview projects a bundle catalog into the public pricing view.
func BuildPricingOverview(bundle *shared.Bundle, plans []*shared.PlanOption) (*shared.PricingOverview, error) {
	if bundle == nil {
		return nil, fmt.Errorf("bundle not configured")
	}
	monthly, yearly := make([]*shared.PlanOption, 0), make([]*shared.PlanOption, 0)
	for _, plan := range plans {
		if !plan.DisplayEnabled && strings.ToLower(strings.TrimSpace(plan.PlanTier)) != "free" {
			continue
		}
		switch plan.BillingInterval {
		case shared.BillingInterval_BILLING_INTERVAL_MONTH:
			monthly = append(monthly, proto.Clone(plan).(*shared.PlanOption))
		case shared.BillingInterval_BILLING_INTERVAL_YEAR:
			yearly = append(yearly, proto.Clone(plan).(*shared.PlanOption))
		}
	}
	sortPlanOptions(monthly)
	sortPlanOptions(yearly)
	return &shared.PricingOverview{Bundle: proto.Clone(bundle).(*shared.Bundle), Monthly: monthly, Yearly: yearly, UpdatedAt: timestamppb.Now()}, nil
}

// BuildBundleCatalog projects every catalog price for administrative operations.
func BuildBundleCatalog(bundle *shared.Bundle, plans []*shared.PlanOption) []BundleCatalogEntry {
	if bundle == nil {
		return []BundleCatalogEntry{}
	}
	prices := make([]*shared.PlanOption, 0, len(plans))
	for _, plan := range plans {
		prices = append(prices, proto.Clone(plan).(*shared.PlanOption))
	}
	sortPlanOptions(prices)
	return []BundleCatalogEntry{{Bundle: proto.Clone(bundle).(*shared.Bundle), Prices: prices}}
}

func sortPlanOptions(plans []*shared.PlanOption) {
	sort.SliceStable(plans, func(i, j int) bool {
		if plans[i].DisplayWeight == plans[j].DisplayWeight {
			return plans[i].PlanRank < plans[j].PlanRank
		}
		return plans[i].DisplayWeight > plans[j].DisplayWeight
	})
}
