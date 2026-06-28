package summarize

import "testing"

func TestModelPolicy_CoercesUnsafeStoredModels(t *testing.T) {
	cases := []struct {
		name       string
		model      string
		installed  []OllamaModel
		wantModel  string
		wantReason ModelDecisionReason
	}{
		{name: "empty", model: "", wantModel: DefaultSummarizeModel, wantReason: ModelDecisionEmptyDefault},
		{name: "reasoning", model: "fixture-reasoning-model", wantModel: DefaultSummarizeModel, wantReason: ModelDecisionUnsafeReasoning},
		{
			name:       "missing with installed evidence",
			model:      "missing:latest",
			installed:  []OllamaModel{{Name: "fixture-chat-small"}},
			wantModel:  DefaultSummarizeModel,
			wantReason: ModelDecisionMissingFallback,
		},
		{name: "safe", model: "chat.small", wantModel: "chat.small", wantReason: ModelDecisionKept},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CoerceUnsafeStoredModel(tc.model, tc.installed)
			if got.Model != tc.wantModel || got.Reason != tc.wantReason {
				t.Fatalf("CoerceUnsafeStoredModel() = %+v, want model=%q reason=%q", got, tc.wantModel, tc.wantReason)
			}
		})
	}
}

func TestModelPolicy_MergeSummarizeModelsMarksStatus(t *testing.T) {
	models := MergeSummarizeModels([]OllamaModel{
		{Name: "fixture-chat-model", SizeBytes: 123, ParameterSize: "3B"},
		{Name: "fixture-reasoning-model", SizeBytes: 456, ParameterSize: "4B"},
		{Name: "all-minilm:latest", SizeBytes: 789},
	})
	byID := map[string]SummarizeModelInfo{}
	for _, model := range models {
		byID[model.ID] = model
	}
	defaultRole := byID[DefaultSummarizeModel]
	if defaultRole.Installed || !defaultRole.DefaultEligible || defaultRole.PullCommand != "resource-ollama ensure --role "+DefaultSummarizeModel {
		t.Fatalf("default role status = %+v, want missing default-eligible role", defaultRole)
	}
	chat := byID["fixture-chat-model"]
	if !chat.Installed || chat.DefaultEligible || chat.Reasoning {
		t.Fatalf("chat fixture status = %+v, want installed direct model and not default-eligible", chat)
	}
	reasoning := byID["fixture-reasoning-model"]
	if !reasoning.Installed || reasoning.DefaultEligible || !reasoning.Reasoning {
		t.Fatalf("reasoning fixture status = %+v, want installed reasoning and not default-eligible", reasoning)
	}
	embedding := byID["all-minilm:latest"]
	if !embedding.Installed || embedding.DefaultEligible {
		t.Fatalf("all-minilm status = %+v, want installed but not default-eligible", embedding)
	}
}

func TestSelectorIsRole(t *testing.T) {
	cases := []struct {
		selector string
		want     bool
	}{
		{"summarize.default", true}, // logical role
		{"chat.small", true},        // logical role
		{"qwen3.5:9b", false},       // concrete tag
		{"nomic-embed-text:latest", false},
		{"  summarize.default  ", true}, // trimmed
		{"", false},                     // empty is not a role
		{"   ", false},
	}
	for _, tc := range cases {
		if got := SelectorIsRole(tc.selector); got != tc.want {
			t.Errorf("SelectorIsRole(%q) = %v, want %v", tc.selector, got, tc.want)
		}
	}
}
