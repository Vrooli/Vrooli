// Package runmanager owns the durable lifecycle of a test-suite run inside the
// test-genie server. It decouples execution from any client request: a run is
// started under a server-lifetime context, tracked in an in-memory registry
// (cancel handle + event broadcaster + live phase state) backed by the durable
// run index, and can be followed, waited on, aborted, or inspected by run id —
// surviving client cancellation. It is the single engine every door funnels
// through (blocking REST, the SSE gateway, and the Connect run surface).
package runmanager

import (
	"test-genie/internal/orchestrator"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Canonical run-event kinds. This is the one event vocabulary all followers
// (the SSE gateway, the Connect FollowRun stream, the CLI) observe, regardless
// of transport. They map onto the orchestrator's lower-level ExecutionEvent
// types plus two synthesized boundaries (run_started / run_completed) and the
// quiet-phase heartbeat the manager emits on a timer.
const (
	// EventRunQueued is emitted when a run is admitted but parked behind the
	// global concurrency cap; EventRunStarted follows when it is promoted.
	EventRunQueued      = "run_queued"
	EventRunStarted     = "run_started"
	EventPhaseStarted   = "phase_started"
	EventPhaseProgress  = "phase_progress"
	EventPhaseHeartbeat = "phase_heartbeat"
	EventPhaseCompleted = "phase_completed"
	EventPhaseFailed    = "phase_failed"
	EventRunCompleted   = "run_completed"
)

// Event is a single canonical run event delivered to followers. Only the
// fields relevant to a given Kind are populated; ElapsedSeconds is always set
// (seconds since the run started, rounded to 0.1s).
type Event struct {
	Kind           string  `json:"event"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`

	// run_started / run_completed boundaries.
	RunID       string `json:"run_id,omitempty"`
	Scenario    string `json:"scenario,omitempty"`
	ArtifactDir string `json:"artifact_dir,omitempty"`
	Preset      string `json:"preset,omitempty"`

	// Phase-scoped fields (phase_started/progress/heartbeat/completed/failed).
	Phase           string  `json:"phase,omitempty"`
	PhaseIndex      int     `json:"phase_index,omitempty"`
	PhaseTotal      int     `json:"phase_total,omitempty"`
	Status          string  `json:"status,omitempty"`
	DurationSeconds int     `json:"duration_seconds,omitempty"`
	QuietSeconds    float64 `json:"quiet_seconds,omitempty"`
	Message         string  `json:"message,omitempty"`

	// Terminal fields (run_completed).
	Success bool   `json:"success,omitempty"`
	Verdict string `json:"verdict,omitempty"`
	Error   string `json:"error,omitempty"`

	// Phase-completed maturity standing + findings summary (Phase Capability
	// Contract). Present on phase_completed/phase_failed events for phases whose
	// provider declares a maturity ladder. Not serialized into the canonical line
	// stream; transports that carry the rich shape read them directly.
	PhasePresentation *commonv1.PhasePresentation  `json:"-"`
	FindingsSummary   *runspb.PhaseFindingsSummary `json:"-"`

	// Result is the full terminal result on run_completed (nil otherwise). It is
	// not serialized into the canonical line stream; transports that need the
	// rich shape (the blocking REST adapter) read it directly.
	Result *orchestrator.SuiteExecutionResult `json:"-"`
}
