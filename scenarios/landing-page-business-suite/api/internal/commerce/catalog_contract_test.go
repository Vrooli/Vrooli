package commerce

import (
	"errors"
	"testing"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func TestNormalizeStripeImportSelections(t *testing.T) {
	selections, validationErrors, err := NormalizeStripeImportSelections([]ImportPlanSelection{
		{PriceID: " price_alpha ", Action: " IMPORT "},
		{PriceID: "price_alpha", Action: "overwrite"},
		{PriceID: "", Action: "skip"},
		{PriceID: "price_beta", Action: "invalid"},
	})
	if err != nil {
		t.Fatalf("NormalizeStripeImportSelections() error = %v", err)
	}
	if len(selections) != 1 || selections[0].PriceID != "price_alpha" || selections[0].Action != "import" {
		t.Fatalf("normalized selections = %#v", selections)
	}
	if len(validationErrors) != 3 {
		t.Fatalf("validation errors = %#v", validationErrors)
	}
}

func TestNormalizeStripeImportSelectionsRejectsNoUsableSelections(t *testing.T) {
	_, validationErrors, err := NormalizeStripeImportSelections([]ImportPlanSelection{{PriceID: "", Action: "import"}})
	if !errors.Is(err, ErrStripeImportNoValidSelections) {
		t.Fatalf("error = %v, want ErrStripeImportNoValidSelections", err)
	}
	if len(validationErrors) != 1 {
		t.Fatalf("validation errors = %#v", validationErrors)
	}
}

func TestBuildPricingOverviewExposesCreditTopupsSeparately(t *testing.T) {
	overview, err := BuildPricingOverview(&shared.Bundle{BundleKey: "business_suite", Name: "Suite"}, []*shared.PlanOption{
		{PlanName: "Pro", PlanTier: "pro", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_MONTH, DisplayEnabled: true},
		{PlanName: "$10 credits", PlanTier: "credits", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_ONE_TIME, AmountCents: 1000, Currency: "usd", StripePriceId: "price_10", Kind: shared.PlanKind_PLAN_KIND_CREDITS_TOPUP, DisplayEnabled: true},
		{PlanName: "Variable credits", PlanTier: "credits", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_ONE_TIME, AmountCents: 0, Currency: "usd", StripePriceId: "price_variable", Kind: shared.PlanKind_PLAN_KIND_CREDITS_TOPUP, IsVariableAmount: true, DisplayEnabled: true},
		{PlanName: "Support", PlanTier: "donation", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_ONE_TIME, AmountCents: 1000, Currency: "usd", Kind: shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION, DisplayEnabled: true},
	})
	if err != nil {
		t.Fatalf("BuildPricingOverview() error = %v", err)
	}
	if len(overview.CreditTopups) != 1 || overview.CreditTopups[0].StripePriceId != "price_10" {
		t.Fatalf("credit_topups = %#v, want only the credit top-up", overview.CreditTopups)
	}
}

func TestNormalizePlanOptionEnforcesFixedCreditTopupPolicy(t *testing.T) {
	for _, plan := range []*shared.PlanOption{
		{StripePriceId: "price_variable", PlanName: "Variable credits", PlanTier: "credits", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_ONE_TIME, AmountCents: 1000, Currency: "usd", Kind: shared.PlanKind_PLAN_KIND_CREDITS_TOPUP, IsVariableAmount: true},
		{StripePriceId: "price_25", PlanName: "$25 credits", PlanTier: "credits", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_ONE_TIME, AmountCents: 2500, Currency: "usd", Kind: shared.PlanKind_PLAN_KIND_CREDITS_TOPUP},
	} {
		if err := NormalizePlanOption(plan, "business_suite"); err == nil {
			t.Fatalf("NormalizePlanOption accepted invalid credit top-up %q", plan.PlanName)
		}
	}
}
