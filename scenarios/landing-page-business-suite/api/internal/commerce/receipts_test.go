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
	_, err := service.RegisterReceipt(context.Background(), validators, Receipt{Source: "apple", Token: "signed", UserIdentity: "buyer@example.com"})
	if !errors.Is(err, ErrReceiptReplay) {
		t.Fatalf("replayed receipt error = %v, want %v", err, ErrReceiptReplay)
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
	if _, err := validator.Validate(context.Background(), Receipt{Source: "google", Token: "x"}); err != ErrReceiptInvalid {
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
			`CREATE TABLE subscriptions (subscription_id TEXT, customer_email TEXT, customer_id TEXT, status TEXT, source TEXT, external_subscription_id TEXT, plan_tier TEXT, price_id TEXT, bundle_key TEXT, canceled_at TIMESTAMP, updated_at TIMESTAMP, billing_cycle_start INTEGER, created_at TIMESTAMP)`,
			`CREATE TABLE credit_wallets (customer_email TEXT, balance_credits INTEGER, bonus_credits INTEGER, updated_at TIMESTAMP)`,
		} {
			if _, err := db.Exec(statement); err != nil {
				t.Fatal(err)
			}
		}

		service := NewService(db, equivalentReceiptPlanCatalog{}, Runtime{LeaseTTL: time.Hour})
		validators := ReceiptValidators{source: VerifiedReceiptFunc(func(context.Context, Receipt) (NormalizedSubscription, error) {
			return canonical, nil
		})}
		got, err := service.RegisterReceipt(context.Background(), validators, Receipt{Source: source, Token: "verified", UserIdentity: canonical.UserIdentity})
		if err != nil {
			t.Fatalf("%s registration: %v", source, err)
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
