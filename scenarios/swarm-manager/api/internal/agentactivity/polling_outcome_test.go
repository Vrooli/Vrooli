package agentactivity

import (
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestMapWorkflowStatusPreservesTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name string
		in   domainpb.WorkflowExecutionStatus
		want Status
	}{
		{"abstained", domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED, StatusAbstained},
		{"budget exhausted", domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED, StatusBudgetExhausted},
		{"failed", domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED, StatusFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapWorkflowStatus(tt.in); got != tt.want {
				t.Fatalf("mapWorkflowStatus(%s) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
