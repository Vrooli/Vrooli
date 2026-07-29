package main

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/types/known/structpb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

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

func buildPlanMetadata(subtitle, badge, ctaLabel *string, highlight *bool, features []string) map[string]*commonv1.JsonValue {
	metadata := make(map[string]*commonv1.JsonValue)

	if subtitle != nil {
		if trimmed := strings.TrimSpace(*subtitle); trimmed != "" {
			metadata["subtitle"] = newStringJsonValue(trimmed)
		}
	}
	if badge != nil {
		if trimmed := strings.TrimSpace(*badge); trimmed != "" {
			metadata["badge"] = newStringJsonValue(trimmed)
		}
	}
	if ctaLabel != nil {
		if trimmed := strings.TrimSpace(*ctaLabel); trimmed != "" {
			metadata["cta_label"] = newStringJsonValue(trimmed)
		}
	}
	if highlight != nil && *highlight {
		metadata["highlight"] = newBoolJsonValue(true)
	}

	var sanitized []string
	for _, feature := range features {
		if trimmed := strings.TrimSpace(feature); trimmed != "" {
			sanitized = append(sanitized, trimmed)
		}
	}
	if len(sanitized) > 0 {
		listValues := make([]*commonv1.JsonValue, 0, len(sanitized))
		for _, feature := range sanitized {
			listValues = append(listValues, newStringJsonValue(feature))
		}
		metadata["features"] = newListJsonValue(listValues)
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func ensureStripePriceMatchesBundle(bundle *BundleProduct, stripeDetails *StripePriceImport) error {
	if stripeDetails == nil {
		return nil
	}
	if bundle == nil {
		return fmt.Errorf("bundle not configured")
	}

	bundleProductID := strings.TrimSpace(bundle.StripeProductId)
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

func mapPlanKind(kind string) shared.PlanKind {
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

var allowedPlanTiers = map[string]int32{
	"free":     0,
	"solo":     1,
	"pro":      2,
	"studio":   3,
	"business": 4,
	"credits":  5,
	"donation": 6,
}

func normalizePlanTier(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", fmt.Errorf("plan_tier is required")
	}
	if _, ok := allowedPlanTiers[value]; !ok {
		return "", fmt.Errorf("unsupported plan_tier: %s", value)
	}
	return value, nil
}

func planKindForTier(tier string) shared.PlanKind {
	switch tier {
	case "credits":
		return shared.PlanKind_PLAN_KIND_CREDITS_TOPUP
	case "donation":
		return shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION
	default:
		return shared.PlanKind_PLAN_KIND_SUBSCRIPTION
	}
}

func derivePlanTierFromStripe(price *StripePriceImport) (string, bool) {
	if price == nil {
		return "", false
	}
	candidates := []string{
		price.LookupKey,
		price.ProductName,
		price.PriceID,
	}

	for _, candidate := range candidates {
		if tier, ok := detectTierToken(candidate); ok {
			return tier, true
		}
	}

	return "", false
}

func detectTierToken(source string) (string, bool) {
	normalized := strings.ToLower(source)
	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if _, ok := allowedPlanTiers[token]; ok {
			return token, true
		}
	}
	return "", false
}

func planNameFromStripeImport(price *StripePriceImport) string {
	if price == nil {
		return ""
	}
	name := strings.TrimSpace(price.ProductName)
	if name == "" {
		name = strings.TrimSpace(price.LookupKey)
	}
	if name == "" {
		name = strings.TrimSpace(price.PriceID)
	}
	return name
}

// stripePriceImportToPlanOption converts a StripePriceImport to a PlanOption.
func stripePriceImportToPlanOption(price *StripePriceImport) *PlanOption {
	interval := mapBillingInterval(price.Interval)
	if interval == shared.BillingInterval_BILLING_INTERVAL_UNSPECIFIED {
		interval = shared.BillingInterval_BILLING_INTERVAL_ONE_TIME
	}

	planTier, ok := derivePlanTierFromStripe(price)
	if !ok {
		planTier = "pro"
	}
	planName := planNameFromStripeImport(price)

	return &PlanOption{
		StripePriceId:   price.PriceID,
		PlanName:        planName,
		PlanTier:        planTier,
		BillingInterval: interval,
		AmountCents:     price.AmountCents,
		Currency:        price.Currency,
		DisplayEnabled:  price.Active,
		DisplayWeight:   10,
		PlanRank:        planRankForTier(planTier),
		Kind:            planKindForTier(planTier),
	}
}

func normalizeCurrency(raw string) (string, error) {
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

func normalizePlanName(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("plan_name is required")
	}
	return value, nil
}

func normalizeStripePriceID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("stripe_price_id is required")
	}
	if !strings.HasPrefix(value, "price_") {
		return "", fmt.Errorf("stripe_price_id must start with price_")
	}
	return value, nil
}

func validateBillingInterval(interval shared.BillingInterval) error {
	if interval == shared.BillingInterval_BILLING_INTERVAL_UNSPECIFIED {
		return fmt.Errorf("billing_interval is required")
	}
	return nil
}

func planRankForTier(tier string) int32 {
	if rank, ok := allowedPlanTiers[tier]; ok {
		return rank
	}
	return 0
}

func planKindString(kind shared.PlanKind) string {
	switch kind {
	case shared.PlanKind_PLAN_KIND_CREDITS_TOPUP:
		return "credits_topup"
	case shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION:
		return "supporter_contribution"
	case shared.PlanKind_PLAN_KIND_SUBSCRIPTION:
		return "subscription"
	default:
		return "subscription"
	}
}

func mapBillingInterval(raw string) shared.BillingInterval {
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

func billingIntervalLabel(interval shared.BillingInterval) string {
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
