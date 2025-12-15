package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *WorkflowService) ListWorkflowVersionsAPI(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowVersionList, error) {
	index, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if index.ProjectID == nil {
		return &basapi.WorkflowVersionList{Versions: []*basapi.WorkflowVersion{}}, nil
	}
	project, err := s.repo.GetProject(ctx, *index.ProjectID)
	if err != nil {
		return nil, err
	}
	versions, err := listWorkflowVersions(ctx, project, workflowID)
	if err != nil {
		return nil, err
	}
	return &basapi.WorkflowVersionList{Versions: versions}, nil
}

func (s *WorkflowService) GetWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32) (*basapi.WorkflowVersion, error) {
	index, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if index.ProjectID == nil {
		return nil, ErrWorkflowRestoreProjectMismatch
	}
	project, err := s.repo.GetProject(ctx, *index.ProjectID)
	if err != nil {
		return nil, err
	}
	return getWorkflowVersion(ctx, project, workflowID, version)
}

func (s *WorkflowService) RestoreWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32, changeDescription string) (*basapi.RestoreWorkflowVersionResponse, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid version")
	}
	index, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if index.ProjectID == nil {
		return nil, ErrWorkflowRestoreProjectMismatch
	}
	project, err := s.repo.GetProject(ctx, *index.ProjectID)
	if err != nil {
		return nil, err
	}

	ver, err := getWorkflowVersion(ctx, project, workflowID, version)
	if err != nil {
		return nil, err
	}

	// Sync workflow index from disk to avoid restoring into a stale file path.
	_ = s.syncProjectWorkflows(ctx, *index.ProjectID)
	index, err = s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	current, err := s.hydrateWorkflowSummary(ctx, index)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, database.ErrNotFound
	}

	desc := strings.TrimSpace(changeDescription)
	if desc == "" {
		desc = fmt.Sprintf("Restored version %d", version)
	}

	updated := proto.Clone(current).(*basapi.WorkflowSummary)
	updated.FlowDefinition = ver.FlowDefinition
	updated.Version = current.Version + 1
	updated.UpdatedAt = timestamppb.New(time.Now().UTC())
	updated.LastChangeSource = basbase.ChangeSource_CHANGE_SOURCE_MANUAL
	updated.LastChangeDescription = desc

	absPath, relPath, err := WriteWorkflowSummaryFile(project, updated, index.FilePath)
	if err != nil {
		return nil, err
	}
	_ = absPath

	index.Version = int(updated.Version)
	index.FilePath = relPath
	if err := s.repo.UpdateWorkflow(ctx, index); err != nil {
		return nil, err
	}
	s.cacheWorkflowPath(workflowID, absPath, relPath)

	// Persist snapshot for the newly-created restored version.
	if err := persistWorkflowVersionSnapshot(project, updated); err != nil {
		if s.log != nil {
			s.log.WithError(err).WithField("workflow_id", workflowID).Warn("Failed to persist workflow version snapshot after restore")
		}
	}

	return &basapi.RestoreWorkflowVersionResponse{
		Workflow:        updated,
		RestoredVersion: ver,
	}, nil
}
