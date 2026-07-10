// Builders and Emitter for typed-operational events.
//
// Two layers:
//
//   - BuildEvent / NewEvent — pure functions that wrap a typed payload in
//     a *domain.RunEvent ready to be appended through any existing
//     EventSink (the per-run emit.Gate, the SQLiteStore directly, or a
//     test-only sink). Phases call these and route the result through the
//     same publishEvent helper as legacy LogEventData / ErrorEventData
//     emissions, so the single-emission-path invariant is preserved.
//
//   - Emitter — a thin per-sink wrapper that exposes typed Emit* methods.
//     Mirrors the swarm-manager eventlog.Emitter shape so future agents
//     coming from that codebase find the same API. Errors are logged but
//     never returned, matching the fire-and-forget contract of
//     phases.publishEvent.
//
// The schema_version on every emitted event is the latest version
// registered in the dispatch table for that event type. Callers never
// pick a version; the registry is the source of truth.

package eventlog

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// BuildEvent constructs a *domain.RunEvent carrying a TypedEventData
// payload built from the given typed payload. The event_type is derived
// from the payload's Go type via EventTypeOf, and the schema_version is
// the latest registered for that event type.
//
// Returns an error only if the payload Go type is not registered in the
// dispatch table or marshaling fails — both are programmer errors and
// should fail loudly during development.
func BuildEvent(runID uuid.UUID, payload Payload) (*domain.RunEvent, error) {
	eventType := EventTypeOf(payload)
	if eventType == "" {
		return nil, fmt.Errorf("eventlog: payload type %T is not registered", payload)
	}
	schemaVersion := LatestSchemaVersion(eventType)
	if schemaVersion == 0 {
		return nil, fmt.Errorf("eventlog: no schema_version registered for %s", eventType)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("eventlog: marshal %s: %w", eventType, err)
	}
	return &domain.RunEvent{
		ID:            uuid.New(),
		RunID:         runID,
		EventType:     eventType,
		Timestamp:     time.Now(),
		SchemaVersion: schemaVersion,
		Data:          &domain.TypedEventData{Type: eventType, Body: json.RawMessage(body)},
	}, nil
}

// MustBuildEvent is BuildEvent with a panic-on-error contract for test
// fixtures and callers that have already validated their payload types.
// Production code paths should use BuildEvent and surface the error.
func MustBuildEvent(runID uuid.UUID, payload Payload) *domain.RunEvent {
	evt, err := BuildEvent(runID, payload)
	if err != nil {
		panic(err)
	}
	return evt
}

// Sink is the minimum surface the Emitter needs from a destination — the
// per-run emit.Gate (passed by phases) and any test fake satisfy it.
//
// Defined here (instead of importing emit.Gate) so the eventlog package
// stays free of orchestration imports and remains usable from tests that
// drive event log writes without standing up a Gate.
type Sink interface {
	Emit(event *domain.RunEvent) error
}

// Emitter is a thin typed wrapper over a Sink. Methods are fire-and-forget;
// errors are logged with slog but never returned, matching the
// publishEvent contract phases already rely on.
type Emitter struct {
	sink  Sink
	runID uuid.UUID
}

// NewEmitter constructs an Emitter bound to a per-run Sink. A nil Sink is
// permitted and produces a no-op emitter (every Emit* call silently
// drops). Callers that want strict mode should construct the Emitter
// themselves and check the sink before passing it in.
func NewEmitter(sink Sink, runID uuid.UUID) *Emitter {
	return &Emitter{sink: sink, runID: runID}
}

// Emit dispatches a typed payload through the wrapped sink.
func (e *Emitter) Emit(payload Payload) {
	if e == nil || e.sink == nil {
		return
	}
	evt, err := BuildEvent(e.runID, payload)
	if err != nil {
		slog.Error("eventlog: build event", "error", err)
		return
	}
	if err := e.sink.Emit(evt); err != nil {
		slog.Error("eventlog: emit event", "event_type", evt.EventType, "run_id", e.runID, "error", err)
	}
}

// EmitRunnerFallbackAttempted records one runner-fallback step.
func (e *Emitter) EmitRunnerFallbackAttempted(p RunnerFallbackAttemptedPayload) {
	e.Emit(p)
}

// EmitRunnerFallbackExhausted records that the runner fallback walk
// completed without finding an available candidate.
func (e *Emitter) EmitRunnerFallbackExhausted(p RunnerFallbackExhaustedPayload) {
	e.Emit(p)
}

// EmitModelFallbackAttempted records one preset-chain step.
func (e *Emitter) EmitModelFallbackAttempted(p ModelFallbackAttemptedPayload) {
	e.Emit(p)
}

// EmitModelFallbackExhausted records preset-chain exhaustion.
func (e *Emitter) EmitModelFallbackExhausted(p ModelFallbackExhaustedPayload) {
	e.Emit(p)
}

// EmitPolicyCandidateAttempt records one persisted-candidate state transition.
func (e *Emitter) EmitPolicyCandidateAttempt(p PolicyCandidateAttemptPayload) {
	e.Emit(p)
}

// EmitModelHealthTransition records a (runner, model) status flip.
func (e *Emitter) EmitModelHealthTransition(p ModelHealthTransitionPayload) {
	e.Emit(p)
}

// EmitRunnerHealthTransition records a runner-level status flip.
func (e *Emitter) EmitRunnerHealthTransition(p RunnerHealthTransitionPayload) {
	e.Emit(p)
}

// EmitSandboxOperation records the outcome of a sandbox lifecycle action.
func (e *Emitter) EmitSandboxOperation(p SandboxOperationPayload) {
	e.Emit(p)
}

// EmitHeartbeatMiss records a heartbeat write that failed.
func (e *Emitter) EmitHeartbeatMiss(p HeartbeatMissPayload) {
	e.Emit(p)
}

// EmitCheckpointFailure records that a checkpoint write failed.
func (e *Emitter) EmitCheckpointFailure(p CheckpointFailurePayload) {
	e.Emit(p)
}

// EmitRetryAttempt records that an operation is being retried.
func (e *Emitter) EmitRetryAttempt(p RetryAttemptPayload) {
	e.Emit(p)
}
