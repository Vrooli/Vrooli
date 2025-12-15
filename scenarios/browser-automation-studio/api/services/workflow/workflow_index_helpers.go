package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func (s *WorkflowService) listAllProjectWorkflows(ctx context.Context, projectID uuid.UUID) ([]*database.WorkflowIndex, error) {
	var all []*database.WorkflowIndex
	offset := 0
	for {
		batch, err := s.repo.ListWorkflowsByProject(ctx, projectID, projectWorkflowPageSize, offset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < projectWorkflowPageSize {
			break
		}
		offset += len(batch)
	}
	return all, nil
}

