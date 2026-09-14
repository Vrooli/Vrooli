package execution

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitions"
	"swarm-manager/internal/workflowcontract"
)

// GoalRunCreator creates exactly one Agent Manager run for a goal execution.
// The concrete implementation owns the Agent Manager run-creation client; the
// execution service owns the goal message, finish line, and idempotency key.
type GoalRunCreator interface {
	CreateGoalRun(ctx context.Context, req agentmanager.GoalRunRequest) (agentmanager.GoalRunResult, error)
}

// GoalRunReader reads a goal run's typed terminal fields from Agent Manager so
// reconcile can map a verdict to finalization and an interruption to
// `interrupted`, and reads the run's metered usage so its reservation can be
// settled. A goal run has no workflow, so the workflow receipt cannot supply it.
type GoalRunReader interface {
	GetGoalRunState(ctx context.Context, runID string) (agentmanager.GoalRunState, error)
	GetGoalRunUsage(ctx context.Context, runID string) (*workflowcontract.Usage, bool, error)
}

func isGoalRecord(record Record) bool {
	return record.ExecutionMode == transitions.ExecutionModeGoal
}

// settleableUsage reports whether usage is a complete owner receipt that may
// settle a reservation. Unknown usage is never zero.
func settleableUsage(usage *workflowcontract.Usage) bool {
	return usage != nil && usage.TokensKnown && usage.ChargeMeasured && usage.WallSeconds > 0 &&
		usage.Tokens >= 0 && usage.Turns >= 0 && usage.ChargeMicroUSD >= 0 && usage.Children >= 0 &&
		usage.NodeAttempts >= 0 && usage.Retries >= 0 && usage.Slices >= 0
}

// ownerRunTerminal reports whether Agent Manager says the execution's recorded
// run has stopped. A read error or missing run is not proof of termination.
func (s *Service) ownerRunTerminal(ctx context.Context, record Record) bool {
	if s.goalRunReader == nil || strings.TrimSpace(record.RunID) == "" {
		return false
	}
	_, terminal, err := s.goalRunReader.GetGoalRunUsage(ctx, record.RunID)
	return err == nil && terminal
}

// settledGoalUsage reads a goal run's owner accounting and returns it only when
// it can settle the reservation: the run is terminal and the receipt is
// complete. It returns nil usage, not an error, while the receipt is pending.
func (s *Service) settledGoalUsage(ctx context.Context, record Record) (*workflowcontract.Usage, error) {
	if s.goalRunReader == nil || strings.TrimSpace(record.RunID) == "" {
		return nil, fmt.Errorf("goal run accounting is unavailable for execution %s", record.ExecutionID)
	}
	usage, terminal, err := s.goalRunReader.GetGoalRunUsage(ctx, record.RunID)
	if err != nil {
		return nil, err
	}
	if !terminal || !settleableUsage(usage) {
		return nil, nil
	}
	return usage, nil
}
