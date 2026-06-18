// Package runs is the domain-scoped home for durable, server-owned remote runs
// (OT-P0-005). It owns the `runs` record and that run's append-only event
// history (logs / status transitions / exit code / artifact refs). A dispatched
// job (dispatch domain) becomes a Run here; the node-agent streams RunEvents
// back via the runs handler's node-facing ReportRunEvent, and the operator
// re-attaches by id with a block-once Wait (no polling) — reusing test-genie's
// run-lifecycle philosophy rather than reinventing run management.
//
// Layering mirrors the canonical Vrooli domain pattern (see registry):
//
//	HTTP → handler → Service (lifecycle policy + block-once coordination) → Repository (persists)
//	                     ↑                                                     ↑
//	                     FakeService (handler tests)                           FakeRepository (service tests)
//	                                                                           Real sqlite (repository tests)
//
// The Service additionally owns the in-memory coordination that durability
// needs: the block-once waiter registry (Wait wakes exactly once on a terminal
// transition) and the live event fan-out (Subscribe tails new events for the
// streaming "follow" verb). That coordination is ephemeral; the persisted run +
// event history is the durable source of truth a re-attaching client reads.
package runs

import (
	"fmt"
	"time"
)

// RunStatus is a run's lifecycle state. QUEUED/RUNNING are non-terminal;
// PASSED/FAILED/ABORTED are terminal.
type RunStatus int

const (
	// StatusUnspecified is the zero value; a persisted run never holds it.
	StatusUnspecified RunStatus = 0
	// StatusQueued — the run record exists and the JobPush has been delivered,
	// but the node has not yet reported it started.
	StatusQueued RunStatus = 1
	// StatusRunning — the node reported the job is executing.
	StatusRunning RunStatus = 2
	// StatusPassed — terminal: the job exited 0.
	StatusPassed RunStatus = 3
	// StatusFailed — terminal: the job exited non-zero.
	StatusFailed RunStatus = 4
	// StatusAborted — terminal: aborted (operator, timeout, or node loss).
	StatusAborted RunStatus = 5
)

// Terminal reports whether the status is a terminal one. WaitRun returns once a
// run reaches a terminal status; a late event for a terminal run is ignored
// (stale-completion safety).
func (s RunStatus) Terminal() bool {
	switch s {
	case StatusPassed, StatusFailed, StatusAborted:
		return true
	default:
		return false
	}
}

// String renders the status as a short lowercase label for logs/CLI.
func (s RunStatus) String() string {
	switch s {
	case StatusQueued:
		return "queued"
	case StatusRunning:
		return "running"
	case StatusPassed:
		return "passed"
	case StatusFailed:
		return "failed"
	case StatusAborted:
		return "aborted"
	default:
		return "unspecified"
	}
}

// EventKind discriminates a RunEvent's payload. It mirrors the channel.proto
// RunEventKind so the domain never imports proto; the handler translates at the
// boundary.
type EventKind int

const (
	// EventUnspecified is the zero value.
	EventUnspecified EventKind = 0
	// EventLog carries a chunk of combined stdout/stderr.
	EventLog EventKind = 1
	// EventStatus carries a human-readable lifecycle transition label.
	EventStatus EventKind = 2
	// EventExit carries the terminal process exit code.
	EventExit EventKind = 3
	// EventArtifactRef carries a device-sync-hub artifact reference.
	EventArtifactRef EventKind = 4
)

// Run is the durable, server-owned record of one dispatched job.
type Run struct {
	ID       string
	NodeID   string
	Scenario string
	Verb     string
	Args     []string
	Status   RunStatus
	ExitCode int32
	// TimeoutSeconds is the wall-clock budget after which the node aborts.
	TimeoutSeconds int64
	CreatedAt      time.Time
	// StartedAt/FinishedAt are zero until the corresponding transition.
	StartedAt    time.Time
	FinishedAt   time.Time
	ArtifactRefs []string
}

// RunEvent is one entry in a run's append-only event history.
type RunEvent struct {
	RunID       string
	Kind        EventKind
	Sequence    uint64
	LogChunk    string
	Status      string
	ExitCode    int32
	ArtifactRef string
	EmittedAt   time.Time
}

// CreateInput is the explicit DTO Service.Create accepts. The dispatch domain
// has already validated the verb against the allowlist + node scopes; the runs
// domain trusts the typed job and owns only the durable lifecycle.
type CreateInput struct {
	NodeID         string
	Scenario       string
	Verb           string
	Args           []string
	TimeoutSeconds int64
}

// ListFilter narrows ListRuns. Zero-value fields are not applied.
type ListFilter struct {
	NodeID string
	Limit  int
}

// ErrRunNotFound is the typed sentinel returned when no run matches an id.
type ErrRunNotFound struct {
	ID string
}

func (e ErrRunNotFound) Error() string {
	return fmt.Sprintf("run %q not found", e.ID)
}

// ErrInvalidRun is the typed sentinel returned on validation failure.
type ErrInvalidRun struct {
	Field  string
	Reason string
}

func (e ErrInvalidRun) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
