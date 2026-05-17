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
		{name: "reasoning", model: "qwen3:4b", wantModel: DefaultSummarizeModel, wantReason: ModelDecisionUnsafeReasoning},
		{
			name:       "missing with installed evidence",
			model:      "missing:latest",
			installed:  []OllamaModel{{Name: "llama3.2:1b"}},
			wantModel:  "llama3.2:1b",
			wantReason: ModelDecisionMissingFallback,
		},
		{name: "safe", model: "llama3.2:3b", wantModel: "llama3.2:3b", wantReason: ModelDecisionKept},
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
		{Name: "llama3.2:3b", SizeBytes: 123, ParameterSize: "3B"},
		{Name: "qwen3:4b", SizeBytes: 456, ParameterSize: "4B"},
		{Name: "all-minilm:latest", SizeBytes: 789},
	})
	byID := map[string]SummarizeModelInfo{}
	for _, model := range models {
		byID[model.ID] = model
	}
	llama := byID["llama3.2:3b"]
	if !llama.Installed || !llama.DefaultEligible || llama.Reasoning {
		t.Fatalf("llama3.2:3b status = %+v, want installed default-eligible non-reasoning", llama)
	}
	qwen := byID["qwen3:4b"]
	if !qwen.Installed || qwen.DefaultEligible || !qwen.Reasoning {
		t.Fatalf("qwen3:4b status = %+v, want installed reasoning and not default-eligible", qwen)
	}
	gemma := byID["gemma3:4b"]
	if gemma.Installed || gemma.PullCommand != "ollama pull gemma3:4b" {
		t.Fatalf("gemma3:4b status = %+v, want missing with pull command", gemma)
	}
	embedding := byID["all-minilm:latest"]
	if !embedding.Installed || embedding.DefaultEligible {
		t.Fatalf("all-minilm status = %+v, want installed but not default-eligible", embedding)
	}
}
