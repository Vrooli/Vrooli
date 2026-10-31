package bundles

import (
	"testing"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

func TestBuildCatalogResponsePreservesLegacyJSONProjection(t *testing.T) {
	response, err := BuildCatalogResponse([]commerce.BundleCatalogEntry{{
		Bundle: &commerce.BundleProduct{BundleKey: "starter", Name: "Starter", CreditsPerUsd: 100},
		Prices: []*commerce.PlanOption{{PlanName: "Starter monthly", BillingInterval: shared.BillingInterval_BILLING_INTERVAL_MONTH, IntroType: shared.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT, IntroPeriods: 14, PlanRank: 2, BonusType: " signup ", Kind: shared.PlanKind_PLAN_KIND_SUBSCRIPTION}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Bundles) != 1 || response.Bundles[0].Bundle.BundleKey != "starter" {
		t.Fatalf("catalog projection = %#v", response)
	}
	price := response.Bundles[0].Prices[0]
	if price.BillingInterval != "month" || price.IntroType == nil || *price.IntroType != "flat_amount" || price.BonusType == nil || *price.BonusType != "signup" || price.Kind == nil || *price.Kind != "subscription" {
		t.Fatalf("price projection lost legacy fields: %#v", price)
	}
}

func TestBuildCatalogResponseRejectsMissingBundle(t *testing.T) {
	if _, err := BuildCatalogResponse([]commerce.BundleCatalogEntry{{}}); err == nil {
		t.Fatal("expected missing bundle to be rejected")
	}
}
