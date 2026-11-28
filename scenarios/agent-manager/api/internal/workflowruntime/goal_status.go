package workflowruntime

import (
	"fmt"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// MapGoalStatus maps the harness corroboration into a durable workflow
// terminal. Active and paused are represented by abstained: the workflow has
// not proven completion and must remain reviewable, while the frontier itself
// is still decided by Plan Manager state.
func MapGoalStatus(status runner.GoalStatus) (domain.WorkflowExecutionStatus, error) {
	switch status {
	case runner.GoalStatusComplete:
		return domain.WorkflowExecutionSucceeded, nil
	case runner.GoalStatusBlocked:
		return domain.WorkflowExecutionBlocked, nil
	case runner.GoalStatusUsageLimited, runner.GoalStatusBudgetLimited:
		return domain.WorkflowExecutionBudgetExhausted, nil
	case runner.GoalStatusActive, runner.GoalStatusPaused:
		return domain.WorkflowExecutionAbstained, nil
	default:
		return "", fmt.Errorf("unsupported harness goal status %q", status)
	}
}
