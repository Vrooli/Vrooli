package intelligence

import (
	"encoding/json"
	"testing"
)

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

func TestDefaultRolePoliciesLeavesAudioToItsFeatureOwner(t *testing.T) {
	if _, ok := DefaultRolePolicies()["audio.summarize"]; ok {
		t.Fatal("Audio summary must use its declared feature, not an LPBS model policy")
	}
}

func TestCodeRoleHasMeteredPolicyAndToolDeclarationsIncreaseReservation(t *testing.T) {
	policy, ok := DefaultRolePolicies()["code.default"]
	if !ok || policy.Model != "openai/gpt-4o-mini" || policy.Pricing.PromptCostPer1K <= 0 || policy.Pricing.CompletionCostPer1K <= 0 {
		t.Fatalf("code.default policy = %+v, want explicit priced model policy", policy)
	}
	messages := []AIMessage{{Role: "assistant", ToolCalls: []OpenRouterToolCall{{ID: "call_1", Type: "function", Function: OpenRouterToolFunction{Name: "read_file", Arguments: `{"path":"src/main.go"}`}}}}}
	base := EstimateInferenceTokens(messages, nil, nil, 500)
	withTools := EstimateInferenceTokens(messages, []json.RawMessage{json.RawMessage(`{"type":"function","function":{"name":"read_file","parameters":{"type":"object","properties":{"path":{"type":"string"}}}}}`)}, json.RawMessage(`"auto"`), 500)
	if withTools.Prompt <= base.Prompt {
		t.Fatalf("prompt reservation estimate without/with tools = %d/%d, want tool declarations counted", base.Prompt, withTools.Prompt)
	}
}
