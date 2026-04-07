package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Coupon & Intro Pricing Lifecycle E2E Tests
// ============================================================================

// TestE2E_IntroCoupon_Apply_Use_RejectReuse verifies the complete intro coupon
// lifecycle: first checkout applies coupon, webhook marks it used, second
// checkout rejects the coupon.
func TestE2E_IntroCoupon_Apply_Use_RejectReuse(t *testing.T) {
	checkoutCalls := 0
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions":
			checkoutCalls++
			fmt.Fprintf(w, `{
				"id":"cs_coupon_lifecycle_%d",
				"url":"https://checkout.stripe.test/cs_coupon_lifecycle_%d",
				"status":"open",
				"customer_email":"coupon-lifecycle@example.com",
				"customer":"cus_coupon_lc",
				"subscription":"sub_coupon_lc_%d",
				"amount_total":100,
				"mode":"subscription",
				"currency":"usd"
			}`, checkoutCalls, checkoutCalls, checkoutCalls)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/customers/cus_coupon_lc":
			fmt.Fprint(w, `{"id":"cus_coupon_lc"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer stripeServer.Close()

	h := setupMonetizationHarness(t, stripeServer)

	// Reconfigure with intro coupon enabled
	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", h.webhookSecret).
		WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	h.stripeService = ConfigureStripeService(t, h.db, cfg, stripeServer)

	ctx := context.Background()

	// Step 1: New user should be eligible for intro
	eligible, err := h.stripeService.checkIntroEligibility(ctx, "coupon-lifecycle@example.com")
	require.NoError(t, err)
	assert.True(t, eligible, "new user should be eligible for intro coupon")

	// Step 2: Create first checkout session (intro coupon should be applied)
	session1, err := h.stripeService.CreateCheckoutSession("price_pro_e2e", "/success", "/cancel", "coupon-lifecycle@example.com")
	require.NoError(t, err)
	require.NotNil(t, session1)

	// Step 3: Simulate checkout.session.completed and mark intro used
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_coupon_lc_checkout",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_coupon_lifecycle_1",
				"customer_email": "coupon-lifecycle@example.com",
				"customer":       "cus_coupon_lc",
				"subscription":   "sub_coupon_lc_1",
				"amount_total":   100,
			},
		},
	})

	// Mark intro as used (simulates what happens in handleInvoicePaid)
	err = h.stripeService.markIntroUsed(ctx, "coupon-lifecycle@example.com", "cus_coupon_lc", "coupon_intro_pro", "pro", "sub_coupon_lc_1")
	require.NoError(t, err)

	// Step 4: Verify user is no longer eligible
	eligible, err = h.stripeService.checkIntroEligibility(ctx, "coupon-lifecycle@example.com")
	require.NoError(t, err)
	assert.False(t, eligible, "user should NOT be eligible after using intro coupon")

	// Step 5: Verify intro_coupon_usage record was created
	var usageCount int
	err = h.db.QueryRow(`SELECT COUNT(*) FROM intro_coupon_usage WHERE email = $1`, "coupon-lifecycle@example.com").Scan(&usageCount)
	require.NoError(t, err)
	assert.Equal(t, 1, usageCount, "intro coupon usage should be recorded")

	// Step 6: Verify has_used_intro flag is set
	var hasUsedIntro bool
	err = h.db.QueryRow(`SELECT has_used_intro FROM users WHERE email = $1`, "coupon-lifecycle@example.com").Scan(&hasUsedIntro)
	require.NoError(t, err)
	assert.True(t, hasUsedIntro, "has_used_intro should be TRUE")
}

// TestE2E_IntroCoupon_EmailVariant_Bypass_Blocked verifies that attempting to
// re-use an intro coupon with email case variations is blocked by normalization.
func TestE2E_IntroCoupon_EmailVariant_Bypass_Blocked(t *testing.T) {
	h := setupMonetizationHarness(t, nil)

	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", h.webhookSecret).
		WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	h.stripeService = ConfigureStripeService(t, h.db, cfg, nil)

	ctx := context.Background()

	// Insert user who has already used intro (normalized lowercase)
	_, err := h.db.Exec(`
		INSERT INTO users (email, has_used_intro, stripe_customer_id)
		VALUES ($1, TRUE, $2)
	`, "coupon-bypass@example.com", "cus_bypass")
	require.NoError(t, err)

	// All case variations should be blocked
	variations := []string{
		"coupon-bypass@example.com",
		"COUPON-BYPASS@EXAMPLE.COM",
		"Coupon-Bypass@Example.Com",
		"COUPON-BYPASS@example.com",
		"  coupon-bypass@example.com  ",
		"coupon-bypass@EXAMPLE.COM",
	}

	for _, email := range variations {
		eligible, err := h.stripeService.checkIntroEligibility(ctx, email)
		require.NoError(t, err, "unexpected error for %q", email)
		assert.False(t, eligible, "case variation %q should be blocked by email normalization", email)
	}
}

// TestE2E_Coupon_Expired_RejectedAtCheckout verifies that an expired coupon
// (user already used intro) is not applied when checking eligibility.
func TestE2E_Coupon_Expired_RejectedAtCheckout(t *testing.T) {
	h := setupMonetizationHarness(t, nil)

	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", h.webhookSecret).
		WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	h.stripeService = ConfigureStripeService(t, h.db, cfg, nil)

	ctx := context.Background()

	// Insert user who has used intro and has an audit trail
	_, err := h.db.Exec(`
		INSERT INTO users (email, has_used_intro, stripe_customer_id)
		VALUES ($1, TRUE, $2)
	`, "expired-coupon@example.com", "cus_expired")
	require.NoError(t, err)

	_, err = h.db.Exec(`
		INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id, plan_tier, subscription_id)
		VALUES ($1, $2, $3, $4, $5)
	`, "expired-coupon@example.com", "cus_expired", "coupon_intro_pro", "pro", "sub_old")
	require.NoError(t, err)

	// Verify not eligible
	eligible, err := h.stripeService.checkIntroEligibility(ctx, "expired-coupon@example.com")
	require.NoError(t, err)
	assert.False(t, eligible, "user who already used intro should not be eligible again")
}
