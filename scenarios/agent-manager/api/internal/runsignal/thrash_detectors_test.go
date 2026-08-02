package runsignal

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestThrashDetectorsFindBoundedCyclesOnly(t *testing.T) {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	makeFact := func(offset int, fingerprint string) (InvocationFact, []*domain.RunEvent) {
		call := &domain.RunEvent{ID: uuid.New(), Timestamp: now.Add(time.Duration(offset) * time.Second), Data: &domain.ToolCallEventData{Input: map[string]any{"read": true}}}
		result := &domain.RunEvent{ID: uuid.New(), Timestamp: call.Timestamp.Add(time.Second), Data: &domain.ToolResultEventData{Success: true}}
		return InvocationFact{CallEventID: call.ID.String(), ResultEventID: result.ID.String(), Fingerprint: fingerprint}, []*domain.RunEvent{call, result}
	}
	var facts []InvocationFact
	var events []*domain.RunEvent
	for i, fp := range []string{"A", "B", "A", "B"} {
		fact, pair := makeFact(i*2, fp)
		facts, events = append(facts, fact), append(events, pair...)
	}
	oscillation := detectOscillations(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
	if len(oscillation) != 1 || oscillation[0].CycleCount != 1 || oscillation[0].RepeatedElement == "" {
		t.Fatalf("oscillation=%+v", oscillation)
	}

	changes := func(offset int, kind string) *domain.RunEvent {
		return &domain.RunEvent{ID: uuid.New(), Timestamp: now.Add(time.Duration(offset) * time.Second), Data: &domain.ToolCallEventData{ToolName: "file_change", Input: map[string]any{"files": []map[string]string{{"path": "safe.txt", "kind": kind}}}}}
	}
	editEvents := []*domain.RunEvent{changes(0, "add"), changes(2, "modify"), changes(4, "delete")}
	reverts := detectEditReverts(EpisodeDetectorContext{Events: editEvents, EventsByID: eventMap(editEvents)})
	if len(reverts) != 1 || reverts[0].CycleCount != 1 || reverts[0].RepeatedElement != "safe.txt" {
		t.Fatalf("reverts=%+v", reverts)
	}
	refinement := []*domain.RunEvent{changes(0, "add"), changes(2, "modify"), changes(4, "modify")}
	if got := detectEditReverts(EpisodeDetectorContext{Events: refinement, EventsByID: eventMap(refinement)}); len(got) != 0 {
		t.Fatalf("refinement=%+v", got)
	}
}
