package telemetry

import (
	"context"
	"testing"
	"time"

	"program-runtime/internal/bindings"

	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

type captureInvocationRecorder struct{ rows []bindings.Invocation }

func (r *captureInvocationRecorder) RecordInvocation(_ context.Context, in bindings.Invocation) error {
	r.rows = append(r.rows, in)
	return nil
}

func (r *captureInvocationRecorder) ListInvocations(context.Context, time.Time, string, string) ([]bindings.Invocation, error) {
	return append([]bindings.Invocation(nil), r.rows...), nil
}

func TestInvocationTelemetryPreservesLedgerLinkAndUnknownPhases(t *testing.T) {
	store := NewStore()
	ledger := &captureInvocationRecorder{}
	recorder := WithInvocationTelemetry(ledger, store)
	now := time.Date(2026, 11, 24, 6, 45, 0, 0, time.UTC)
	if err := recorder.RecordInvocation(context.Background(), bindings.Invocation{
		BindingID: "ai-gateway/inference/run", ProgramID: "program-1", SessionID: "session-1",
		Outcome: "failed", LatencyMS: 42, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if len(ledger.rows) != 1 || ledger.rows[0].InvocationID == "" {
		t.Fatalf("ledger invocation=%+v", ledger.rows)
	}
	events := store.List("session-1", telemetryv1.EventKind_BINDING_CALL_MEASURED)
	if len(events) != 1 {
		t.Fatalf("measured events=%d, want 1", len(events))
	}
	event := events[0]
	if event.GetProgramId() != "program-1" || event.GetBindingId() != "ai-gateway/inference/run" || event.GetBindingInvocationId() != ledger.rows[0].InvocationID {
		t.Fatalf("event linkage=(program %q, binding %q, invocation %q)", event.GetProgramId(), event.GetBindingId(), event.GetBindingInvocationId())
	}
	if event.BindingCallLatencyMs == nil || event.GetBindingCallLatencyMs() != 42 || event.GetBindingCallOutcome() != "failed" {
		t.Fatalf("binding measurement=%+v", event)
	}
	if event.ExecutionWallTimeMs != nil {
		t.Fatal("binding event must not claim an execution total")
	}
}
