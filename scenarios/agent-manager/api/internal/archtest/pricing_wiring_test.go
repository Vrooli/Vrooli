package archtest

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/wiring"
)

func TestCostTrackingCodecsHaveChargeSource(t *testing.T) {
	runners := wiring.NewRunners(testPricingService{})
	// INVARIANT: costTrackingRunnerHasChargeSource
	for _, registered := range runners.Registry.List() {
		if !registered.Capabilities().SupportsCostTracking {
			continue
		}
		aware, ok := registered.(interface{ HasChargeSource() bool })
		if !ok || !aware.HasChargeSource() {
			t.Fatalf("cost-tracking runner %s was constructed without a charge source", registered.Type())
		}
	}
}

type testPricingService struct{}

func (testPricingService) CalculateCost(_ context.Context, _ codecs.PricingCostRequest) (*codecs.PricingCostCalculation, error) {
	return &codecs.PricingCostCalculation{CostSource: "unpriced", ChargeReason: "test"}, nil
}
