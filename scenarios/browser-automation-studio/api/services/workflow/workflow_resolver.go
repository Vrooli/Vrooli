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
// It resolves a subflow path using either:
// 1. Database-based resolution (if calling workflow has a ProjectID)
// 2. Filesystem-based resolution (if projectRoot is provided)
func (s *WorkflowService) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string, projectRoot string) (*basapi.WorkflowSummary, error) {
	// Validate and normalize the workflow path
	rel, err := validateAndNormalizePath(workflowPath)
	if err != nil {
		return nil, err
	}

	// Strategy 1: Try database-based resolution if calling workflow has a project
	callingIndex, _ := s.repo.GetWorkflow(ctx, callingWorkflowID)
	if callingIndex != nil && callingIndex.ProjectID != nil {
		project, err := s.repo.GetProject(ctx, *callingIndex.ProjectID)
		if err == nil {
			return s.resolveFromProjectDir(ctx, project, rel, workflowPath)
		}
	}

	// Strategy 2: Filesystem-based resolution using projectRoot
	if projectRoot != "" {
		return s.resolveFromFilesystemRoot(ctx, projectRoot, rel, workflowPath)
	}

	return nil, database.ErrNotFound
}

// validateAndNormalizePath validates and normalizes a workflow path.
func validateAndNormalizePath(workflowPath string) (string, error) {
	trimmed := strings.TrimSpace(workflowPath)
	if trimmed == "" {
		return "", fmt.Errorf("workflow path is required")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	rel := filepath.Clean(filepath.FromSlash(trimmed))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid workflow path: %q", workflowPath)
	}
	return rel, nil
}

// resolveFromProjectDir resolves a workflow from the project's workflows directory.
func (s *WorkflowService) resolveFromProjectDir(ctx context.Context, project *database.ProjectIndex, rel string, workflowPath string) (*basapi.WorkflowSummary, error) {
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

// resolveFromFilesystemRoot resolves a workflow from the filesystem using projectRoot.
// For test-genie, projectRoot is the bas/ folder, so "actions/test.json"
// resolves to {projectRoot}/actions/test.json (e.g., bas/actions/test.json).
func (s *WorkflowService) resolveFromFilesystemRoot(ctx context.Context, projectRoot string, rel string, workflowPath string) (*basapi.WorkflowSummary, error) {
	// Security: require absolute path
	if !filepath.IsAbs(projectRoot) {
		return nil, fmt.Errorf("project_root must be absolute path")
	}

	// Use projectRoot directly as base (NOT projectRoot/workflows/)
	base := projectRoot
	abs := filepath.Join(base, rel)

	// Security: validate path stays within base
	if relCheck, err := filepath.Rel(base, abs); err != nil || strings.HasPrefix(relCheck, "..") {
		return nil, fmt.Errorf("invalid workflow path: %q", workflowPath)
	}

	// Read file using synthetic project for compatibility
	syntheticProject := &database.ProjectIndex{FolderPath: projectRoot}
	snapshot, err := ReadWorkflowSummaryFile(ctx, syntheticProject, abs)
	if err != nil {
		return nil, err
	}
	if snapshot.Workflow == nil {
		return nil, database.ErrNotFound
	}
	return snapshot.Workflow, nil
}
