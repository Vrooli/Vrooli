package execution

import "context"

// Repository is the persistence seam for the execution store. Production wires
// the SQLite implementation (sqlite.go) over the ~/.vrooli home store; service
// unit tests substitute a fake or a real sqlite repo. The execution rows are
// queried across one another (handoffs and velocity by plan/execution), so the
// surface is per-record rather than whole-aggregate.
type Repository interface {
	// SaveExecution upserts an execution keyed by id.
	SaveExecution(ctx context.Context, e Execution) error
	// GetExecution returns the execution matching id; ok=false when absent.
	GetExecution(ctx context.Context, id string) (Execution, bool, error)
	// LatestExecutionForPlan returns the newest execution for a plan; ok=false
	// when the plan has never been started.
	LatestExecutionForPlan(ctx context.Context, planID string) (Execution, bool, error)

	// SaveHandoff upserts the canonical handoff for an execution.
	SaveHandoff(ctx context.Context, h Handoff) error
	// GetHandoff returns the most recent handoff for an execution; ok=false when
	// none has been assembled yet.
	GetHandoff(ctx context.Context, executionID string) (Handoff, bool, error)

	// SaveVelocity inserts a velocity point.
	SaveVelocity(ctx context.Context, v VelocityPoint) error
	// ListVelocity returns a plan's velocity series oldest-first.
	ListVelocity(ctx context.Context, planID string) ([]VelocityPoint, error)

	// WithTx runs fn against a repository bound to a single transaction so a
	// multi-write operation (Complete: handoff + velocity + execution state)
	// commits atomically or rolls back as a unit.
	WithTx(ctx context.Context, fn func(Repository) error) error
}

// activeExecutionLister is an additive production capability used to enforce
// the single-active-execution invariant without widening every test double.
type activeExecutionLister interface {
	ListExecutionsForPlan(ctx context.Context, planID string) ([]Execution, error)
}
