package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	_ "modernc.org/sqlite"
)

type testPlanCatalog struct{}

func (testPlanCatalog) BundleKey() string { return "business_suite" }

func (testPlanCatalog) GetPricingOverview() (*shared.PricingOverview, error) { return nil, nil }

func (testPlanCatalog) GetPlanByPriceID(string) (*shared.PlanOption, error) { return nil, nil }

type testEntitlementPlanCatalog struct{ testPlanCatalog }

func (testEntitlementPlanCatalog) GetPricingOverview() (*shared.PricingOverview, error) {
	return &shared.PricingOverview{Bundle: &shared.Bundle{
		BundleKey:                "business_suite",
		DisplayCreditsLabel:      "credits",
		DisplayCreditsMultiplier: 1,
	}}, nil
}

type appScopedPlanCatalog struct {
	testEntitlementPlanCatalog
	plans map[string]*shared.PlanOption
}

func (c appScopedPlanCatalog) GetPlanByPriceID(priceID string) (*shared.PlanOption, error) {
	plan := c.plans[priceID]
	if plan == nil {
		return nil, fmt.Errorf("plan %s not found", priceID)
	}
	return plan, nil
}

func TestServiceNormalizesMissingIdentityWithoutPersistence(t *testing.T) {
	service := NewService(nil, testPlanCatalog{}, Runtime{})
	status, err := service.GetSubscriptionContext(context.Background(), "  ")
	if err != nil {
		t.Fatal(err)
	}
	if status.GetState() != shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE || status.GetMessage() != "user not provided" {
		t.Fatalf("unexpected missing-identity status: %#v", status)
	}
}

func TestMapSubscriptionState(t *testing.T) {
	if got := MapSubscriptionState("past-due"); got != shared.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE {
		t.Fatalf("MapSubscriptionState(past-due) = %v", got)
	}
	if got := MapSubscriptionState("unknown"); got != shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE {
		t.Fatalf("MapSubscriptionState(unknown) = %v", got)
	}
}

func TestBusinessAccountEntitlementsDisplayOnlyAttributedFundingLots(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	for _, statement := range []string{
		`CREATE TABLE business_account_credit_wallets (business_account_id TEXT PRIMARY KEY, customer_email TEXT NOT NULL, balance_credits INTEGER NOT NULL, bonus_credits INTEGER NOT NULL DEFAULT 0, updated_at DATETIME)`,
		`CREATE TABLE credit_funding_lots (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, remaining INTEGER NOT NULL, status TEXT NOT NULL, settlement_state TEXT NOT NULL, livemode BOOLEAN, spendable_after DATETIME, expires_at DATETIME, created_at DATETIME NOT NULL)`,
		`CREATE TABLE subscriptions (subscription_id TEXT, status TEXT, source TEXT, external_subscription_id TEXT, plan_tier TEXT, price_id TEXT, bundle_key TEXT, canceled_at DATETIME, updated_at DATETIME, customer_email TEXT, customer_id TEXT, business_account_id TEXT, billing_cycle_start INTEGER)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create fixture table: %v", err)
		}
	}

	const accountID = "acct-d18"
	const email = "buyer@example.test"
	if _, err := db.Exec(`INSERT INTO business_account_credit_wallets (business_account_id, customer_email, balance_credits, bonus_credits, updated_at) VALUES (?, ?, 42, 9, CURRENT_TIMESTAMP)`, accountID, email); err != nil {
		t.Fatalf("insert historical wallet cache: %v", err)
	}

	service := NewService(db, testEntitlementPlanCatalog{}, Runtime{})
	got, err := service.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
	if err != nil {
		t.Fatalf("get historical entitlements: %v", err)
	}
	if got.Credits.GetBalanceCredits() != 0 {
		t.Fatalf("cache-only historical balance leaked into entitlements: got %d credits, want 0", got.Credits.GetBalanceCredits())
	}

	if _, err := db.Exec(`INSERT INTO credit_funding_lots (id, business_account_id, remaining, status, settlement_state, livemode, created_at) VALUES ('lot-new', ?, 7, 'open', 'settled', 1, CURRENT_TIMESTAMP)`, accountID); err != nil {
		t.Fatalf("insert attributed live lot: %v", err)
	}
	got, err = service.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
	if err != nil {
		t.Fatalf("get attributed entitlements: %v", err)
	}
	if got.Credits.GetBalanceCredits() != 7 {
		t.Fatalf("attributed funding balance = %d, want 7 exactly once", got.Credits.GetBalanceCredits())
	}
}

func TestBusinessAccountEntitlementsUnionScopedActivePlansForSignedLease(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE business_account_credit_wallets (business_account_id TEXT PRIMARY KEY, customer_email TEXT NOT NULL, balance_credits INTEGER NOT NULL, bonus_credits INTEGER NOT NULL DEFAULT 0, updated_at DATETIME)`,
		`CREATE TABLE credit_funding_lots (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, remaining INTEGER NOT NULL, status TEXT NOT NULL, settlement_state TEXT NOT NULL, livemode BOOLEAN, spendable_after DATETIME, expires_at DATETIME, created_at DATETIME NOT NULL)`,
		`CREATE TABLE subscriptions (subscription_id TEXT, status TEXT, source TEXT, external_subscription_id TEXT, plan_tier TEXT, price_id TEXT, bundle_key TEXT, canceled_at DATETIME, updated_at DATETIME, customer_email TEXT, customer_id TEXT, business_account_id TEXT, billing_cycle_start INTEGER)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create fixture table: %v", err)
		}
	}
	const accountID, email = "acct-scoped", "buyer@example.test"
	if _, err := db.Exec(`INSERT INTO subscriptions (subscription_id,status,source,plan_tier,price_id,bundle_key,updated_at,customer_email,business_account_id) VALUES
		('sub-audio','active','stripe','pro','price_audio','business_suite','2026-11-01',?,?),
		('sub-suite','active','stripe','studio','price_suite','business_suite','2026-11-02',?,?),
		('sub-canceled','canceled','stripe','business','price_legacy','business_suite','2026-11-03',?,?)`, email, accountID, email, accountID, email, accountID); err != nil {
		t.Fatalf("insert subscription fixture: %v", err)
	}
	catalog := appScopedPlanCatalog{plans: map[string]*shared.PlanOption{
		"price_audio": {
			StripePriceId: "price_audio", PlanTier: "pro", PlanRank: 2,
			Metadata: appEntitlementsMetadata(map[string]any{"audio-tools": map[string]any{
				"features": []any{"audio:export"}, "limits": map[string]any{"ai_credits": int64(100)},
			}}),
		},
		"price_suite": {
			StripePriceId: "price_suite", PlanTier: "studio", PlanRank: 3,
			Metadata: appEntitlementsMetadata(map[string]any{
				"audio-tools": map[string]any{"features": []any{"audio:advanced"}, "limits": map[string]any{"ai_credits": int64(500)}},
				"web-console": map[string]any{"features": []any{"web:admin"}, "limits": map[string]any{"ai_credits": int64(250)}},
			}),
		},
		"price_audio_plus": {
			StripePriceId: "price_audio_plus", PlanTier: "business", PlanRank: 4,
			Metadata: appEntitlementsMetadata(map[string]any{"audio-tools": map[string]any{
				"features": []any{"audio:export", "audio:master"}, "limits": map[string]any{"ai_credits": int64(400)},
			}}),
		},
	}}
	service := NewService(db, catalog, Runtime{})
	got, err := service.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
	if err != nil {
		t.Fatalf("get effective entitlements: %v", err)
	}
	if len(got.AppEntitlements) != 2 {
		t.Fatalf("app grants = %#v; want audio-tools and web-console", got.AppEntitlements)
	}
	if got.Status != "active" || got.PlanRank != 3 || got.PlanTier != "studio" || got.PriceID != "price_suite" {
		t.Fatalf("effective account summary = status %q rank %d tier %q price %q", got.Status, got.PlanRank, got.PlanTier, got.PriceID)
	}
	if got.AppEntitlements[0].AppKey != "audio-tools" || got.AppEntitlements[0].Status != "active" || got.AppEntitlements[0].PlanRank != 3 {
		t.Fatalf("audio effective grant = %#v", got.AppEntitlements[0])
	}
	if !containsString(got.AppEntitlements[0].Features, "audio:export") || !containsString(got.AppEntitlements[0].Features, "audio:advanced") {
		t.Fatalf("overlapping audio grants were not unioned: %#v", got.AppEntitlements[0].Features)
	}
	if len(got.AppEntitlements[0].Limits) != 1 || got.AppEntitlements[0].Limits[0].Value != 500 {
		t.Fatalf("audio overlapping limit should retain maximum allowance: %#v", got.AppEntitlements[0].Limits)
	}
	if got.AppEntitlements[1].AppKey != "web-console" || !containsString(got.AppEntitlements[1].Features, "web:admin") || len(got.AppEntitlements[1].Limits) != 1 || got.AppEntitlements[1].Limits[0].Value != 250 {
		t.Fatalf("bundle-expanded web grant = %#v", got.AppEntitlements[1])
	}

	if _, err := db.Exec(`UPDATE subscriptions SET price_id='price_audio_plus' WHERE subscription_id='sub-audio'`); err != nil {
		t.Fatal(err)
	}
	service.cacheMutex.Lock()
	clear(service.cache)
	service.cacheMutex.Unlock()
	got, err = service.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
	if err != nil || got.AppEntitlements[0].PlanRank != 4 || !containsString(got.AppEntitlements[0].Features, "audio:master") {
		t.Fatalf("upgraded app subscription grant = %#v, err=%v", got, err)
	}
	if _, err := db.Exec(`UPDATE subscriptions SET price_id='price_audio' WHERE subscription_id='sub-audio'`); err != nil {
		t.Fatal(err)
	}
	service.cacheMutex.Lock()
	clear(service.cache)
	service.cacheMutex.Unlock()
	got, err = service.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
	if err != nil || got.AppEntitlements[0].PlanRank != 3 || containsString(got.AppEntitlements[0].Features, "audio:master") {
		t.Fatalf("downgraded app subscription grant = %#v, err=%v", got, err)
	}
	if _, err := db.Exec(`UPDATE subscriptions SET status='canceled' WHERE subscription_id='sub-suite'`); err != nil {
		t.Fatal(err)
	}
	service.cacheMutex.Lock()
	clear(service.cache)
	service.cacheMutex.Unlock()
	got, err = service.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
	if err != nil || len(got.AppEntitlements) != 1 || got.AppEntitlements[0].AppKey != "audio-tools" || got.AppEntitlements[0].PlanRank != 2 {
		t.Fatalf("canceled bundle grant remained: %#v, err=%v", got.AppEntitlements, err)
	}
	if _, err := db.Exec(`UPDATE subscriptions SET status='active' WHERE subscription_id='sub-suite'`); err != nil {
		t.Fatal(err)
	}
	service.cacheMutex.Lock()
	clear(service.cache)
	service.cacheMutex.Unlock()
	got, err = service.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
	if err != nil || len(got.AppEntitlements) != 2 || got.AppEntitlements[1].AppKey != "web-console" {
		t.Fatalf("recovered bundle grant was not restored: %#v, err=%v", got.AppEntitlements, err)
	}
}

func TestBusinessAccountEntitlementsRejectMixedScopedAndUnscopedPlans(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE subscriptions (subscription_id TEXT, status TEXT, source TEXT, external_subscription_id TEXT, plan_tier TEXT, price_id TEXT, bundle_key TEXT, canceled_at DATETIME, updated_at DATETIME, customer_email TEXT, customer_id TEXT, business_account_id TEXT, billing_cycle_start INTEGER)`); err != nil {
		t.Fatal(err)
	}
	const email, accountID = "buyer@example.test", "acct-mixed"
	for _, price := range []string{"price_scoped", "price_legacy"} {
		if _, err := db.Exec(`INSERT INTO subscriptions (subscription_id,status,source,plan_tier,price_id,bundle_key,updated_at,customer_email,business_account_id) VALUES (?,?,?,?,?,?,?,?,?)`, "sub-"+price, "active", "stripe", "pro", price, "business_suite", "2026-11-01", email, accountID); err != nil {
			t.Fatal(err)
		}
	}
	catalog := appScopedPlanCatalog{plans: map[string]*shared.PlanOption{
		"price_scoped": {StripePriceId: "price_scoped", PlanTier: "pro", Metadata: appEntitlementsMetadata(map[string]any{"audio-tools": map[string]any{"features": []any{"audio:export"}}})},
		"price_legacy": {StripePriceId: "price_legacy", PlanTier: "pro", Metadata: map[string]*commonv1.JsonValue{"features": toJsonValue([]any{"legacy:feature"})}},
	}}
	service := NewService(db, catalog, Runtime{})
	if _, _, err := service.deriveAppEntitlements(context.Background(), email, accountID); err == nil {
		t.Fatal("mixed scoped and unscoped active plans produced a partial grant")
	}
}

func appEntitlementsMetadata(values map[string]any) map[string]*commonv1.JsonValue {
	return map[string]*commonv1.JsonValue{"app_entitlements": toJsonValue(values)}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
