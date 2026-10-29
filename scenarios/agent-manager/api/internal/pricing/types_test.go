package pricing

import (
	"testing"
	"time"
)

func TestModelPricingComponentAccessorsAndCloneAreIndependent(t *testing.T) {
	prices := []float64{1, 2, 3, 4, 5, 6}
	components := AllComponents()
	if len(components) != len(prices) {
		t.Fatalf("components=%v", components)
	}
	model := &ModelPricing{}
	for index, component := range components {
		model.SetPrice(component, &prices[index])
		model.SetSource(component, SourceProviderAPI)
		if got := model.GetPrice(component); got == nil || *got != prices[index] {
			t.Fatalf("component %s price=%v", component, got)
		}
		if got := model.GetSource(component); got != SourceProviderAPI {
			t.Fatalf("component %s source=%s", component, got)
		}
	}
	if model.GetPrice(PricingComponent("unknown")) != nil || model.GetSource(PricingComponent("unknown")) != SourceUnknown {
		t.Fatal("unknown component must fail closed")
	}
	clone := model.Clone()
	if clone == nil || clone == model {
		t.Fatal("clone was not allocated")
	}
	*clone.InputTokenPrice = 99
	if *model.InputTokenPrice == 99 {
		t.Fatal("clone mutated original price pointer")
	}
	if (*ModelPricing)(nil).Clone() != nil {
		t.Fatal("nil pricing clone must remain nil")
	}
}

func TestPricingExpiryAndDefaults(t *testing.T) {
	now := time.Now()
	if !(&ModelPricing{ExpiresAt: now.Add(-time.Second)}).IsExpired() || (&ModelPricing{ExpiresAt: now.Add(time.Second)}).IsExpired() {
		t.Fatal("model pricing expiry boundary incorrect")
	}
	if (&ManualPriceOverride{}).IsExpired() || !(&ManualPriceOverride{ExpiresAt: ptrTime(now.Add(-time.Second))}).IsExpired() || (&ManualPriceOverride{ExpiresAt: ptrTime(now.Add(time.Second))}).IsExpired() {
		t.Fatal("manual override expiry boundary incorrect")
	}
	defaults := DefaultPricingSettings()
	if defaults.ProviderCacheTTL != 6*time.Hour {
		t.Fatalf("defaults=%+v", defaults)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
