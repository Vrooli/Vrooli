package runs

import "context"

// Repository persists runs and their per-target outcomes, and answers the
// last-success / last-run rollup the catalog and health views need.
//
// seam: Repository persists run history. Production wires SqliteRepository
// (sqlite.go); tests wire mocks.FakeRepository.
type Repository interface {
	// CreateRun persists a new run (status pending) and returns it with ID and
	// StartedAt populated.
	CreateRun(ctx context.Context, r Run) (Run, error)

	// SaveRun persists the run's final status, FinishedAt, and the full set of
	// per-target outcomes (replacing any existing outcomes for the run).
	SaveRun(ctx context.Context, r Run) (Run, error)

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
