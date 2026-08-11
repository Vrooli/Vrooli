package intelligence

import "testing"

func TestAllowedModelsReturnsIndependentPolicyCopies(t *testing.T) {
	first := AllowedModels()
	first["unapproved/model"] = true

	if AllowedModels()["unapproved/model"] {
		t.Fatal("metered inference callers must not be able to mutate the allowed-model policy")
	}
}

func TestCalculateCostUsesModelAndDefaultPricing(t *testing.T) {
	pricing := DefaultModelPricing()
	if got, want := CalculateCost(pricing, "openai/gpt-4o-mini", 1_000, 1_000), int64(750_000); got != want {
		t.Fatalf("model cost = %d, want %d", got, want)
	}
	if got, want := CalculateCost(pricing, "unknown/model", 1_000, 1_000), int64(3_000_000); got != want {
		t.Fatalf("fallback cost = %d, want %d", got, want)
	}
}

func TestEstimateTokensHonorsExplicitCompletionLimit(t *testing.T) {
	estimate := EstimateTokens([]AIMessage{{Role: "user", Content: "abcdefgh"}}, 321)
	if estimate.Prompt != 7 { // (8 content + 4 role + 10 framing) / 4 × 1.5
		t.Fatalf("prompt estimate = %d, want 7", estimate.Prompt)
	}
	if estimate.Completion != 321 {
		t.Fatalf("explicit completion estimate = %d, want 321", estimate.Completion)
	}
}
