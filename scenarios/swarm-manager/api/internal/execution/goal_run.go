package execution

import (
	"context"

	"swarm-manager/internal/agentmanager"
)

// GoalRunCreator creates exactly one Agent Manager run for a goal execution.
// The concrete implementation owns the Agent Manager run-creation client; the
// execution service owns the goal message, finish line, and idempotency key.
type GoalRunCreator interface {
	CreateGoalRun(ctx context.Context, req agentmanager.GoalRunRequest) (agentmanager.GoalRunResult, error)
}

// GoalRunReader reads a goal run's typed terminal fields from Agent Manager so
// reconcile can map a verdict to finalization and an interruption to
// `interrupted`.
type GoalRunReader interface {
	GetGoalRunState(ctx context.Context, runID string) (agentmanager.GoalRunState, error)
}
