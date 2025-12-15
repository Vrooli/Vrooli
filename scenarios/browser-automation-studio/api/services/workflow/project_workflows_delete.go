package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func (s *WorkflowService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	for _, workflowID := range workflowIDs {
		if workflowID == uuid.Nil {
			continue
		}
		index, err := s.repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			if err == database.ErrNotFound {
				continue
			}
			return err
		}
		if index.ProjectID == nil || *index.ProjectID != projectID {
			return fmt.Errorf("%w: workflow %s does not belong to project %s", database.ErrNotFound, workflowID, projectID)
		}

		if index.FilePath != "" {
			abs := filepath.Join(projectWorkflowsDir(project), filepath.FromSlash(index.FilePath))
			_ = os.Remove(abs)
		}
		_ = os.RemoveAll(workflowVersionsDir(project.FolderPath, workflowID))

		if err := s.repo.DeleteWorkflow(ctx, workflowID); err != nil {
			return err
		}
		s.removeWorkflowPath(workflowID)
	}

	return nil
}

