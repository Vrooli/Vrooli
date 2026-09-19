package account

import (
	"fmt"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/stripe"
	"landing-page-react-vite-api/internal/testutil/billingfix"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func TestSubscriptionCache(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema, stripe.Schema)
	_, err := db.Exec(`DELETE FROM subscriptions`)
	require.NoError(t, err)

	svc := NewService(db, plan.NewService(db))
	svc.cacheTTL = 40 * time.Millisecond

	const user, subID = "cache-test@example.com", "sub-cache-test"
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()
	`, subID, user, "active", "solo", "price_solo_monthly", svc.bundleKey)
	require.NoError(t, err)

	info, err := svc.GetSubscription(user)
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE, info.State)
	require.NotNil(t, info.SubscriptionId)
	require.Equal(t, subID, *info.SubscriptionId)

	_, err = db.Exec(`UPDATE subscriptions SET status = 'canceled', updated_at = NOW() WHERE subscription_id = $1`, subID)
	require.NoError(t, err)

	cached, err := svc.GetSubscription(user)
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE, cached.State, "cache should still report active")

	time.Sleep(svc.cacheTTL + 10*time.Millisecond)
	refreshed, err := svc.GetSubscription(user)
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED, refreshed.State)
}

func TestCreditsReflectBundleMetadata(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema, stripe.Schema)

	bundleKey := configureBundleEnv(t, "credits_env")
	productID := billingfix.UpsertBundleProduct(t, db, bundleKey, "Credits Test Bundle", "prod_credits_test", "credits_env", 2_000_000, 0.0025, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_credits_test", "Credits Plan", "solo", "month", "usd",
		3000, false, "none", 0, 0, "credits_lookup", 3_000_000, 500_000, 1, 10, "none", "subscription", nil)

	const email = "credits-test@example.com"
	_, err := db.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, bonus_credits, updated_at)
		VALUES ($1,$2,$3,NOW())
		ON CONFLICT (customer_email) DO UPDATE SET balance_credits = EXCLUDED.balance_credits, bonus_credits = EXCLUDED.bonus_credits, updated_at = NOW()
	`, email, 1_000_000, 200_000)
	require.NoError(t, err)

	credits, err := NewService(db, plan.NewService(db)).GetCredits(email)
	require.NoError(t, err)
	require.Equal(t, "credits", credits.DisplayCreditsLabel)
	require.Equal(t, 0.0025, credits.DisplayCreditsMultiplier)
	require.Equal(t, int64(1_000_000), credits.Balance.BalanceCredits)
}

func TestEntitlementsIncludesFeatures(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema, stripe.Schema)

	bundleKey := configureBundleEnv(t, "entitlements_env")
	productID := billingfix.UpsertBundleProduct(t, db, bundleKey, "Entitlements Test Bundle", "prod_entitlements_test", "entitlements_env", 2_500_000, 0.003, "entitlements")
	billingfix.InsertBundlePrice(t, db, productID, "price_entitlements_plan", "Entitlements Plan", "studio", "month", "usd",
		9999, false, "none", 0, 0, "entitlements_lookup", 5_000_000, 0, 3, 20, "none", "subscription",
		map[string]interface{}{"features": []string{"Download gating", "Credits top-up"}})

	const email = "entitlements@example.com"
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()
	`, "sub_entitlements_123", email, "active", "studio", "price_entitlements_plan", bundleKey)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, bonus_credits, updated_at)
		VALUES ($1,$2,$3,NOW())
		ON CONFLICT (customer_email) DO UPDATE SET balance_credits = EXCLUDED.balance_credits, updated_at = NOW()
	`, email, 2_000_000, 300_000)
	require.NoError(t, err)

	ent, err := NewService(db, plan.NewService(db)).GetEntitlements(email)
	require.NoError(t, err)
	require.Equal(t, "active", ent.Status)
	require.Equal(t, "studio", ent.PlanTier)
	require.NotNil(t, ent.Credits)
	require.NotEmpty(t, ent.Features)
}

func configureBundleEnv(t *testing.T, env string) string {
	t.Helper()
	replacer := strings.NewReplacer("/", "_", ".", "_")
	bundleKey := fmt.Sprintf("bundle_%s", replacer.Replace(strings.ToLower(t.Name())))
	prevKey, prevEnv := os.Getenv("BUNDLE_KEY"), os.Getenv("BUNDLE_ENVIRONMENT")
	require.NoError(t, os.Setenv("BUNDLE_KEY", bundleKey))
	require.NoError(t, os.Setenv("BUNDLE_ENVIRONMENT", env))
	t.Cleanup(func() {
		if prevKey == "" {
			_ = os.Unsetenv("BUNDLE_KEY")
		} else {
			_ = os.Setenv("BUNDLE_KEY", prevKey)
		}
		if prevEnv == "" {
			_ = os.Unsetenv("BUNDLE_ENVIRONMENT")
		} else {
			_ = os.Setenv("BUNDLE_ENVIRONMENT", prevEnv)
		}
	})
	return bundleKey
}
