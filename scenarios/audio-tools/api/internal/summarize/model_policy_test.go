package summarize

import (
	"testing"
)

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

func TestSummarizeRoleOptionsProjectsNewlyDeclaredInventoryRole(t *testing.T) {
	roles := []DeclaredSummaryRole{
		{ID: "summarize.audio_tools.experimental", Description: "New role description from the presentation catalog."},
		{ID: "summarize.faithful", Description: "Shared role must not be exposed by Audio Tools."},
		{ID: "summarize.audio_tools", Description: "Compact role caveat: 14/22 facts on one 300-word report."},
	}
	options := SummarizeRoleOptions(roles)
	if len(options) != 2 || options[0].ID != DefaultSummarizeModel || options[1].ID != "summarize.audio_tools.experimental" {
		t.Fatalf("role options = %+v, want sorted Audio Tools inventory roles only", options)
	}
	if options[1].Notes != roles[0].Description || options[0].Notes != roles[2].Description {
		t.Fatalf("role descriptions not projected for display: %+v", options)
	}
	if !IsAudioToolsSummarizeRole("summarize.audio_tools.experimental") || IsAudioToolsSummarizeRole("summarize.audio_tools_other") || IsAudioToolsSummarizeRole("summarize.faithful") {
		t.Fatal("Audio Tools role namespace filter should accept only its exact namespace and dotted descendants")
	}
}

func TestSummarizeRoleOptionsSuppliesFallbackDescription(t *testing.T) {
	options := SummarizeRoleOptions([]DeclaredSummaryRole{
		{ID: "summarize.audio_tools.experimental"},
	})
	if len(options) != 1 || options[0].Notes == "" || options[0].Family != "ai-gateway-role" || !options[0].DefaultEligible {
		t.Fatalf("role options = %+v, want declared role with safe display fallback", options)
	}
}

func TestSelectorIsRole(t *testing.T) {
	cases := []struct {
		selector string
		want     bool
	}{
		{DefaultSummarizeModel, true}, // logical Audio Tools role
		{"chat.small", true},          // logical role
		{"qwen3.5:9b", false},         // concrete tag
		{"nomic-embed-text:latest", false},
		{"  " + DefaultSummarizeModel + "  ", true}, // trimmed
		{"", false}, // empty is not a role
		{"   ", false},
	}
	for _, tc := range cases {
		if got := SelectorIsRole(tc.selector); got != tc.want {
			t.Errorf("SelectorIsRole(%q) = %v, want %v", tc.selector, got, tc.want)
		}
	}
}

func TestModelPolicy_Eligibility(t *testing.T) {
	if !IsDefaultEligibleSummarizeModel("chat.small") || IsDefaultEligibleSummarizeModel("fixture-reasoning-model") || IsDefaultEligibleSummarizeModel("") {
		t.Fatal("default eligibility must reject empty and reasoning selectors")
	}
}
