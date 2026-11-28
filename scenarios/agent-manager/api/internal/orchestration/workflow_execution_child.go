// This file reads the persisted run a workflow node attempt owns, proving the
// child run was created for this exact execution and attempt. The lookup is
// bounded to two rows so an ambiguous child is an error, not an arbitrary choice.
package orchestration

import (
	"context"
	"fmt"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

func (o *Orchestrator) persistedWorkflowChild(ctx context.Context, executionID, attemptID uuid.UUID) (*domain.Run, error) {
	if o.runs == nil {
		return nil, nil
	}
	rows, err := o.runs.ListByTask(ctx, uuid.NewSHA1(attemptID, []byte("workflow-node-task")), repository.ListFilter{Limit: 2})
	if err != nil {
		return nil, err
	}
	if len(rows) > 1 {
		return nil, fmt.Errorf("workflow attempt has ambiguous persisted children")
	}
	if len(rows) == 0 {
		return nil, nil
	}
	child, err := o.runs.Get(ctx, rows[0].ID)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, fmt.Errorf("persisted workflow child disappeared")
	}
	if child.CustomEnv[workflowAttemptEnv] != attemptID.String() || child.CustomEnv[workflowExecutionEnv] != executionID.String() {
		return nil, fmt.Errorf("workflow task child provenance mismatch")
	}
	return child, nil
}
