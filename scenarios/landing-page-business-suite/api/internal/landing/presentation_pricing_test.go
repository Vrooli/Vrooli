package landing

import (
	"strings"
	"testing"

	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/commerce"
)

func TestPresentationPricingCarriesCommerceFactsWithoutPrivateMetadata(t *testing.T) { // [REQ:LP-PRES-009]
	intro := int64(499)
	private := map[string]*common.JsonValue{"features": {Kind: &common.JsonValue_StringValue{StringValue: "BAS private workflow narrative /private/key"}}}
	plan := &commerce.PlanOption{
		PlanName: "Solo", PlanTier: "solo", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents: 1299, Currency: "eur", IntroEnabled: true, IntroType: shared.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT,
		IntroAmountCents: &intro, IntroPeriods: 2, IntroPriceLookupKey: "intro-solo", StripePriceId: "price-solo",
		MonthlyIncludedCredits: 120, OneTimeBonusCredits: 25, PlanRank: 1, BonusType: "credits",
		Kind: shared.PlanKind_PLAN_KIND_SUBSCRIPTION, IsVariableAmount: false, DisplayEnabled: true,
		BundleKey: "configured-suite", DisplayWeight: 50, Metadata: private,
	}
	source := &commerce.PricingOverview{
		Bundle:  &commerce.BundleProduct{BundleKey: "configured-suite", Name: "Configured Suite", StripeProductId: "product-suite", CreditsPerUsd: 1000, DisplayCreditsMultiplier: .01, DisplayCreditsLabel: "credits", Environment: "test", Metadata: private},
		Monthly: []*commerce.PlanOption{nil, {PlanName: "Hidden contract", StripePriceId: "price-private"}, plan},
		Yearly:  []*commerce.PlanOption{plan}, CreditTopups: []*commerce.PlanOption{plan},
		UpdatedAt: &timestamppb.Timestamp{Seconds: 100, Nanos: 12},
	}
	result := sanitizePresentationPricing(source)
	want := proto.Clone(source).(*commerce.PricingOverview)
	want.Bundle.Metadata = nil
	want.Monthly = want.Monthly[2:]
	for _, collection := range [][]*commerce.PlanOption{want.Monthly, want.Yearly, want.CreditTopups} {
		for _, value := range collection {
			value.Metadata = nil
		}
	}
	if !proto.Equal(result, want) {
		t.Fatalf("public pricing altered commerce facts: got %v want %v", result, want)
	}
	encoded, err := protojson.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"BAS", "/private", "price-private", "Hidden contract", "metadata"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("public pricing leaks %q", forbidden)
		}
	}
	result.Bundle.Name = "changed"
	result.Monthly[0].PlanName = "changed"
	*result.Monthly[0].IntroAmountCents = 1
	result.UpdatedAt.Seconds = 1
	if source.Bundle.Name != "Configured Suite" || plan.PlanName != "Solo" || intro != 499 || source.UpdatedAt.Seconds != 100 || len(plan.Metadata) != 1 {
		t.Fatal("public pricing aliases or mutates commerce owner data")
	}
	if result.Yearly[0].PlanName != "Solo" || *result.CreditTopups[0].IntroAmountCents != 499 {
		t.Fatal("public plan lists share mutable values")
	}
}

func TestPresentationPricingPreservesUnavailableOwner(t *testing.T) {
	if sanitizePresentationPricing(nil) != nil {
		t.Fatal("nil owner became a pricing offer")
	}
	result := sanitizePresentationPricing(&commerce.PricingOverview{})
	if result.Bundle != nil || len(result.Monthly)+len(result.Yearly)+len(result.CreditTopups) != 0 {
		t.Fatal("empty owner became a pricing offer")
	}
}
