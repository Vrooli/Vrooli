package runs

import (
	"context"
	"time"
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

	// UpdateRunStatus persists a (non-terminal) status transition and bumps the
	// run's heartbeat. Used by the executor for pending→capturing→snapshotting.
	UpdateRunStatus(ctx context.Context, runID string, status RunStatus) error

	// SaveOutcome upserts a single per-target outcome as it completes (keyed by
	// run_id+target_id+destination_id) and bumps the run's heartbeat, so partial
	// progress is durable before the run reaches its terminal state.
	SaveOutcome(ctx context.Context, runID string, o TargetOutcome) error

	// FinishRun records the run's terminal status, an optional run-level error
	// reason, FinishedAt, and the physical (deduped) repo-growth bytes attributed
	// to the run. It is the single terminal-write the executor and startup
	// reconciliation share (reconciliation passes 0 — an orphan wrote nothing
	// measurable).
	FinishRun(ctx context.Context, runID string, status RunStatus, errMsg string, finishedAt time.Time, physicalBytes int64) error

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
}
