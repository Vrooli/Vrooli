package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type receiptResult int64

func (r receiptResult) LastInsertId() (int64, error) { return 0, nil }
func (r receiptResult) RowsAffected() (int64, error) { return int64(r), nil }

type replayReceiptStore struct{}

func (replayReceiptStore) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("query unsupported")
}
func (replayReceiptStore) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }
func (replayReceiptStore) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return receiptResult(0), nil
}

func TestRegisterReceiptRejectsReplayBeforeRefreshingEntitlements(t *testing.T) {
	service := NewService(replayReceiptStore{}, testPlanCatalog{}, Runtime{})
	validators := ReceiptValidators{
		"apple": AppleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) {
			return NormalizedSubscription{
				SubscriptionID: "sub-1", ExternalSubscription: "tx-1", UserIdentity: "buyer@example.com",
				Status: "active", PlanTier: "pro", PriceID: "pro-monthly", BundleKey: "business_suite",
			}, nil
		}},
	}
	_, err := service.RegisterReceipt(context.Background(), validators, Receipt{Source: "apple", Token: "signed", UserIdentity: "buyer@example.com", BusinessAccountID: "account-1"})
	if !errors.Is(err, ErrReceiptReplay) {
		t.Fatalf("replayed receipt error = %v, want %v", err, ErrReceiptReplay)
	}
}

func TestRegisterReceiptRequiresResolvedBusinessAccount(t *testing.T) {
	validated := false
	service := NewService(replayReceiptStore{}, testPlanCatalog{}, Runtime{})
	validators := ReceiptValidators{"apple": AppleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) {
		validated = true
		return NormalizedSubscription{}, nil
	}}}
	_, err := service.RegisterReceipt(context.Background(), validators, Receipt{Source: "apple", Token: "signed", UserIdentity: "buyer@example.com"})
	if !errors.Is(err, ErrReceiptInvalid) || validated {
		t.Fatalf("registration error=%v validated=%t, want invalid receipt before validation", err, validated)
	}
}

func TestReceiptValidatorsNormalizeEveryPurchaseRailToOneShape(t *testing.T) {
	canonical := NormalizedSubscription{
		SubscriptionID: "sub-canonical", ExternalSubscription: "external-canonical",
		UserIdentity: "buyer@example.com", Status: "active", PlanTier: "pro",
		PriceID: "price-pro", BundleKey: "business_suite",
	}
	validators := ReceiptValidators{
		"stripe": StripeReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) { return canonical, nil }},
		"apple":  AppleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) { return canonical, nil }},
		"google": GoogleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) { return canonical, nil }},
	}
	for _, source := range []string{"stripe", "apple", "google"} {
		got, err := validators.Validate(context.Background(), Receipt{Source: source, Token: "verified", UserIdentity: canonical.UserIdentity})
		if err != nil {
			t.Fatalf("%s: %v", source, err)
		}
		if !reflect.DeepEqual(got, canonical) {
			t.Fatalf("%s normalized = %+v, want %+v", source, got, canonical)
		}
	}
}

func TestReceiptValidatorsRejectWrongSourceAndUnknownRail(t *testing.T) {
	validator := AppleReceiptValidator{Verify: func(context.Context, Receipt) (NormalizedSubscription, error) {
		return NormalizedSubscription{}, nil
	}}
	if _, err := validator.Validate(context.Background(), Receipt{Source: "google", Token: "x"}); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("wrong source error = %v", err)
	}
	if _, err := (ReceiptValidators{"apple": validator}).Validate(context.Background(), Receipt{Source: "play", Token: "x"}); err == nil {
		t.Fatal("unknown source unexpectedly accepted")
	}
}

type equivalentReceiptPlanCatalog struct{}

func (equivalentReceiptPlanCatalog) BundleKey() string { return "business_suite" }

func (equivalentReceiptPlanCatalog) GetPricingOverview() (*shared.PricingOverview, error) {
	return &shared.PricingOverview{Bundle: &shared.Bundle{BundleKey: "business_suite"}}, nil
}

func (equivalentReceiptPlanCatalog) GetPlanByPriceID(string) (*shared.PlanOption, error) {
	return nil, fmt.Errorf("plan lookup intentionally omitted")
}

func TestReceiptSourcesProduceEquivalentEntitlementPayloads(t *testing.T) {
	canonical := NormalizedSubscription{
		SubscriptionID: "sub-canonical", ExternalSubscription: "external-canonical",
		UserIdentity: "buyer@example.com", Status: "active", PlanTier: "pro",
		PriceID: "price-pro", BundleKey: "business_suite",
	}
	var want *EntitlementPayload
	for _, source := range []string{"stripe", "apple", "google"} {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		for _, statement := range []string{
			`CREATE TABLE subscriptions (subscription_id TEXT, customer_email TEXT, customer_id TEXT, business_account_id TEXT, status TEXT, source TEXT, external_subscription_id TEXT, plan_tier TEXT, price_id TEXT, bundle_key TEXT, canceled_at TIMESTAMP, updated_at TIMESTAMP, billing_cycle_start INTEGER, created_at TIMESTAMP)`,
			`CREATE TABLE business_account_credit_wallets (business_account_id TEXT PRIMARY KEY, customer_email TEXT NOT NULL, balance_credits INTEGER NOT NULL DEFAULT 0, bonus_credits INTEGER NOT NULL DEFAULT 0, updated_at DATETIME)`,
			`CREATE TABLE credit_funding_lots (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, source_kind TEXT, provider TEXT, provider_ref TEXT, livemode BOOLEAN, settlement_state TEXT NOT NULL, spendable_after DATETIME, reversal_window_ends DATETIME, granted INTEGER, remaining INTEGER NOT NULL, expires_at DATETIME, status TEXT NOT NULL, created_at DATETIME)`,
		} {
			if _, err := db.Exec(statement); err != nil {
				t.Fatal(err)
			}
		}

		service := NewService(db, equivalentReceiptPlanCatalog{}, Runtime{LeaseTTL: time.Hour})
		validators := ReceiptValidators{source: VerifiedReceiptFunc(func(context.Context, Receipt) (NormalizedSubscription, error) {
			return canonical, nil
		})}
		got, err := service.RegisterReceipt(context.Background(), validators, Receipt{Source: source, Token: "verified", UserIdentity: canonical.UserIdentity, BusinessAccountID: "account-1"})
		if err != nil {
			t.Fatalf("%s registration: %v", source, err)
		}
		if got.Status != "active" || got.PlanTier != "pro" {
			t.Fatalf("%s receipt returned account entitlements status=%q plan=%q, want active pro", source, got.Status, got.PlanTier)
		}
		var storedAccount string
		if err := db.QueryRow(`SELECT business_account_id FROM subscriptions WHERE subscription_id = ?`, canonical.SubscriptionID).Scan(&storedAccount); err != nil {
			t.Fatalf("%s read persisted receipt account: %v", source, err)
		}
		if storedAccount != "account-1" {
			t.Fatalf("%s receipt account = %q, want selected account", source, storedAccount)
		}
		got.NotAfter = time.Time{}
		got.Subscription = nil
		if got.Credits != nil {
			got.Credits.UpdatedAt = nil
		}
		if want == nil {
			want = got
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s entitlement payload = %#v, want %#v", source, got, want)
		}
	}
}
