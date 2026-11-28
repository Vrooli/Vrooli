// Package jobs is image-tools' server-owned, durable async job system. It
// mirrors the test-genie run-lifecycle philosophy: a submit returns a job id +
// ETA immediately, work runs under a server-lifetime context (so a client
// disconnect never destroys it), callers block ONCE on Wait (no polling), and
// progress streams over a subscription for SSE. One priority scheduler admits
// local and remote work; capacity leases determine safe local concurrency.
package jobs

import (
	"context"
	"time"
)

// MetaGatewayExecutionEvidence is the only runner metadata the manager may
// retain when an AI job finalizes as failed or canceled.
const MetaGatewayExecutionEvidence = "gateway_execution_evidence_v1"

// MetaProvisioningEvidence retains the finalized per-model preparation report,
// including partial outcomes when preparation fails or is canceled.
const MetaProvisioningEvidence = "provisioning_evidence_v1"

// MetaRoutePrefix prefixes the routing record (where a job ran, why, and its
// spend caps). Failed and canceled jobs keep these keys; see failedExecutionEvidence.
const MetaRoutePrefix = "route_"

// MetaCapacityRelease and MetaCapacityReleaseWarning record the outcome of the
// job's capacity cleanup. A job whose work succeeded keeps its result when the
// release could not be confirmed; these keys carry what happened instead.
const (
	MetaCapacityRelease        = "capacity_release"
	MetaCapacityReleaseWarning = "capacity_release_warning"
)

// Reroute is a Rerouter's decision to move a queued job onto another route
// before it starts: its payload, lane and ETA are replaced and the job keeps
// its identity, creation time and queue position rules.
type Reroute struct {
	Payload          []byte
	Lane             Lane
	EstimatedSeconds int
	// Message is shown as the job's status when the move is applied.
	Message string
	// Meta is merged into the job's meta when the move is applied, so facts that
	// must outlive the move (and be queryable while the job still waits) are
	// durable at once.
	Meta map[string]string
}

// RouteBackError is returned by an Admitter or Runner whose route was refused
// before it did any work or billing (for example a cloud provider out of
// credits). The manager releases the admission and puts the job back on the
// route in Move instead of failing it. It is honoured only for a job that was
// not canceled; it never moves a job that produced output.
type RouteBackError struct {
	Move  Reroute
	Cause error
}

func (e *RouteBackError) Error() string {
	if e.Cause == nil {
		return "jobs: route refused; returning the job to its previous route"
	}
	return "jobs: route refused; returning the job to its previous route: " + e.Cause.Error()
}

func (e *RouteBackError) Unwrap() error { return e.Cause }

// Rerouter inspects a job that has not started running, either queued or still
// inside admission (waiting on capacity or preparing a model), and returns a
// Reroute to move it, nil to leave it, or an error when it wanted to move it
// but could not (the manager logs it and asks again later). It runs outside all
// manager locks and must be idempotent: the manager applies the result only if
// the job has still not started running, so a running job is never moved and a
// route is never run twice. A job moved out of admission has its admission
// cancelled and released first; the new route starts only after that cleanup.
type Rerouter func(ctx context.Context, job Job) (*Reroute, error)

// State is a job's lifecycle state.
type State string

const (
	// StateQueued: accepted, waiting for a worker.
	StateQueued State = "queued"
	// StateAdmitting: the scheduler is obtaining the job's resource leases.
	StateAdmitting State = "admitting"
	// StateRunning: a worker is executing it.
	StateRunning State = "running"
	// StateCanceling: cancellation was requested; owned work and claims are
	// still being stopped. Only cleanup completion makes it terminal.
	StateCanceling State = "canceling"
	// StateSucceeded: finished successfully (terminal).
	StateSucceeded State = "succeeded"
	// StateFailed: finished with an error, incl. restart-interruption (terminal).
	StateFailed State = "failed"
	// StateCanceled: aborted by request before completing (terminal).
	StateCanceled State = "canceled"
)

// Terminal reports whether s is a final state.
func (s State) Terminal() bool {
	switch s {
	case StateSucceeded, StateFailed, StateCanceled:
		return true
	default:
		return false
	}
}

// Lane is a compatibility scheduling class, not an execution-device claim.
// Local classes share a capacity-admitted priority queue. Remote work has a
// separate concurrency bound so local contention cannot block it.
type Lane string

const (
	// LaneGPU is the legacy class for model work, which may use CPU or GPU.
	LaneGPU Lane = "gpu"
	// LaneCPU runs cheap work concurrently.
	LaneCPU Lane = "cpu"
	// LaneNetwork runs remote-tier work on a bounded worker pool, independent of
	// the serialized local queue.
	LaneNetwork Lane = "network"
)

// Priority uses the platform-capacity vocabulary. Within a priority jobs are
// FIFO; the scheduler ages queued jobs to prevent batch starvation.
type Priority string

const (
	PriorityInteractive Priority = "interactive"
	PriorityService     Priority = "service"
	PriorityBatch       Priority = "batch"
)

// Spec describes a unit of work to submit.
type Spec struct {
	// Operation is the op name (e.g. "upscale", "resize"). Informational +
	// surfaced in status; the Runner decides what to do with it.
	Operation string
	// Lane selects local or remote concurrency accounting.
	Lane     Lane
	Priority Priority
	// Payload is opaque JSON the Runner interprets (request params, refs).
	Payload []byte
	// ClientOperationID and RequestFingerprint deduplicate caller retries when set.
	ClientOperationID  string
	RequestFingerprint string
	// EstimatedSeconds is the caller's pre-run estimate, surfaced as the initial
	// ETA. 0 means "unknown".
	EstimatedSeconds int
	// ExecutionTimeoutSeconds bounds work after a lane worker starts it; 0 is unbounded.
	ExecutionTimeoutSeconds int
}

// Job is the persisted, observable record of a submitted unit of work.
type Job struct {
	ID        string
	Operation string
	Lane      Lane
	Priority  Priority
	State     State
	Progress  int    // 0..100
	Message   string // latest progress / status message
	Error     string // failure detail when State == failed
	ResultRef string // output ref (blob key / path) when succeeded
	// Meta is the backend's record of the run: which model served it, which
	// tier it ran on, and what the route cost. Successful jobs retain backend
	// metadata. Failed/canceled jobs retain only the approved partial Gateway
	// execution-evidence entry, when present. See Result.Meta.
	Meta               map[string]string
	Payload            []byte
	ClientOperationID  string
	RequestFingerprint string
	// TestMode pins persistence and execution to the active routed test pool.
	TestMode                bool
	EstimatedSeconds        int
	ExecutionTimeoutSeconds int
	CreatedAt               time.Time
	StartedAt               *time.Time
	FinishedAt              *time.Time
}

// HeavyJobObservation is the image-tools owner's report-only view of local
// model/image queue work. The legacy GPU lane is not observed device placement.
// WaiterCount counts live server-side WaitJob/WatchJob calls.
type HeavyJobObservation struct {
	OwnerID     string    `json:"owner_id"`
	JobID       string    `json:"job_id"`
	Operation   string    `json:"operation"`
	State       string    `json:"state"`
	StartedAt   time.Time `json:"started_at"`
	WaiterCount int       `json:"waiter_count"`
	Heavy       bool      `json:"heavy"`
}

// HeavyJobSnapshot is complete for the image-tools manager instance only.
type HeavyJobSnapshot struct {
	ObservedAt time.Time             `json:"observed_at"`
	Complete   bool                  `json:"complete"`
	Jobs       []HeavyJobObservation `json:"jobs"`
}

// ProgressEvent is one progress update streamed to subscribers (SSE).
type ProgressEvent struct {
	JobID    string
	State    State
	Progress int
	Message  string
	At       time.Time
}
