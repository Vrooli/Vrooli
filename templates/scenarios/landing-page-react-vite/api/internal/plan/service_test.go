package plan_test

import (
	"fmt"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/testutil/billingfix"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func TestPricingOverview(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema)

	bundleKey := configureBundleEnv(t, "pricing_env")
	productID := billingfix.UpsertBundleProduct(t, db, bundleKey, "Pricing Test Bundle", "prod_pricing_test", "pricing_env", 1_000_000, 0.01, "credits")

	billingfix.InsertBundlePrice(t, db, productID, "price_pricing_monthly", "Pricing Monthly", "pro", "month", "usd",
		4999, true, "flat_amount", 100, 1, "monthly_intro_key", 5_000_000, 0, 1, 30, "none", "subscription",
		map[string]interface{}{"features": []string{"Fast coupling", "Priority support"}})
	billingfix.InsertBundlePrice(t, db, productID, "price_pricing_yearly", "Pricing Yearly", "pro", "year", "usd",
		55999, false, "none", 0, 0, "yearly_lookup_key", 60_000_000, 10_000_000, 2, 10, "yearly_bonus", "subscription",
		map[string]interface{}{"features": []string{"Annual loyalty", "Bonus credits"}})

	overview, err := plan.NewService(db).GetPricingOverview()
	require.NoError(t, err)
	require.Equal(t, bundleKey, overview.Bundle.BundleKey)
	require.Len(t, overview.Monthly, 1)
	require.Len(t, overview.Yearly, 1)
	require.Equal(t, "price_pricing_monthly", overview.Monthly[0].StripePriceId)
	require.True(t, overview.Monthly[0].IntroEnabled)
	require.Equal(t, landingv1.BillingInterval_BILLING_INTERVAL_MONTH, overview.Monthly[0].BillingInterval)
	require.Equal(t, landingv1.BillingInterval_BILLING_INTERVAL_YEAR, overview.Yearly[0].BillingInterval)
	require.False(t, overview.Yearly[0].IntroEnabled)
}

func TestGetPlanByPriceID(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema)

	bundleKey := configureBundleEnv(t, "pricing_env")
	productID := billingfix.UpsertBundleProduct(t, db, bundleKey, "Pricing Test Bundle", "prod_lookup_test", "pricing_env", 1_000_000, 0.01, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_lookup_test", "Lookup Plan", "pro", "month", "usd",
		9999, true, "flat_amount", 100, 1, "lookup_key", 10_000_000, 0, 5, 40, "none", "subscription",
		map[string]interface{}{"features": []string{"Lookup feature"}})

	option, err := plan.NewService(db).GetPlanByPriceID("price_lookup_test")
	require.NoError(t, err)
	require.Equal(t, "Lookup Plan", option.PlanName)
	require.NotNil(t, option.Metadata)
	require.Contains(t, option.Metadata, "features")
}

// configureBundleEnv points the plan service at a per-test bundle key/environment
// so the shared test database stays free of cross-test collisions.
func configureBundleEnv(t *testing.T, env string) string {
	t.Helper()
	replacer := strings.NewReplacer("/", "_", ".", "_")
	bundleKey := fmt.Sprintf("bundle_%s", replacer.Replace(strings.ToLower(t.Name())))
	prevKey, prevEnv := os.Getenv("BUNDLE_KEY"), os.Getenv("BUNDLE_ENVIRONMENT")
	require.NoError(t, os.Setenv("BUNDLE_KEY", bundleKey))
	require.NoError(t, os.Setenv("BUNDLE_ENVIRONMENT", env))
	t.Cleanup(func() {
		setEnvOrClear("BUNDLE_KEY", prevKey)
		setEnvOrClear("BUNDLE_ENVIRONMENT", prevEnv)
	})
	return bundleKey
}

func setEnvOrClear(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, value)
}
