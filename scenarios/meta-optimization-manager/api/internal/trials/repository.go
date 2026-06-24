package trials

import "context"

// RunFilter narrows a runs query. A zero value matches everything.
type RunFilter struct {
	TaskID string
	Suite  string
}

// Repository is the owned-state seam for the trials history time-series and the
// per-Guide-task gate registry. Production wires the SQLite implementation; tests
// use a fake. A nil Repository disables persistence (RunTrials still dispatches
// but nothing is recorded and history/gate-coverage are empty).
type Repository interface {
	// RecordRun appends a run to the history and bumps the run's Guide-task gate
	// count (so gate coverage reflects it).
	RecordRun(ctx context.Context, run TrialRun) error
	// GetRun returns one run by id; ok=false when absent.
	GetRun(ctx context.Context, id string) (TrialRun, bool, error)
	// Runs returns runs matching the filter. limit<=0 returns all; desc orders
	// newest-first (for recent-runs), else oldest-first (for trend aggregation).
	Runs(ctx context.Context, filter RunFilter, limit int, desc bool) ([]TrialRun, error)
	// GatedGuideTaskCount returns the number of distinct Guide tasks with at least
	// one recorded run (gate_count > 0).
	GatedGuideTaskCount(ctx context.Context) (int, error)
}
