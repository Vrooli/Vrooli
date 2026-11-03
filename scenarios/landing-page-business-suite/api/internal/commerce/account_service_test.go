package commerce

import (
	"context"
	"database/sql"
	"testing"

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

func TestServiceNormalizesMissingIdentityWithoutPersistence(t *testing.T) {
	service := NewService(nil, testPlanCatalog{}, Runtime{})
	status, err := service.GetSubscription("  ")
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
