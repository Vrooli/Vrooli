package commerce

import "strings"

// PlanResolution is the domain result of reconciling Stripe metadata with the catalog.
type PlanResolution struct{ PlanTier, BundleKey, PriceID string }

func ResolvePlanFromMetadata(metadata map[string]interface{}, defaultBundleKey string) PlanResolution {
	result := PlanResolution{BundleKey: defaultBundleKey}
	if value, ok := metadata["plan_tier"].(string); ok && value != "" {
		result.PlanTier = value
	}
	if value, ok := metadata["bundle_key"].(string); ok && value != "" {
		result.BundleKey = value
	}
	if value, ok := metadata["price_id"].(string); ok && value != "" {
		result.PriceID = value
	}
	return result
}

func InferPlanTierFromPriceID(priceID, currentTier string) string {
	if strings.TrimSpace(currentTier) != "" || strings.TrimSpace(priceID) == "" {
		return currentTier
	}
	if inferred, ok := DetectTierToken(priceID); ok {
		return inferred
	}
	return currentTier
}

func ExtractCouponIDFromDiscount(discount map[string]interface{}) string {
	if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
		if id, ok := coupon["id"].(string); ok {
			return id
		}
	}
	return ""
}

func ExtractCouponFromDiscountAmount(amount map[string]interface{}) string {
	if discount, ok := amount["discount"].(map[string]interface{}); ok {
		return ExtractCouponIDFromDiscount(discount)
	}
	return ""
}

func SafeStringFromMap(values map[string]interface{}, key string) string {
	value, _ := values[key].(string)
	return value
}

func SafeInt64FromMap(values map[string]interface{}, key string) int64 {
	switch value := values[key].(type) {
	case float64:
		return int64(value)
	case int:
		return int64(value)
	case int64:
		return value
	}
	return 0
}

func SafeMapFromMap(values map[string]interface{}, key string) map[string]interface{} {
	value, _ := values[key].(map[string]interface{})
	return value
}

func SafeArrayFromMap(values map[string]interface{}, key string) []interface{} {
	value, _ := values[key].([]interface{})
	return value
}

func BuildSubscriptionUpsertSQL() string {
	return `
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, billing_cycle_start, canceled_at, created_at, updated_at)
		VALUES ($1::varchar,$2::varchar,$3::varchar,$4::varchar,$5::varchar,$6::varchar,$7::varchar,$8::int,$9::timestamp,COALESCE((SELECT created_at FROM subscriptions WHERE subscription_id = $1::varchar), NOW()), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET customer_id = EXCLUDED.customer_id, customer_email = EXCLUDED.customer_email, status = EXCLUDED.status, plan_tier = COALESCE(NULLIF(EXCLUDED.plan_tier,''), subscriptions.plan_tier), price_id = COALESCE(NULLIF(EXCLUDED.price_id,''), subscriptions.price_id), bundle_key = COALESCE(NULLIF(EXCLUDED.bundle_key,''), subscriptions.bundle_key), billing_cycle_start = EXCLUDED.billing_cycle_start, canceled_at = EXCLUDED.canceled_at, updated_at = NOW()
	`
}

func CoalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
