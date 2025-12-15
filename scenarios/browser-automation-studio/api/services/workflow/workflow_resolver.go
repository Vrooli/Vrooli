package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"google.golang.org/protobuf/proto"
)

// GetWorkflow implements automation/executor.WorkflowResolver.
func (s *WorkflowService) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error) {
	index, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if index.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *index.ProjectID); err != nil {
			return nil, err
		}
		index, err = s.repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			return nil, err
		}
	}
	return s.hydrateWorkflowSummary(ctx, index)
}

// GetWorkflowVersion implements automation/executor.WorkflowResolver.
func (s *WorkflowService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error) {
	index, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if index.ProjectID == nil {
		return nil, fmt.Errorf("%w: workflow has no project", ErrWorkflowRestoreProjectMismatch)
	}
	project, err := s.repo.GetProject(ctx, *index.ProjectID)
	if err != nil {
		return nil, err
	}

	ver, err := getWorkflowVersion(ctx, project, workflowID, int32(version))
	if err != nil {
		return nil, err
	}

	// Base the version summary off the current workflow so metadata is retained.
	current, err := s.hydrateWorkflowSummary(ctx, index)
	if err != nil {
		return nil, err
	}
	out := proto.Clone(current).(*basapi.WorkflowSummary)
	out.Version = ver.Version
	out.FlowDefinition = ver.FlowDefinition
	if ver.CreatedAt != nil {
		out.UpdatedAt = ver.CreatedAt
	}
	if strings.TrimSpace(ver.ChangeDescription) != "" {
		out.LastChangeDescription = ver.ChangeDescription
	}
	if strings.TrimSpace(ver.CreatedBy) != "" {
		out.CreatedBy = ver.CreatedBy
	}
	return out, nil
}

// GetWorkflowByProjectPath implements automation/executor.WorkflowResolver.
func (s *WorkflowService) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string) (*basapi.WorkflowSummary, error) {
	callingIndex, err := s.repo.GetWorkflow(ctx, callingWorkflowID)
	if err != nil {
		return nil, err
	}
	if callingIndex.ProjectID == nil {
		return nil, database.ErrNotFound
	}
	project, err := s.repo.GetProject(ctx, *callingIndex.ProjectID)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(workflowPath)
	if trimmed == "" {
		return nil, fmt.Errorf("workflow path is required")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	rel := filepath.Clean(filepath.FromSlash(trimmed))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil, fmt.Errorf("invalid workflow path: %q", workflowPath)
	}

	abs := filepath.Join(projectWorkflowsDir(project), rel)
	base := projectWorkflowsDir(project)
	if relCheck, err := filepath.Rel(base, abs); err != nil || strings.HasPrefix(relCheck, "..") {
		return nil, fmt.Errorf("invalid workflow path: %q", workflowPath)
	}

	snapshot, err := ReadWorkflowSummaryFile(ctx, project, abs)
	if err != nil {
		return nil, err
	}
	if snapshot.Workflow == nil {
		return nil, database.ErrNotFound
	}
	return snapshot.Workflow, nil
}
