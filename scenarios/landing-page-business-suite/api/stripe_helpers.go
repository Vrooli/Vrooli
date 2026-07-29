package main

import (
	"strings"

	"landing-page-business-suite-api/internal/commerce"
)

// PlanResolution contains the resolved plan tier and bundle key.
type PlanResolution struct {
	PlanTier  string
	BundleKey string
	PriceID   string
}

// ResolvePlanFromMetadata extracts plan tier and bundle key from subscription metadata.
// This consolidates the pattern used in persistSubscriptionFromStripe, persistInvoiceStatus,
// and VerifySubscription.
func ResolvePlanFromMetadata(metadata map[string]interface{}, defaultBundleKey string) PlanResolution {
	result := PlanResolution{
		BundleKey: defaultBundleKey,
	}

	if planTierVal, ok := metadata["plan_tier"].(string); ok && planTierVal != "" {
		result.PlanTier = planTierVal
	}
	if bundleVal, ok := metadata["bundle_key"].(string); ok && bundleVal != "" {
		result.BundleKey = bundleVal
	}
	if priceVal, ok := metadata["price_id"].(string); ok && priceVal != "" {
		result.PriceID = priceVal
	}

	return result
}

// ResolvePlanFromPriceID looks up plan details from a price ID using the plan store.
// Returns updated resolution with plan tier and bundle key from the plan store.
func (s *StripeService) ResolvePlanFromPriceID(priceID string, current PlanResolution) PlanResolution {
	if priceID == "" {
		return current
	}

	current.PriceID = priceID

	plan, err := s.planService.GetPlanByPriceID(priceID)
	if err != nil {
		logStructured("stripe_plan_lookup_failed", map[string]interface{}{
			"level":    "warn",
			"price_id": priceID,
			"error":    err.Error(),
		})
		return current
	}

	if plan.PlanTier != "" {
		current.PlanTier = plan.PlanTier
	}
	if plan.BundleKey != "" {
		current.BundleKey = plan.BundleKey
	}

	return current
}

// InferPlanTierFromPriceID attempts to detect the plan tier from the price ID string.
// Falls back to tier token detection if direct lookup fails.
func InferPlanTierFromPriceID(priceID string, currentTier string) string {
	if strings.TrimSpace(currentTier) != "" {
		return currentTier
	}
	if strings.TrimSpace(priceID) == "" {
		return currentTier
	}
	if inferred, ok := commerce.DetectTierToken(priceID); ok {
		return inferred
	}
	return currentTier
}

// ValidatePlanTier ensures the plan tier is valid, returning empty string if invalid.
func ValidatePlanTier(planTier, priceID, subscriptionID string) string {
	if strings.TrimSpace(planTier) == "" {
		return ""
	}
	if _, err := commerce.NormalizePlanTier(planTier); err != nil {
		logStructured("stripe_subscription_plan_tier_invalid", map[string]interface{}{
			"level":        "warn",
			"plan_tier":    planTier,
			"price_id":     priceID,
			"subscription": subscriptionID,
		})
		return ""
	}
	return planTier
}

// FullPlanResolution performs the complete plan resolution process:
// 1. Extract from metadata
// 2. Look up from price ID
// 3. Infer from price ID tokens
// 4. Validate
func (s *StripeService) FullPlanResolution(metadata map[string]interface{}, priceID, subscriptionID string) PlanResolution {
	// Start with metadata
	result := ResolvePlanFromMetadata(metadata, s.planService.BundleKey())

	// Look up from price ID
	if priceID != "" {
		result = s.ResolvePlanFromPriceID(priceID, result)
	}

	// Infer from price ID if still empty
	result.PlanTier = InferPlanTierFromPriceID(result.PriceID, result.PlanTier)

	// Validate
	result.PlanTier = ValidatePlanTier(result.PlanTier, result.PriceID, subscriptionID)

	return result
}

// ExtractCouponIDFromDiscount extracts a coupon ID from a discount map structure.
// This consolidates the nested extraction logic used in extractIntroCouponFromInvoice.
func ExtractCouponIDFromDiscount(discount map[string]interface{}) string {
	if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
		if couponID, ok := coupon["id"].(string); ok {
			return couponID
		}
	}
	return ""
}

// ExtractCouponFromDiscountAmount extracts a coupon ID from a total_discount_amounts entry.
func ExtractCouponFromDiscountAmount(tda map[string]interface{}) string {
	if discount, ok := tda["discount"].(map[string]interface{}); ok {
		return ExtractCouponIDFromDiscount(discount)
	}
	return ""
}

// ExtractSubscriptionItem extracts the first item from a Stripe subscription items structure.
type SubscriptionItemData struct {
	PriceID string
}

// ExtractFirstSubscriptionItem extracts the price ID from the first subscription item.
func ExtractFirstSubscriptionItem(sub *stripeSubscription) string {
	if len(sub.Items.Data) == 0 {
		return ""
	}
	return sub.Items.Data[0].Price.ID
}

// SafeStringFromMap safely extracts a string value from a map.
func SafeStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SafeInt64FromMap safely extracts an int64 value from a map.
// Handles both float64 (from JSON) and int types.
func SafeInt64FromMap(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

// SafeMapFromMap safely extracts a nested map from a map.
func SafeMapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// SafeArrayFromMap safely extracts an array from a map.
func SafeArrayFromMap(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}

// SubscriptionInsertParams holds parameters for subscription upsert operations.
type SubscriptionInsertParams struct {
	SubscriptionID    string
	CustomerID        string
	CustomerEmail     string
	Status            string
	PlanTier          string
	PriceID           string
	BundleKey         string
	BillingCycleStart int
	CanceledAt        interface{} // *time.Time or nil
}

// BuildSubscriptionUpsertSQL returns the SQL for upserting a subscription.
// This consolidates the 3+ variations of subscription persistence SQL.
func BuildSubscriptionUpsertSQL() string {
	return `
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, billing_cycle_start, canceled_at, created_at, updated_at)
		VALUES ($1::varchar,$2::varchar,$3::varchar,$4::varchar,$5::varchar,$6::varchar,$7::varchar,$8::int,$9::timestamp,COALESCE((SELECT created_at FROM subscriptions WHERE subscription_id = $1::varchar), NOW()), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET
			customer_id = EXCLUDED.customer_id,
			customer_email = EXCLUDED.customer_email,
			status = EXCLUDED.status,
			plan_tier = COALESCE(NULLIF(EXCLUDED.plan_tier,''), subscriptions.plan_tier),
			price_id = COALESCE(NULLIF(EXCLUDED.price_id,''), subscriptions.price_id),
			bundle_key = COALESCE(NULLIF(EXCLUDED.bundle_key,''), subscriptions.bundle_key),
			billing_cycle_start = EXCLUDED.billing_cycle_start,
			canceled_at = EXCLUDED.canceled_at,
			updated_at = NOW()
	`
}

// CoalesceString returns the first non-empty string from the provided values.
func CoalesceString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ChooseUserIdentity determines the best user identity to return.
// Prefers userHint, then customer email, then customer ID.
func ChooseUserIdentityFromSubscription(userHint string, sub *stripeSubscription) string {
	if strings.TrimSpace(userHint) != "" {
		return strings.TrimSpace(userHint)
	}
	if sub.CustomerEmail != "" {
		return sub.CustomerEmail
	}
	return sub.Customer
}
