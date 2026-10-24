package commerce

import (
	"context"
	"errors"
	"testing"
)

type creditTopupPlansFake struct {
	bundle *BundleProduct
	err    error
}

func (f creditTopupPlansFake) GetBundleProduct() (*BundleProduct, error) { return f.bundle, f.err }

type creditTopupWalletFake struct {
	email, kind, event string
	credits            int64
	metadata           map[string]interface{}
}

func (f *creditTopupWalletFake) AddCredits(email string, credits int64, kind, event string, metadata map[string]interface{}) error {
	f.email, f.credits, f.kind, f.event, f.metadata = email, credits, kind, event, metadata
	return nil
}

func (*creditTopupWalletFake) ConsumeCredits(context.Context, string, int64, string, map[string]interface{}) error {
	return nil
}

func (*creditTopupWalletFake) ConsumeCreditsIdempotent(context.Context, string, int64, string, string, map[string]interface{}) error {
	return nil
}
func (*creditTopupWalletFake) Balance(string) (int64, error) { return 0, nil }

func TestCreditTopupServiceApplyRecordsReplaySafeWalletCredit(t *testing.T) {
	wallet := &creditTopupWalletFake{}
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 150}}, wallet, nil)
	metadata := map[string]interface{}{"checkout_session": "cs_123"}

	err := service.Apply("buyer@example.test", 1000, &PlanOption{StripePriceId: "price_123", PlanTier: "credits", AmountCents: 1000}, "evt_123", metadata)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if wallet.email != "buyer@example.test" || wallet.credits != 1500 || wallet.kind != "credit_topup" || wallet.event != "evt_123" {
		t.Fatalf("wallet call = %#v, want normalized top-up credit", wallet)
	}
	if got := wallet.metadata["price_id"]; got != "price_123" {
		t.Fatalf("metadata price_id = %v, want price_123", got)
	}
	if got := wallet.metadata["session_type"]; got != "credits_topup" {
		t.Fatalf("metadata session_type = %v, want credits_topup", got)
	}
}

func TestCreditTopupServiceApplyRejectsMissingPlanAndCatalog(t *testing.T) {
	service := NewCreditTopupService(nil, &creditTopupWalletFake{}, nil)
	if err := service.Apply("buyer@example.test", 100, nil, "evt_123", nil); err == nil {
		t.Fatal("Apply() error = nil, want missing plan rejection")
	}

	service = NewCreditTopupService(creditTopupPlansFake{err: errors.New("catalog unavailable")}, &creditTopupWalletFake{}, nil)
	err := service.Apply("buyer@example.test", 1000, &PlanOption{StripePriceId: "price_123", PlanTier: "credits", AmountCents: 1000}, "evt_123", nil)
	if err == nil || err.Error() != "catalog unavailable" {
		t.Fatalf("Apply() error = %v, want catalog error", err)
	}
}

func TestCreditTopupServiceApplyRejectsUnapprovedPlanAmount(t *testing.T) {
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 1_000_000}}, &creditTopupWalletFake{}, nil)

	for _, plan := range []*PlanOption{
		{StripePriceId: "price_subscription", PlanTier: "pro", AmountCents: 1000},
		{StripePriceId: "price_variable", PlanTier: "credits", AmountCents: 1000, IsVariableAmount: true},
		{StripePriceId: "price_25", PlanTier: "credits", AmountCents: 2500},
	} {
		if err := service.Apply("buyer@example.test", plan.AmountCents, plan, "evt_invalid", nil); err == nil {
			t.Fatalf("Apply() error = nil for invalid plan %#v", plan)
		}
	}
}

func TestCreditTopupServiceApplyRejectsProviderAmountMismatch(t *testing.T) {
	service := NewCreditTopupService(creditTopupPlansFake{bundle: &BundleProduct{CreditsPerUsd: 1_000_000}}, &creditTopupWalletFake{}, nil)
	plan := &PlanOption{StripePriceId: "price_10", PlanTier: "credits", AmountCents: 1000}

	if err := service.Apply("buyer@example.test", 500, plan, "evt_mismatch", nil); err == nil {
		t.Fatal("Apply() error = nil, want provider/catalog amount mismatch rejection")
	}
}
