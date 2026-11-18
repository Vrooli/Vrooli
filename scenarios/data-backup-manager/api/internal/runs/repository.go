package runs

import (
	"context"
	"time"

	"data-backup-manager/internal/failures"
)

// Repository persists runs and their per-target outcomes, and answers the
// last-success / last-run rollup the catalog and health views need. The async
// executor drives a run through it incrementally — UpdateRunStatus on each
// lifecycle transition, SaveOutcome as each target lands, FinishRun at the
// terminal state — so a run's live progress is observable and a crashed run is
// reconcilable rather than stranded mid-flight.
//
// seam: Repository persists run history. Production wires SqliteRepository
// (sqlite.go); tests wire mocks.FakeRepository.
type Repository interface {
	// CreateRun persists a new run (status pending) and returns it with ID and
	// StartedAt populated.
	CreateRun(ctx context.Context, r Run) (Run, error)

	// UpdateRunStatus applies a legal non-terminal lifecycle event and bumps the
	// run's heartbeat. Used by the executor for pending→capturing→snapshotting.
	UpdateRunStatus(ctx context.Context, runID string, event RunEvent) error

	// SaveOutcome upserts a single per-target outcome as it completes (keyed by
	// run_id+target_id+destination_id) and bumps the run's heartbeat, so partial
	// progress is durable before the run reaches its terminal state.
	SaveOutcome(ctx context.Context, runID string, o TargetOutcome) error

	// FinishRun applies a terminal event and records the optional run-level error
	// reason, FinishedAt, and the physical (deduped) repo-growth bytes attributed
	// to the run. It is the single terminal-write the executor and startup
	// reconciliation share (reconciliation passes 0 — an orphan wrote nothing
	// measurable).
	FinishRun(ctx context.Context, runID string, event RunEvent, errMsg string, finishedAt time.Time, physicalBytes int64) error

	// ListNonTerminalRuns returns runs left in a non-terminal status
	// (pending/capturing/snapshotting) — the orphans startup reconciliation
	// closes. Outcomes are not attached (reconciliation does not need them).
	ListNonTerminalRuns(ctx context.Context) ([]Run, error)

	// GetRun returns the run with its outcomes, or ErrRunNotFound.
	GetRun(ctx context.Context, id string) (Run, error)

	// ListRuns returns runs newest-first, optionally filtered by plan id.
	// limit <= 0 returns no rows.
	ListRuns(ctx context.Context, planID string, limit int) ([]Run, error)

	// TargetStatuses returns the last-success / last-run rollup per target,
	// optionally filtered to targets owned by a given owner (empty = all).
	// Owner filtering is best-effort — runs does not own targets, so the
	// owner filter is applied by the caller when it has the mapping; the
	// repository keys purely on target id.
	TargetStatuses(ctx context.Context, targetIDs []string) ([]TargetStatus, error)
	// TargetDestinationStatuses returns the latest successful capture for each
	// target × destination pair, optionally filtered by target ids.
	TargetDestinationStatuses(ctx context.Context, targetIDs []string) ([]TargetDestinationStatus, error)
}

// FailureUpdater is an optional additive persistence seam. Keeping it
// separate preserves compatibility for narrow test repositories while the
// catalog gains stable root-cause evidence.
type FailureUpdater interface {
	UpdateRunFailure(ctx context.Context, runID string, code failures.Code, category failures.Category, nextAction string) error
}

type IncidentUpdater interface {
	SavePreflightIncidents(ctx context.Context, runID string, incidents []failures.Cause) error
	IncidentsForRun(ctx context.Context, runID string) ([]failures.Cause, error)
}

// PlanSuccess is the recorded success history of one plan: the newest
// succeeded target outcome overall and per target. It is read-only evidence for
// the stalled-backup monitor.
type PlanSuccess struct {
	At       time.Time
	ByTarget map[string]time.Time
}

// PlanSuccessReader is an optional additive read seam, kept separate so narrow
// test repositories stay compatible.
type PlanSuccessReader interface {
	PlanSuccess(ctx context.Context, planID string) (PlanSuccess, error)
}

// PhysicalHistoryReader is an optional additive read seam. Preflight uses the
// last successful run's measured physical growth as the capacity model's
// incremental estimate; a repository without the seam stays conservative.
type PhysicalHistoryReader interface {
	LastSuccessfulPhysicalBytes(ctx context.Context, planID string) (int64, bool, error)
}

// StatsWindowReader is implemented by a repository that can bound the run-stats
// window in attempts (scheduled and manual runs) instead of in rows, so
// automatic retries do not crowd real attempts out of it.
type StatsWindowReader interface {
	StatsWindow(ctx context.Context, planID string, attempts int) ([]Run, error)
}
