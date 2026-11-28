package telemetry

import (
	"context"
	"fmt"
	"time"

	"program-runtime/internal/bindings"

	"github.com/google/uuid"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

type invocationTelemetryRecorder struct {
	inner  bindings.InvocationRecorder
	events *Store
}

// WithInvocationTelemetry records a binding invocation in its durable ledger,
// then publishes an additive measurement event linked by program and ledger ID.
// The event contains only the binding-call duration already measured by the
// registry; it makes no claim about model time or thermal state.
func WithInvocationTelemetry(inner bindings.InvocationRecorder, events *Store) bindings.InvocationRecorder {
	return &invocationTelemetryRecorder{inner: inner, events: events}
}

func (r *invocationTelemetryRecorder) RecordInvocation(ctx context.Context, invocation bindings.Invocation) error {
	if r == nil || r.inner == nil {
		return fmt.Errorf("binding invocation recorder is unavailable")
	}
	if invocation.InvocationID == "" {
		invocation.InvocationID = "inv_" + uuid.NewString()
	}
	if err := r.inner.RecordInvocation(ctx, invocation); err != nil {
		return err
	}
	if r.events == nil || invocation.ProgramID == "" || invocation.ProgramID == "sweep" {
		return nil
	}
	latencyMS := invocation.LatencyMS
	occurredAt := invocation.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return r.events.AppendContext(ctx, &telemetryv1.ProgramEvent{
		EventId:              uuid.NewString(),
		OccurredAt:           occurredAt.UTC().Format(time.RFC3339Nano),
		Kind:                 telemetryv1.EventKind_BINDING_CALL_MEASURED,
		ProgramId:            invocation.ProgramID,
		SessionId:            invocation.SessionID,
		BindingId:            invocation.BindingID,
		BindingInvocationId:  invocation.InvocationID,
		BindingCallLatencyMs: &latencyMS,
		BindingCallOutcome:   invocation.Outcome,
	})
}

func (r *invocationTelemetryRecorder) ListInvocations(ctx context.Context, since time.Time, bindingID, scenario string) ([]bindings.Invocation, error) {
	if r == nil || r.inner == nil {
		return nil, fmt.Errorf("binding invocation recorder is unavailable")
	}
	return r.inner.ListInvocations(ctx, since, bindingID, scenario)
}
