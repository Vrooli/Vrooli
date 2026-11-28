package commerce

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"
)

type creditTopupPlansFake struct {
	bundle *BundleProduct
	err    error
}

func (f creditTopupPlansFake) GetBundleProduct() (*BundleProduct, error) { return f.bundle, f.err }

type creditTopupWalletFake struct {
	account, email, kind, event string
	credits                     int64
	metadata                    map[string]interface{}
	ctx                         context.Context
	grants                      map[string]string
	grantCount                  int
}

func (f *creditTopupWalletFake) AddCreditsForBusinessAccountContext(ctx context.Context, account, email string, credits int64, kind, source, event string, metadata map[string]interface{}) error {
	f.ctx = ctx
	if f.grants == nil {
		f.grants = make(map[string]string)
	}
	key := source + ":" + event
	grant := account + ":" + email + ":" + kind + ":" + strconv.FormatInt(credits, 10)
	if previous, exists := f.grants[key]; exists {
		if previous != grant {
			return errors.New("event replay differs from original grant")
		}
		return nil
	}
	f.grants[key] = grant
	f.grantCount++
	f.account, f.email, f.credits, f.kind, f.event, f.metadata = account, email, credits, kind, event, metadata
	return nil
}
func (*creditTopupWalletFake) BalanceForBusinessAccountContext(context.Context, string, string) (int64, error) {
	return 0, nil
}

func TestCreditTopupServiceApplyRecordsReplaySafeWalletCredit(t *testing.T) {
	wallet := &creditTopupWalletFake{}
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 150}}, wallet, nil)
	metadata := map[string]interface{}{"checkout_session": "cs_123", "business_account_id": "acct:one", "livemode": false}

	ctx := context.WithValue(context.Background(), struct{}{}, "request-context")
	plan := &PlanOption{StripePriceId: "price_123", PlanTier: "credits", AmountCents: 1000}
	err := service.ApplyContext(ctx, "buyer@example.test", 1000, plan, "evt_123", metadata)
	if err != nil {
		t.Fatalf("ApplyContext() error = %v", err)
	}
	if wallet.account != "acct:one" || wallet.email != "buyer@example.test" || wallet.credits != 1500 || wallet.kind != "credit_topup" || wallet.event != "evt_123" {
		t.Fatalf("wallet call = %#v, want normalized top-up credit", wallet)
	}
	if got := wallet.metadata["price_id"]; got != "price_123" {
		t.Fatalf("metadata price_id = %v, want price_123", got)
	}
	if got := wallet.metadata["session_type"]; got != "credits_topup" {
		t.Fatalf("metadata session_type = %v, want credits_topup", got)
	}
	if wallet.ctx != ctx {
		t.Fatal("top-up did not preserve the caller context")
	}
	if wallet.grantCount != 1 {
		t.Fatalf("context grant count = %d; want one account grant", wallet.grantCount)
	}
	if err := service.ApplyContext(ctx, "buyer@example.test", 1000, plan, "evt_123", map[string]interface{}{"checkout_session": "cs_123", "business_account_id": "acct:one", "livemode": false}); err != nil {
		t.Fatalf("identical provider event replay error = %v", err)
	}
	if wallet.grantCount != 1 {
		t.Fatalf("identical provider event replay created %d grants, want 1", wallet.grantCount)
	}
}

func TestCreditTopupServiceApplyRejectsMissingPlanAndCatalog(t *testing.T) {
	service := NewCreditTopupService(nil, &creditTopupWalletFake{}, nil)
	if err := service.ApplyContext(context.Background(), "buyer@example.test", 100, nil, "evt_123", nil); err == nil {
		t.Fatal("Apply() error = nil, want missing plan rejection")
	}

	service = NewCreditTopupService(creditTopupPlansFake{err: errors.New("catalog unavailable")}, &creditTopupWalletFake{}, nil)
	err := service.ApplyContext(context.Background(), "buyer@example.test", 1000, &PlanOption{StripePriceId: "price_123", PlanTier: "credits", AmountCents: 1000}, "evt_123", nil)
	if err == nil || err.Error() != "catalog unavailable" {
		t.Fatalf("Apply() error = %v, want catalog error", err)
	}
}

func TestCreditTopupServiceApplyRequiresCallerContext(t *testing.T) {
	wallet := &creditTopupWalletFake{}
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 150}}, wallet, nil)
	plan := &PlanOption{StripePriceId: "price_123", PlanTier: "credits", AmountCents: 1000}
	metadata := map[string]interface{}{"business_account_id": "acct:one", "livemode": false}
	if err := service.ApplyContext(nil, "buyer@example.test", 1000, plan, "evt_123", metadata); err == nil {
		t.Fatal("ApplyContext(nil) error = nil, want missing context rejection")
	}
	if wallet.grantCount != 0 {
		t.Fatalf("nil context created %d wallet grants, want none", wallet.grantCount)
	}
	if _, exists := metadata["price_id"]; exists {
		t.Fatal("nil context should reject before mutating provider metadata")
	}
}

func TestCreditTopupServiceApplyRejectsUnapprovedPlanAmount(t *testing.T) {
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 1_000_000}}, &creditTopupWalletFake{}, nil)

	for _, plan := range []*PlanOption{
		{StripePriceId: "price_subscription", PlanTier: "pro", AmountCents: 1000},
		{StripePriceId: "price_variable", PlanTier: "credits", AmountCents: 1000, IsVariableAmount: true},
		{StripePriceId: "price_25", PlanTier: "credits", AmountCents: 2500},
	} {
		if err := service.ApplyContext(context.Background(), "buyer@example.test", plan.AmountCents, plan, "evt_invalid", nil); err == nil {
			t.Fatalf("Apply() error = nil for invalid plan %#v", plan)
		}
	}
}

func TestCreditTopupServiceApplyRejectsProviderAmountMismatch(t *testing.T) {
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 1_000_000}}, &creditTopupWalletFake{}, nil)
	plan := &PlanOption{StripePriceId: "price_10", PlanTier: "credits", AmountCents: 1000}

	if err := service.ApplyContext(context.Background(), "buyer@example.test", 500, plan, "evt_mismatch", nil); err == nil {
		t.Fatal("Apply() error = nil, want provider/catalog amount mismatch rejection")
	}
}

func TestCreditTopupServiceRequiresAccountAndModeAttribution(t *testing.T) {
	plan := &PlanOption{StripePriceId: "price_10", PlanTier: "credits", AmountCents: 1000}
	for _, tc := range []struct {
		name     string
		metadata map[string]interface{}
		event    string
	}{
		{"missing account", map[string]interface{}{"livemode": false}, "event"},
		{"missing mode", map[string]interface{}{"business_account_id": "acct"}, "event"},
		{"unknown mode", map[string]interface{}{"business_account_id": "acct", "livemode": "unknown"}, "event"},
		{"missing event", map[string]interface{}{"business_account_id": "acct", "livemode": false}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wallet := &creditTopupWalletFake{}
			service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 100}}, wallet, nil)
			if err := service.ApplyContext(context.Background(), "member@example.test", 1000, plan, tc.event, tc.metadata); err == nil {
				t.Fatal("an unattributed top-up must fail")
			}
			if wallet.credits != 0 {
				t.Fatal("invalid attribution reached the credit grant owner")
			}
		})
	}
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: math.MaxInt64}}, &creditTopupWalletFake{}, nil)
	if err := service.ApplyContext(context.Background(), "member@example.test", 1000, plan, "event", map[string]interface{}{"business_account_id": "acct", "livemode": false}); err == nil {
		t.Fatal("overflowing grant conversion must fail")
	}
}
