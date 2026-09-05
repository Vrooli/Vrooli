package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestResolveRunResultEvidenceRules(t *testing.T) {
	runID := uuid.New()
	event := func(content string, evidence MessageEventData, sequence int64) *RunEvent {
		e := NewProviderMessageEvent(runID, "assistant", content, evidence)
		e.Sequence = sequence
		return e
	}

	tests := []struct {
		name   string
		events []*RunEvent
		status FinalOutputSelectionStatus
		output string
	}{
		{
			name: "terminal evidence beats later chatter",
			events: []*RunEvent{
				event("handoff", MessageEventData{Terminal: true, ProviderEventType: "result"}, 1),
				event("post-handoff chatter", MessageEventData{}, 2),
			},
			status: FinalOutputSelectionSelected,
			output: "handoff",
		},
		{
			name: "equally supported terminal messages abstain",
			events: []*RunEvent{
				event("one", MessageEventData{Terminal: true}, 1),
				event("two", MessageEventData{Terminal: true}, 2),
			},
			status: FinalOutputSelectionAmbiguous,
		},
		{
			name:   "single generic main message is conservative fallback",
			events: []*RunEvent{event("only", MessageEventData{}, 1)},
			status: FinalOutputSelectionSelected,
			output: "only",
		},
		{
			name:   "multiple generic messages are ambiguous not tail selected",
			events: []*RunEvent{event("one", MessageEventData{}, 1), event("two", MessageEventData{}, 2)},
			status: FinalOutputSelectionAmbiguous,
		},
		{
			name:   "single provider message without completion evidence is unavailable",
			events: []*RunEvent{event("draft", MessageEventData{ProviderOrigin: "opencode"}, 1)},
			status: FinalOutputSelectionUnavailable,
		},
		{
			name: "multiple provider messages without completion evidence are ambiguous",
			events: []*RunEvent{
				event("one", MessageEventData{ProviderOrigin: "codex"}, 1),
				event("two", MessageEventData{ProviderOrigin: "codex"}, 2),
			},
			status: FinalOutputSelectionAmbiguous,
		},
		{
			name:   "child output is never promoted",
			events: []*RunEvent{event("child", MessageEventData{Terminal: true, ParentMessageID: "tool-1"}, 1)},
			status: FinalOutputSelectionUnavailable,
		},
		{name: "missing assistant evidence is unavailable", status: FinalOutputSelectionUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveRunResult(tt.events, true, 0, "completed")
			if result.Selection.Status != tt.status {
				t.Fatalf("status = %q, want %q", result.Selection.Status, tt.status)
			}
			if result.FinalOutput != tt.output {
				t.Fatalf("final output = %q, want %q", result.FinalOutput, tt.output)
			}
			if result.Selection.AlgorithmVersion != FinalOutputResolverVersion {
				t.Fatalf("algorithm version = %q", result.Selection.AlgorithmVersion)
			}
		})
	}
}

func TestSummaryFromRunResultOnlyProjectsSelectedOutput(t *testing.T) {
	ambiguous := &RunResult{FinalOutput: "must not leak", Selection: FinalOutputSelection{Status: FinalOutputSelectionAmbiguous}}
	summary := SummaryFromRunResult(ambiguous, 2, 3, 4, 0.5)
	if summary.Description != "" {
		t.Fatalf("ambiguous output leaked into legacy summary: %q", summary.Description)
	}
	if summary.TurnsUsed != 2 || summary.TokensUsed != 3 || summary.ContextTokens != 4 || summary.CostEstimate != 0.5 {
		t.Fatalf("metrics projection = %#v", summary)
	}
}
