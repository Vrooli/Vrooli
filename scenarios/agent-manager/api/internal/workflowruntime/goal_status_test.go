package workflowruntime

import (
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestMapGoalStatusCoversEveryDeclaredStatus(t *testing.T) {
	tests := []struct {
		status runner.GoalStatus
		want   domain.WorkflowExecutionStatus
	}{
		{runner.GoalStatusActive, domain.WorkflowExecutionAbstained},
		{runner.GoalStatusPaused, domain.WorkflowExecutionAbstained},
		{runner.GoalStatusBlocked, domain.WorkflowExecutionBlocked},
		{runner.GoalStatusUsageLimited, domain.WorkflowExecutionBudgetExhausted},
		{runner.GoalStatusBudgetLimited, domain.WorkflowExecutionBudgetExhausted},
		{runner.GoalStatusComplete, domain.WorkflowExecutionSucceeded},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got, err := MapGoalStatus(tt.status)
			if err != nil || got != tt.want {
				t.Fatalf("MapGoalStatus(%q) = %q, %v; want %q", tt.status, got, err, tt.want)
			}
		})
	}
}
