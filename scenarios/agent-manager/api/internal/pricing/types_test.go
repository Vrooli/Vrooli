package pricing

import (
	"testing"
	"time"
)

func TestModelPricingCloneIsIndependent(t *testing.T) {
	input := 1.0
	model := &ModelPricing{InputTokenPrice: &input}
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

func TestPricingExpiry(t *testing.T) {
	past, future := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	if !(&ModelPricing{ExpiresAt: past}).IsExpired() || (&ModelPricing{ExpiresAt: future}).IsExpired() {
		t.Fatal("model pricing expiry must follow ExpiresAt")
	}
	if (&ManualPriceOverride{}).IsExpired() || (&ManualPriceOverride{ExpiresAt: &future}).IsExpired() || !(&ManualPriceOverride{ExpiresAt: &past}).IsExpired() {
		t.Fatal("an override without ExpiresAt is permanent; otherwise it follows ExpiresAt")
	}
}
