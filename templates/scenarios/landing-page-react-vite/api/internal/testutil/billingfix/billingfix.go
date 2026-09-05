// Package billingfix provides shared bundle-product / bundle-price seed helpers
// for the financial-cluster domain tests (plan, stripe, account, payments). It
// mirrors the old flat-API test_bundle_helpers so ported tests seed identical
// fixtures.
package billingfix

import (
	"database/sql"
	"encoding/json"
	"testing"
)

// UpsertBundleProduct ensures a bundle product exists for the given key and
// environment, returning its primary id.
func UpsertBundleProduct(t *testing.T, db *sql.DB, bundleKey, bundleName, stripeProductID, environment string, creditsPerUSD int64, displayMultiplier float64, displayLabel string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(`
		INSERT INTO bundle_products (
			bundle_key, bundle_name, stripe_product_id,
			credits_per_usd, display_credits_multiplier, display_credits_label, environment
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (bundle_key) DO UPDATE SET
			bundle_name = EXCLUDED.bundle_name,
			stripe_product_id = EXCLUDED.stripe_product_id,
			credits_per_usd = EXCLUDED.credits_per_usd,
			display_credits_multiplier = EXCLUDED.display_credits_multiplier,
			display_credits_label = EXCLUDED.display_credits_label,
			environment = EXCLUDED.environment,
			updated_at = NOW()
		RETURNING id
	`, bundleKey, bundleName, stripeProductID, creditsPerUSD, displayMultiplier, displayLabel, environment).Scan(&id)
	if err != nil {
		t.Fatalf("failed to upsert bundle product: %v", err)
	}
	return id
}

// InsertBundlePrice stores a pricing tier connected to the given bundle product.
func InsertBundlePrice(
	t *testing.T, db *sql.DB, productID int64,
	priceID, planName, planTier, billingInterval, currency string,
	amountCents int, introEnabled bool, introType string,
	introAmountCents, introPeriods int, introLookupKey string,
	monthlyIncluded, oneTimeBonus, planRank, displayWeight int,
	bonusType, kind string, metadata map[string]interface{},
) {
	t.Helper()
	metaBytes := []byte(`{}`)
	if len(metadata) > 0 {
		var err error
		metaBytes, err = json.Marshal(metadata)
		if err != nil {
			t.Fatalf("failed to marshal metadata: %v", err)
		}
	}

	introValue := sql.NullInt64{}
	if introEnabled && introAmountCents > 0 {
		introValue.Int64 = int64(introAmountCents)
		introValue.Valid = true
	}

	_, err := db.Exec(`
		INSERT INTO bundle_prices (
			product_id, stripe_price_id, plan_name, plan_tier, billing_interval,
			amount_cents, currency, intro_enabled, intro_type, intro_amount_cents, intro_periods, intro_price_lookup_key,
			monthly_included_credits, one_time_bonus_credits, plan_rank, bonus_type,
			kind, metadata, display_weight
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19
		)
	`, productID, priceID, planName, planTier, billingInterval,
		amountCents, currency, introEnabled, introType, introValue, introPeriods, introLookupKey,
		monthlyIncluded, oneTimeBonus, planRank, bonusType,
		kind, string(metaBytes), displayWeight)
	if err != nil {
		t.Fatalf("failed to insert bundle price %s: %v", priceID, err)
	}
}
