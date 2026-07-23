package handlers

import (
	"testing"

	"agent-manager/internal/pricing"
)

func TestPricingPathDecodeAndComponentValidation(t *testing.T) {
	if got := decodePathParam("anthropic%2Fclaude%203"); got != "anthropic/claude 3" {
		t.Fatalf("decoded model = %q", got)
	}
	if got := decodePathParam("  model  "); got != "model" {
		t.Fatalf("trimmed model = %q", got)
	}
	for _, component := range []pricing.PricingComponent{pricing.ComponentInputTokens, pricing.ComponentOutputTokens, pricing.ComponentCacheRead, pricing.ComponentCacheCreation, pricing.ComponentWebSearch, pricing.ComponentServerToolUse} {
		if !isValidComponent(component) {
			t.Fatalf("component %q should be valid", component)
		}
	}
	if isValidComponent(pricing.PricingComponent("invalid")) {
		t.Fatal("invalid component accepted")
	}
}
