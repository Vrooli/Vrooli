package commerce

import (
	"testing"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type testPlanCatalog struct{}

func (testPlanCatalog) BundleKey() string { return "business_suite" }

func (testPlanCatalog) GetPricingOverview() (*shared.PricingOverview, error) { return nil, nil }

func (testPlanCatalog) GetPlanByPriceID(string) (*shared.PlanOption, error) { return nil, nil }

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
