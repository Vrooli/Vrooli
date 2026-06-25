package execution

import "context"

// Repository is the persistence seam for the execution store. Production wires
// the SQLite implementation (sqlite.go) over the ~/.vrooli home store; service
// unit tests substitute a fake or a real sqlite repo. The execution rows are
// queried across one another (findings by execution, velocity by plan), so the
// surface is per-record rather than whole-aggregate.
type Repository interface {
	// SaveExecution upserts an execution keyed by id.
	SaveExecution(ctx context.Context, e Execution) error
	// GetExecution returns the execution matching id; ok=false when absent.
	GetExecution(ctx context.Context, id string) (Execution, bool, error)

	// SaveDecision inserts an in-flow decision.
	SaveDecision(ctx context.Context, d Decision) error
	// ListDecisions returns an execution's decisions oldest-first.
	ListDecisions(ctx context.Context, executionID string) ([]Decision, error)

	// SaveFinding inserts a candidate finding (or updates triage on conflict).
	SaveFinding(ctx context.Context, f Finding) error
	// GetFinding returns the finding matching id; ok=false when absent.
	GetFinding(ctx context.Context, id string) (Finding, bool, error)
	// ListFindings returns findings, optionally scoped to an execution and/or a
	// triage state. An empty executionID lists across executions; an empty triage
	// matches every state.
	ListFindings(ctx context.Context, executionID string, triage FindingTriage) ([]Finding, error)

	// SaveHandoff upserts the canonical handoff for an execution.
	SaveHandoff(ctx context.Context, h Handoff) error
	// GetHandoff returns the most recent handoff for an execution; ok=false when
	// none has been assembled yet.
	GetHandoff(ctx context.Context, executionID string) (Handoff, bool, error)

	// SaveVelocity inserts a velocity point.
	SaveVelocity(ctx context.Context, v VelocityPoint) error
	// ListVelocity returns a plan's velocity series oldest-first.
	ListVelocity(ctx context.Context, planID string) ([]VelocityPoint, error)
}
