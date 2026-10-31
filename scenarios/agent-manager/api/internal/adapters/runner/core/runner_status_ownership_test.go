package core

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func assertRunnerEmittedNoTerminalStatus(t *testing.T, events []*domain.RunEvent) {
	t.Helper()
	for _, event := range events {
		status, ok := event.Data.(*domain.StatusEventData)
		if !ok {
			continue
		}
		if domain.RunStatus(status.NewStatus).IsTerminal() {
			t.Fatalf("runner invocation claimed terminal Run status %q: %+v", status.NewStatus, status)
		}
	}
}

func TestContinueDoesNotEmitTerminalStatusAfterParkWinsDuringInvocation(t *testing.T) {
	codec := newFakeCodec()
	launcher := &fakeLauncher{stdout: `{"type":"message","content":"park before return"}` + "\n"}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()
	parked := false
	sink.onEmit = func(evt *domain.RunEvent) {
		if parked {
			return
		}
		if _, ok := evt.Data.(*domain.MessageEventData); !ok {
			return
		}
		parked = true
		if err := sink.Emit(domain.NewStatusEvent(runID, "running", "parked", "owner parked while provider was returning")); err != nil {
			t.Errorf("publish synthetic owner park in test sink: %v", err)
		}
	}

	_, err := r.Continue(context.Background(), runner.ContinueRequest{
		RunID: runID, SessionID: "session", Prompt: "continue", EventSink: sink,
		ResolvedConfig: domain.DefaultRunConfig(),
	})
	if err != nil {
		t.Fatalf("Continue returned error: %v", err)
	}
	if !parked {
		t.Fatal("test sink did not publish the parked owner transition during the invocation")
	}
	events := sink.snapshot()
	assertRunnerEmittedNoTerminalStatus(t, events)
	var parkedEventSeen bool
	for _, event := range events {
		if status, ok := event.Data.(*domain.StatusEventData); ok && status.NewStatus == string(domain.RunStatusParked) {
			parkedEventSeen = true
		}
	}
	if !parkedEventSeen {
		t.Fatal("owner park transition missing from event stream")
	}
}
