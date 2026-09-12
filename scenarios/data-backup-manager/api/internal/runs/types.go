// Package runs is the domain-scoped home for backup-run execution: one run is
// one execution of a plan that, for each member target × destination, captures
// the source, checks the destination cap, writes a kopia snapshot, applies
// retention, and records a per-target outcome. A single target's failure does
// not abort the others — the run is partial_failed. Runs surface
// last-success-per-target for the catalog and health views.
//
// The orchestration depends on narrow reader/effect seams this package
// declares (deps.go): PlanLookup, TargetLookup, DestinationLookup, the
// KopiaEngine, the sources Registry, an EventSink, and the Repository. None of
// the sibling domains import runs, so there is no import cycle; main.go wires
// thin adapters from the concrete services to these seams.
package runs

import (
	"fmt"
	"time"

	"data-backup-manager/internal/failures"
)

// RunStatus is the lifecycle state of a run. The legal transitions and
// invariants live in lifecycle.go (the Level-2 workflow model).
type RunStatus string

const (
	RunPending       RunStatus = "pending"
	RunCapturing     RunStatus = "capturing"
	RunSnapshotting  RunStatus = "snapshotting"
	RunCompleted     RunStatus = "completed"
	RunPartialFailed RunStatus = "partial_failed"
	RunFailed        RunStatus = "failed"
)

// TriggerSource records who started a run.
type TriggerSource string

const (
	TriggerScheduler TriggerSource = "scheduler"
	TriggerManual    TriggerSource = "manual"
)

// OutcomeStatus is the per (target × destination) result within a run.
type OutcomeStatus string

const (
	OutcomeSucceeded OutcomeStatus = "succeeded"
	OutcomeFailed    OutcomeStatus = "failed"
	OutcomeBlocked   OutcomeStatus = "blocked" // storage-cap block; no bytes written
)

// TargetOutcome is one target×destination result inside a run.
type TargetOutcome struct {
	TargetID        string
	DestinationID   string
	Status          OutcomeStatus
	SnapshotID      string
	Bytes           int64
	Error           string
	FailureCode     failures.Code
	FailureCategory failures.Category
	Warning         string
	StartedAt       time.Time
	FinishedAt      time.Time
}

// Run is the internal domain shape for one plan execution.
type Run struct {
	ID         string
	PlanID     string
	Trigger    TriggerSource
	Status     RunStatus
	StartedAt  time.Time
	FinishedAt time.Time
	// Error carries a run-level failure reason — set when a run fails before
	// any per-target work (e.g. plan resolution) or when startup reconciliation
	// closes a run that was in-flight across a restart. Per-target failures live
	// on each TargetOutcome instead.
	Error           string
	FailureCode     failures.Code
	FailureCategory failures.Category
	NextAction      string
	Preflight       []failures.Cause
	// UpdatedAt is the heartbeat: the last time the run's status or an outcome
	// was persisted. It makes a long-running or wedged run observable.
	UpdatedAt time.Time
	// PhysicalBytes is the deduped+compressed repo growth attributable to this
	// run, summed across the destinations it wrote (a repo-size delta measured
	// around the run). Compared against the logical bytes on Outcomes it yields a
	// dedup ratio. Approximate when runs to the same repo overlap; clamped >= 0.
	PhysicalBytes int64
	Outcomes      []TargetOutcome
}

// TargetStatus is the last-success / last-run rollup for one target, derived
// from run history. Powers the catalog and the health overdue check.
type TargetStatus struct {
	TargetID      string
	LastSuccessAt time.Time
	LastRunStatus RunStatus
	LastRunAt     time.Time
	// Freshness (cadence) signals computed by the service against the configured
	// overdue threshold. Overdue is the single source of truth shared by the
	// CLI, /health, and the UI so the rule never drifts between surfaces.
	Overdue               bool
	LastSuccessAgeSeconds int64
	// NextScheduledAt is when this target's soonest scheduled backup next fires
	// (the earliest next-fire across the scheduled plans it belongs to). Zero
	// when the target is manual-only, on a disabled plan, or its schedule has
	// not fired since startup — surfaces decide how to render "not scheduled".
	NextScheduledAt time.Time
}

// ErrRunNotFound is the typed sentinel for an unknown run id.
type ErrRunNotFound struct{ ID string }

func (e ErrRunNotFound) Error() string { return fmt.Sprintf("run %q not found", e.ID) }

// ErrInvalidRun is the typed validation sentinel.
type ErrInvalidRun struct {
	Field  string
	Reason string
}

func (e ErrInvalidRun) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrRunAlreadyActive is returned instead of starting a second execution for
// the same plan. Callers can inspect the existing run and retry after it is
// terminal; no duplicate repository work is started.
type ErrRunAlreadyActive struct{ PlanID, RunID string }

func (e ErrRunAlreadyActive) Error() string {
	return fmt.Sprintf("plan %q already has active run %q", e.PlanID, e.RunID)
}
