package orchestration

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestLatestTurnResultEventsExcludesEarlierContinuationHandoff(t *testing.T) {
	runID := uuid.New()
	event := func(turn, content string, sequence int64) *domain.RunEvent {
		evt := domain.NewProviderMessageEvent(runID, "assistant", content, domain.MessageEventData{TurnID: turn, Terminal: true, ProviderOrigin: "codex"})
		evt.Sequence = sequence
		return evt
	}
	events := []*domain.RunEvent{event("turn-1", "first handoff", 1), event("turn-2", "latest handoff", 2)}
	result := domain.ResolveRunResult(latestTurnResultEvents(events), true, 0, "completed")
	if result.Selection.Status != domain.FinalOutputSelectionSelected || result.FinalOutput != "latest handoff" {
		t.Fatalf("continued run result = %#v", result)
	}
}

func TestLatestTurnResultEventsPreservesSameTurnAmbiguity(t *testing.T) {
	runID := uuid.New()
	first := domain.NewProviderMessageEvent(runID, "assistant", "one", domain.MessageEventData{TurnID: "turn-2", Terminal: true})
	second := domain.NewProviderMessageEvent(runID, "assistant", "two", domain.MessageEventData{TurnID: "turn-2", Terminal: true})
	first.Sequence, second.Sequence = 1, 2
	result := domain.ResolveRunResult(latestTurnResultEvents([]*domain.RunEvent{first, second}), true, 0, "completed")
	if result.Selection.Status != domain.FinalOutputSelectionAmbiguous {
		t.Fatalf("same-turn ambiguity was hidden: %#v", result.Selection)
	}
}
