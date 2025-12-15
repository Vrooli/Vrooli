package workflow

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	if s == nil {
		return nil, fmt.Errorf("workflow service not configured")
	}
	return s.repo.ListExecutions(ctx, workflowID, limit, offset)
}

func (s *WorkflowService) ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	_ = ctx
	_ = parameters
	return nil, fmt.Errorf("execution %s cannot be resumed: resume support not implemented", executionID)
}

