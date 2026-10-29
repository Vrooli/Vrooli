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

func TestDetectEditRevertsAcceptsJSONDecodedFiles(t *testing.T) {
	events := []*domain.RunEvent{
		{ID: uuid.New(), Timestamp: time.Now(), Data: &domain.ToolCallEventData{ToolName: "file_change", Input: map[string]any{"files": []any{map[string]any{"path": "safe.txt", "kind": "add"}}}}},
		{ID: uuid.New(), Timestamp: time.Now().Add(time.Second), Data: &domain.ToolCallEventData{ToolName: "file_change", Input: map[string]any{"files": []any{map[string]any{"path": "safe.txt", "kind": "delete"}}}}},
	}
	if got := detectEditReverts(EpisodeDetectorContext{Events: events, EventsByID: eventMap(events)}); len(got) != 1 {
		t.Fatalf("got %d edit-revert episodes", len(got))
	}
}

// F-W5-4: a handoff continuation is the work after a terminal answer and
// before the next user message, and counts only those calls. Work after the
// operator's next request is a conversation and is no episode.
func TestHandoffContinuationCountsOnlyUnpromptedWorkAfterATerminalAnswer(t *testing.T) {
	now := time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)
	var events []*domain.RunEvent
	var facts []InvocationFact
	at := 0
	add := func(data domain.EventPayload) *domain.RunEvent {
		at++
		event := &domain.RunEvent{ID: uuid.New(), Timestamp: now.Add(time.Duration(at) * time.Second), Data: data}
		events = append(events, event)
		return event
	}
	call := func() {
		event := add(&domain.ToolCallEventData{ToolName: "shell", Input: map[string]any{"command": "ls"}})
		facts = append(facts, InvocationFact{CallEventID: event.ID.String()})
	}
	call()
	call()
	answer := add(&domain.MessageEventData{Role: "assistant", Content: "done", Terminal: true})
	call() // unprompted work after the terminal answer
	call()
	add(&domain.MessageEventData{Role: "user", Content: "now do the next thing"})
	call() // a conversation: the operator asked
	call()
	call()
	add(&domain.MessageEventData{Role: "assistant", Content: "done again", Terminal: true})
	add(&domain.MessageEventData{Role: "user", Content: "thanks, one more"})
	call()

	got := detectHandoffContinuation(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
	if len(got) != 1 {
		t.Fatalf("episodes=%+v, want one for the unprompted work only", got)
	}
	if got[0].Turns != 2 || got[0].EvidenceEventIDs[0] != answer.ID.String() || got[0].StartEventID != facts[2].CallEventID || got[0].EndEventID != facts[3].CallEventID {
		t.Fatalf("episode=%+v, want the 2 calls after the first answer", got[0])
	}
}
