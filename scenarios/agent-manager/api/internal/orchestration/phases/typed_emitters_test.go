// Tests that the typed-event helpers in emitters.go land structured
// payloads in the event store. These assertions guard the contract that
// fallback / sandbox-op / heartbeat / checkpoint signals are queryable
// without parsing log strings — the whole point of Phase 1.

package phases

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

// makeDeps wires a Deps with just the FakeEventStore — sufficient for
// asserting event emission without standing up a Gate, repository, or
// broadcaster.
func makeDeps() (Deps, *mocks.FakeEventStore) {
	store := mocks.NewFakeEventStore()
	return Deps{Events: store}, store
}

func decodeTypedPayload(t *testing.T, evt *domain.RunEvent) eventlog.Payload {
	t.Helper()
	body, ok := evt.Data.(*domain.TypedEventData)
	if !ok {
		t.Fatalf("event %s carries %T, want *TypedEventData", evt.EventType, evt.Data)
	}
	payload, err := eventlog.Decode(evt.EventType, evt.SchemaVersion, body.Body)
	if err != nil {
		t.Fatalf("decode %s: %v", evt.EventType, err)
	}
	return payload
}

func TestEmitRunnerFallbackAttempted_LandsTypedPayload(t *testing.T) {
	deps, store := makeDeps()
	runID := uuid.New()

	EmitRunnerFallbackAttempted(context.Background(), deps, runID, eventlog.RunnerFallbackAttemptedPayload{
		From:      "claude-code",
		To:        "codex",
		Reason:    "binary missing",
		AttemptNo: 1,
	})

	events := store.TypedEvents(runID, domain.EventTypeRunnerFallbackAttempted)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	payload, ok := decodeTypedPayload(t, events[0]).(*eventlog.RunnerFallbackAttemptedPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", decodeTypedPayload(t, events[0]))
	}
	if payload.From != "claude-code" || payload.To != "codex" || payload.AttemptNo != 1 {
		t.Errorf("payload mismatch: %+v", payload)
	}
}

func TestEmitRunnerFallbackExhausted_LandsTypedPayload(t *testing.T) {
	deps, store := makeDeps()
	runID := uuid.New()

	EmitRunnerFallbackExhausted(context.Background(), deps, runID, eventlog.RunnerFallbackExhaustedPayload{
		Primary:         "claude-code",
		CandidatesTried: []string{"codex", "opencode"},
		LastReason:      "all unavailable",
	})

	events := store.TypedEvents(runID, domain.EventTypeRunnerFallbackExhausted)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	payload, ok := decodeTypedPayload(t, events[0]).(*eventlog.RunnerFallbackExhaustedPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", decodeTypedPayload(t, events[0]))
	}
	if payload.Primary != "claude-code" || len(payload.CandidatesTried) != 2 {
		t.Errorf("payload mismatch: %+v", payload)
	}
}

func TestEmitModelFallbackAttempted_LandsTypedPayload(t *testing.T) {
	deps, store := makeDeps()
	runID := uuid.New()

	EmitModelFallbackAttempted(context.Background(), deps, runID, eventlog.ModelFallbackAttemptedPayload{
		From:          "sonnet-4",
		To:            "haiku",
		Reason:        "rate limited",
		AttemptNo:     2,
		ChainPosition: 2,
		ChainLength:   3,
	})

	events := store.TypedEvents(runID, domain.EventTypeModelFallbackAttempted)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	payload, ok := decodeTypedPayload(t, events[0]).(*eventlog.ModelFallbackAttemptedPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", decodeTypedPayload(t, events[0]))
	}
	if payload.ChainLength != 3 || payload.ChainPosition != 2 {
		t.Errorf("payload mismatch: %+v", payload)
	}
}

func TestEmitHeartbeatMiss_LandsTypedPayload(t *testing.T) {
	deps, store := makeDeps()
	runID := uuid.New()

	EmitHeartbeatMiss(context.Background(), deps, runID, eventlog.HeartbeatMissPayload{
		Target:    eventlog.HeartbeatTargetCheckpoint,
		AttemptNo: 1,
		Message:   "db locked",
	})

	events := store.TypedEvents(runID, domain.EventTypeHeartbeatMiss)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	payload, ok := decodeTypedPayload(t, events[0]).(*eventlog.HeartbeatMissPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", decodeTypedPayload(t, events[0]))
	}
	if payload.Target != eventlog.HeartbeatTargetCheckpoint {
		t.Errorf("Target = %s, want %s", payload.Target, eventlog.HeartbeatTargetCheckpoint)
	}
}

func TestEmitCheckpointFailure_LandsTypedPayload(t *testing.T) {
	deps, store := makeDeps()
	runID := uuid.New()

	EmitCheckpointFailure(context.Background(), deps, runID, eventlog.CheckpointFailurePayload{
		Phase:   "running",
		Step:    eventlog.CheckpointFailureSavePhase,
		Message: "io error",
	})

	events := store.TypedEvents(runID, domain.EventTypeCheckpointFailure)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	payload, ok := decodeTypedPayload(t, events[0]).(*eventlog.CheckpointFailurePayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", decodeTypedPayload(t, events[0]))
	}
	if payload.Step != eventlog.CheckpointFailureSavePhase {
		t.Errorf("Step = %s, want %s", payload.Step, eventlog.CheckpointFailureSavePhase)
	}
}

func TestEmitSandboxOperation_LandsTypedPayload(t *testing.T) {
	deps, store := makeDeps()
	runID := uuid.New()

	EmitSandboxOperation(context.Background(), deps, runID, eventlog.SandboxOperationPayload{
		Operation:  eventlog.SandboxOpStop,
		Success:    true,
		DurationMS: 17,
		Reason:     "finalize",
	})

	events := store.TypedEvents(runID, domain.EventTypeSandboxOperation)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	payload, ok := decodeTypedPayload(t, events[0]).(*eventlog.SandboxOperationPayload)
	if !ok {
		t.Fatalf("unexpected payload type %T", decodeTypedPayload(t, events[0]))
	}
	if !payload.Success || payload.Operation != eventlog.SandboxOpStop {
		t.Errorf("payload mismatch: %+v", payload)
	}
}
