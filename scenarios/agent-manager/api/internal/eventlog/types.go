// Package eventlog defines the typed-operational event taxonomy for
// agent-manager: payload struct definitions, schema versioning, the
// (event_type, schema_version) → Go-type dispatch table, and helpers for
// constructing typed *domain.RunEvent values.
//
// Events the eventlog package owns are the structured, queryable signals
// that drive fallback insights, model/runner health, and the future stats
// engine — runner fallback walks, model fallback walks, sandbox lifecycle
// outcomes, heartbeat misses, checkpoint failures, and model/runner health
// transitions. Freeform diagnostic messages stay on domain.LogEventData.
//
// Why a separate package?
//   - Screaming architecture: "what happened" lives in eventlog; "why does
//     a runner reject" lives in fallback (Phase 2); "what patterns emerge"
//     lives in stats (Phase 3). Each package has one job.
//   - Forward-compatible payload evolution: adding a field to an existing
//     payload is a no-op (older readers ignore unknown JSON fields).
//     Renaming a field bumps the schema version and registers a new entry
//     in the dispatch table; old events keep decoding through the old
//     entry indefinitely.
//
// DOC: scenarios/agent-manager/docs/internal/EVENT_TAXONOMY.md — full event
// list, payload schemas, and version policy.
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md — eventlog seam.
package eventlog

import (
	"time"

	"agent-manager/internal/domain"
)

// SchemaVersionDefault is the schema_version used by every Phase 1 typed
// event. Future renames bump per-event; new optional fields do not.
const SchemaVersionDefault = 1

// FallbackReason is a forward-declared placeholder for the canonical
// fallback/classification taxonomy that the fallback package owns from
// Phase 2 onward. Phase 1 emitters accept a string so phases can populate
// it from existing signals (runner.IsAvailable message, exec error string)
// without depending on a Reason enum that doesn't exist yet.
//
// Phase 2 replaces this alias with `fallback.Reason` (a closed enum). The
// JSON shape is unaffected — both are string-typed at rest.
type FallbackReason = string

// Health status values used by *.health.transition payloads. Phase 2
// replaces the underlying string type with the health package's typed
// enum; the JSON shape is unaffected.
const (
	HealthStatusOK      = "ok"
	HealthStatusUnknown = "unknown"
	HealthStatusFailed  = "failed"
)

// SandboxOperationKind names the lifecycle action observed in a
// sandbox.operation event. The closed set matches the verbs the
// finalize/apply paths actually invoke.
const (
	SandboxOpDelete = "delete"
	SandboxOpStop   = "stop"
)

// HeartbeatTarget identifies which heartbeat write missed in a
// heartbeat.miss event. Different targets warrant different operator
// responses (run-row write vs. checkpoint ping), so it is an enum, not a
// freeform message.
const (
	HeartbeatTargetRun        = "run"
	HeartbeatTargetCheckpoint = "checkpoint"
)

// CheckpointFailureKind names which checkpoint write failed. Phase
// persistence and dedicated checkpoint saves are different code paths and
// have different recovery shapes.
const (
	CheckpointFailureSavePhase = "save_phase"
	CheckpointFailureSaveStep  = "save_step"
)

// =============================================================================
// PAYLOAD STRUCTS
// =============================================================================
//
// Each payload struct corresponds to exactly one (RunEventType,
// SchemaVersion) pair, registered in dispatch.go. Adding an optional
// field is non-breaking; renaming or removing a field requires a new
// payload struct + a new dispatch entry.
//
// Field ordering and json tags are part of the wire contract. Avoid
// reordering once a payload has been emitted in production.

// RunnerFallbackAttemptedPayload records one step of the runner fallback
// chain walk: the requested runner couldn't be acquired, and the executor
// is trying the next candidate.
type RunnerFallbackAttemptedPayload struct {
	From      string         `json:"from"`
	To        string         `json:"to"`
	Reason    FallbackReason `json:"reason"`
	AttemptNo int            `json:"attempt_no"`
}

// RunnerFallbackExhaustedPayload records that the runner fallback walk
// completed without finding an available candidate. Emitted in addition
// to the failing runner-acquire error event so consumers can distinguish
// "primary runner failed, fallback succeeded" from "every candidate
// failed".
type RunnerFallbackExhaustedPayload struct {
	Primary         string         `json:"primary"`
	CandidatesTried []string       `json:"candidates_tried,omitempty"`
	LastReason      FallbackReason `json:"last_reason"`
}

// ModelFallbackAttemptedPayload records one step of the preset chain walk
// inside ExecuteWithModelFallback: the previously-attempted model was
// rejected and the executor is moving to the next chain entry.
type ModelFallbackAttemptedPayload struct {
	From          string         `json:"from"`
	To            string         `json:"to"`
	Reason        FallbackReason `json:"reason"`
	AttemptNo     int            `json:"attempt_no"`
	ChainPosition int            `json:"chain_position"`
	ChainLength   int            `json:"chain_length"`
}

// ModelFallbackExhaustedPayload records that every entry in the preset
// chain was rejected. Emitted in addition to the terminal failure event so
// consumers can count exhaustions independently of generic execution
// failures.
type ModelFallbackExhaustedPayload struct {
	Preset     string         `json:"preset,omitempty"`
	Chain      []string       `json:"chain"`
	LastReason FallbackReason `json:"last_reason"`
}

// PolicyCandidateOutcome is the stable state vocabulary for one persisted
// policy candidate. Exhausted is the terminal sequence-level signal and points
// at the last candidate considered.
type PolicyCandidateOutcome string

const (
	PolicyCandidateOutcomeAttempted PolicyCandidateOutcome = "attempted"
	PolicyCandidateOutcomeSkipped   PolicyCandidateOutcome = "skipped"
	PolicyCandidateOutcomeFailed    PolicyCandidateOutcome = "failed"
	PolicyCandidateOutcomeSelected  PolicyCandidateOutcome = "selected"
	PolicyCandidateOutcomeExhausted PolicyCandidateOutcome = "exhausted"
)

// PolicyCandidateAttemptPayload is the complete per-candidate audit signal for
// snapshot-driven execution. Events always carry immutable revision/index
// provenance, including the terminal exhausted outcome.
type PolicyCandidateAttemptPayload struct {
	CatalogDigest   string                 `json:"catalog_digest"`
	SnapshotIndex   int                    `json:"snapshot_index"`
	Runner          string                 `json:"runner"`
	SelectionType   string                 `json:"selection_type"`
	Model           string                 `json:"model,omitempty"`
	Outcome         PolicyCandidateOutcome `json:"outcome"`
	Reason          string                 `json:"reason,omitempty"`
	FailureClass    string                 `json:"failure_class,omitempty"`
	ChallengerModel string                 `json:"challenger_model,omitempty"`
	CanaryArm       string                 `json:"canary_arm,omitempty"`
}

// ModelHealthTransitionPayload records that a (runner, model) pair has
// flipped status. Emitted by the runtime classification path in execute.go
// (Phase 2) and by the probe loop (Phase 2). The audit-table writes are a
// separate concern owned by internal/health/.
type ModelHealthTransitionPayload struct {
	Runner     string         `json:"runner"`
	Model      string         `json:"model"`
	FromStatus string         `json:"from_status"`
	ToStatus   string         `json:"to_status"`
	Reason     FallbackReason `json:"reason,omitempty"`
	Message    string         `json:"message,omitempty"`
}

// RunnerHealthTransitionPayload records a runner-level health change
// (e.g., codex binary disappeared, claude CLI fails its availability
// probe). Phase 2 introduces the persisted runner-health audit; Phase 1
// just defines the event shape.
type RunnerHealthTransitionPayload struct {
	Runner     string         `json:"runner"`
	FromStatus string         `json:"from_status"`
	ToStatus   string         `json:"to_status"`
	Reason     FallbackReason `json:"reason,omitempty"`
}

// SandboxOperationPayload records the outcome of one sandbox lifecycle
// action issued from finalize. Replaces the older "failed to delete
// sandbox: <err>" / "sandbox stopped (finalize)" string emissions.
type SandboxOperationPayload struct {
	Operation  string         `json:"operation"`
	Success    bool           `json:"success"`
	DurationMS int64          `json:"duration_ms,omitempty"`
	Reason     FallbackReason `json:"reason,omitempty"`
	Message    string         `json:"message,omitempty"`
}

// HeartbeatMissPayload records a heartbeat write that failed. Operators
// care about the failure rate and the kind of write that missed; the
// underlying error string is included for diagnostics but operationally
// significant aggregations key off Target + AttemptNo.
type HeartbeatMissPayload struct {
	Target        string     `json:"target"`
	AttemptNo     int        `json:"attempt_no"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	ErrorCode     string     `json:"error_code,omitempty"`
	Message       string     `json:"message,omitempty"`
}

// CheckpointFailurePayload records that a checkpoint write failed. Phase
// is the run phase the failure happened in; Step distinguishes
// phase-advance saves from explicit Save() calls inside a phase.
type CheckpointFailurePayload struct {
	Phase     string `json:"phase"`
	Step      string `json:"step"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message,omitempty"`
}

// RetryAttemptPayload records that an operation is being retried. Phase 1
// reserves the shape but does not emit it; Phase 2's classifier paths and
// the codec rewrite populate it as retries become structured.
type RetryAttemptPayload struct {
	Operation   string         `json:"operation"`
	AttemptNo   int            `json:"attempt_no"`
	MaxAttempts int            `json:"max_attempts"`
	Reason      FallbackReason `json:"reason"`
}

// =============================================================================
// PAYLOAD INTERFACE
// =============================================================================

// Payload is the closed set of typed-operational payload Go types. The
// dispatch table maps (RunEventType, SchemaVersion) to one of these.
//
// Implemented by every *Payload struct in this file via the marker method
// payloadMarker, which is unexported so external packages cannot register
// new payload types out from under the dispatch table.
type Payload interface {
	payloadMarker()
}

func (RunnerFallbackAttemptedPayload) payloadMarker() {}
func (RunnerFallbackExhaustedPayload) payloadMarker() {}
func (ModelFallbackAttemptedPayload) payloadMarker()  {}
func (ModelFallbackExhaustedPayload) payloadMarker()  {}
func (PolicyCandidateAttemptPayload) payloadMarker()  {}
func (ModelHealthTransitionPayload) payloadMarker()   {}
func (RunnerHealthTransitionPayload) payloadMarker()  {}
func (SandboxOperationPayload) payloadMarker()        {}
func (HeartbeatMissPayload) payloadMarker()           {}
func (CheckpointFailurePayload) payloadMarker()       {}
func (RetryAttemptPayload) payloadMarker()            {}

// EventTypeOf returns the RunEventType the given payload corresponds to.
// Used by builders to look up the on-wire event_type from a typed value
// without callers having to repeat themselves.
func EventTypeOf(p Payload) domain.RunEventType {
	switch p.(type) {
	case RunnerFallbackAttemptedPayload, *RunnerFallbackAttemptedPayload:
		return domain.EventTypeRunnerFallbackAttempted
	case RunnerFallbackExhaustedPayload, *RunnerFallbackExhaustedPayload:
		return domain.EventTypeRunnerFallbackExhausted
	case ModelFallbackAttemptedPayload, *ModelFallbackAttemptedPayload:
		return domain.EventTypeModelFallbackAttempted
	case ModelFallbackExhaustedPayload, *ModelFallbackExhaustedPayload:
		return domain.EventTypeModelFallbackExhausted
	case PolicyCandidateAttemptPayload, *PolicyCandidateAttemptPayload:
		return domain.EventTypePolicyCandidateAttempt
	case ModelHealthTransitionPayload, *ModelHealthTransitionPayload:
		return domain.EventTypeModelHealthTransition
	case RunnerHealthTransitionPayload, *RunnerHealthTransitionPayload:
		return domain.EventTypeRunnerHealthTransition
	case SandboxOperationPayload, *SandboxOperationPayload:
		return domain.EventTypeSandboxOperation
	case HeartbeatMissPayload, *HeartbeatMissPayload:
		return domain.EventTypeHeartbeatMiss
	case CheckpointFailurePayload, *CheckpointFailurePayload:
		return domain.EventTypeCheckpointFailure
	case RetryAttemptPayload, *RetryAttemptPayload:
		return domain.EventTypeRetryAttempt
	}
	return ""
}
