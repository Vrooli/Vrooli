package orchestration

import (
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

func TestPersistedRunSummaryUsesProviderReceiptInsteadOfAssistantMessageCount(t *testing.T) {
	runID := uuid.New()
	store := mocks.NewFakeEventStore()
	var events []*domain.RunEvent
	for i := 0; i < 6; i++ {
		events = append(events, domain.NewProviderMessageEvent(runID, "assistant", "progress", domain.MessageEventData{TurnID: "original:turn-1", ProviderOrigin: "codex"}))
	}
	events = append(events, &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{InputTokens: 89765, CacheReadTokens: 1108224, OutputTokens: 7080, Turns: 1, ReconciliationAuthority: true}})
	if err := store.Append(t.Context(), runID, events...); err != nil {
		t.Fatal(err)
	}
	_, summary, err := resolvePersistedRunResult(t.Context(), store, runID, true, 0, "completed")
	if err != nil || summary.TurnsUsed != 1 || summary.TokensUsed != 1205069 {
		t.Fatalf("persisted summary inflated provider usage: %+v %v", summary, err)
	}
}

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
