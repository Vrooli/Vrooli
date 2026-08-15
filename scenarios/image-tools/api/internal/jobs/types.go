// Package jobs is image-tools' server-owned, durable async job system. It
// mirrors the test-genie run-lifecycle philosophy: a submit returns a job id +
// ETA immediately, work runs under a server-lifetime context (so a client
// disconnect never destroys it), callers block ONCE on Wait (no polling), and
// progress streams over a subscription for SSE. Heavy GPU jobs are serialized
// (one at a time) while cheap CPU jobs run concurrently.
package jobs

import "time"

// State is a job's lifecycle state.
type State string

const (
	// StateQueued: accepted, waiting for a worker.
	StateQueued State = "queued"
	// StateRunning: a worker is executing it.
	StateRunning State = "running"
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

// Lane selects the execution lane. GPU jobs are serialized; CPU jobs run
// concurrently.
type Lane string

const (
	// LaneGPU serializes heavy GPU work (one at a time) to avoid VRAM contention.
	LaneGPU Lane = "gpu"
	// LaneCPU runs cheap work concurrently.
	LaneCPU Lane = "cpu"
)

// Spec describes a unit of work to submit.
type Spec struct {
	// Operation is the op name (e.g. "upscale", "resize"). Informational +
	// surfaced in status; the Runner decides what to do with it.
	Operation string
	// Lane selects GPU (serialized) or CPU (concurrent).
	Lane Lane
	// Payload is opaque JSON the Runner interprets (request params, refs).
	Payload []byte
	// EstimatedSeconds is the caller's pre-run estimate, surfaced as the initial
	// ETA. 0 means "unknown".
	EstimatedSeconds int
}

// Job is the persisted, observable record of a submitted unit of work.
type Job struct {
	ID        string
	Operation string
	Lane      Lane
	State     State
	Progress  int    // 0..100
	Message   string // latest progress / status message
	Error     string // failure detail when State == failed
	ResultRef string // output ref (blob key / path) when succeeded
	// Meta is the backend's record of the run: which model served it, which
	// tier it ran on, what the route cost. Set only on success, and only by
	// backends that report it. See Result.Meta.
	Meta             map[string]string
	Payload          []byte
	EstimatedSeconds int
	CreatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
}

// ProgressEvent is one progress update streamed to subscribers (SSE).
type ProgressEvent struct {
	JobID    string
	State    State
	Progress int
	Message  string
	At       time.Time
}
