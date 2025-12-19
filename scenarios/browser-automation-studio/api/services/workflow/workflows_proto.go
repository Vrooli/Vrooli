package workflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

func (s *WorkflowService) CreateWorkflow(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	projectID, err := uuid.Parse(strings.TrimSpace(req.ProjectId))
	if err != nil {
		return nil, fmt.Errorf("invalid project_id: %w", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}

	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve project: %w", err)
	}

	if existing, err := s.repo.GetWorkflowByName(ctx, name, normalizeFolderPath(req.FolderPath)); err == nil && existing != nil && existing.ProjectID != nil && *existing.ProjectID == projectID {
		return nil, ErrWorkflowNameConflict
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("check workflow name: %w", err)
	}

	now := autocontracts.NowTimestamp()
	workflowID := uuid.New()
	folderPath := normalizeFolderPath(req.FolderPath)
	def := req.FlowDefinition
	aiPrompt := strings.TrimSpace(req.AiPrompt)
	if aiPrompt != "" && (def == nil || (len(def.Nodes) == 0 && len(def.Edges) == 0)) {
		generated, err := s.generateWorkflowDefinitionFromPrompt(ctx, aiPrompt, nil)
		if err != nil {
			return nil, err
		}
		def = generated
	}
	if def == nil {
		def = &basworkflows.WorkflowDefinitionV2{}
	}

	summary := &basapi.WorkflowSummary{
		Id:          workflowID.String(),
		ProjectId:   projectID.String(),
		Name:        name,
		FolderPath:  folderPath,
		Description: "",
		Tags:        nil,
		Version:     1,
		IsTemplate:  false,
		CreatedBy:   "",
		LastChangeSource: func() basbase.ChangeSource {
			if aiPrompt != "" {
				return basbase.ChangeSource_CHANGE_SOURCE_AI_GENERATED
			}
			return basbase.ChangeSource_CHANGE_SOURCE_MANUAL
		}(),
		LastChangeDescription: func() string {
			if aiPrompt != "" {
				return "AI generated workflow"
			}
			return "Initial workflow creation"
		}(),
		CreatedAt:   now,
		UpdatedAt:   now,
		FlowDefinition: def,
	}

	absPath, relPath, err := WriteWorkflowSummaryFile(project, summary, "")
	if err != nil {
		return nil, err
	}
	_ = persistWorkflowVersionSnapshot(project, summary)

	index := &database.WorkflowIndex{
		ID:         workflowID,
		ProjectID:  &projectID,
		Name:       name,
		FolderPath: folderPath,
		FilePath:   relPath,
		Version:    int(summary.Version),
	}
	if err := s.repo.CreateWorkflow(ctx, index); err != nil {
		_ = os.Remove(absPath)
		return nil, err
	}
	s.cacheWorkflowPath(workflowID, absPath, relPath)

	return &basapi.CreateWorkflowResponse{
		Workflow:      summary,
		FlowDefinition: summary.FlowDefinition,
	}, nil
}

func (s *WorkflowService) ListWorkflows(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
	if req == nil {
		req = &basapi.ListWorkflowsRequest{}
	}
	limit := int(req.GetLimit())
	offset := int(req.GetOffset())
	if limit <= 0 {
		limit = 50
	}

	folder := strings.TrimSpace(req.GetFolderPath())
	if folder == "" {
		// Eager sync every project so filesystem edits are reflected.
		if err := s.syncProjectsForWorkflowListing(ctx); err != nil {
			return nil, err
		}
	}

	var workflows []*database.WorkflowIndex
	var err error
	if req.ProjectId != nil && strings.TrimSpace(req.GetProjectId()) != "" {
		projectID, parseErr := uuid.Parse(strings.TrimSpace(req.GetProjectId()))
		if parseErr != nil {
			return nil, fmt.Errorf("invalid project_id: %w", parseErr)
		}
		if err := s.syncProjectWorkflows(ctx, projectID); err != nil {
			return nil, err
		}
		workflows, err = s.repo.ListWorkflowsByProject(ctx, projectID, limit, offset)
	} else {
		workflows, err = s.repo.ListWorkflows(ctx, folder, limit, offset)
	}
	if err != nil {
		return nil, err
	}

	out := make([]*basapi.WorkflowSummary, 0, len(workflows))
	for _, wf := range workflows {
		if wf == nil {
			continue
		}
		summary, err := s.hydrateWorkflowSummary(ctx, wf)
		if err != nil {
			if s.log != nil {
				s.log.WithError(err).WithField("workflow_id", wf.ID).Warn("Failed to hydrate workflow summary from file")
			}
			continue
		}
		out = append(out, summary)
	}

	return &basapi.ListWorkflowsResponse{
		Workflows: out,
		Total:     int32(len(out)),
		HasMore:   len(out) == limit,
	}, nil
}

func (s *WorkflowService) GetWorkflowAPI(ctx context.Context, req *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	workflowID, err := uuid.Parse(strings.TrimSpace(req.WorkflowId))
	if err != nil {
		return nil, fmt.Errorf("invalid workflow_id: %w", err)
	}

	if req.Version != nil && req.GetVersion() > 0 {
		versionWf, err := s.GetWorkflowVersion(ctx, workflowID, int(req.GetVersion()))
		if err != nil {
			return nil, err
		}
		return &basapi.GetWorkflowResponse{Workflow: versionWf}, nil
	}

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

	summary, err := s.hydrateWorkflowSummary(ctx, index)
	if err != nil {
		return nil, err
	}

	return &basapi.GetWorkflowResponse{Workflow: summary}, nil
}

func (s *WorkflowService) UpdateWorkflow(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	workflowID, err := uuid.Parse(strings.TrimSpace(req.GetWorkflowId()))
	if err != nil {
		return nil, fmt.Errorf("invalid workflow_id: %w", err)
	}

	currentIndex, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if currentIndex.ProjectID == nil {
		return nil, fmt.Errorf("workflow %s is not associated with a project", workflowID)
	}
	project, err := s.repo.GetProject(ctx, *currentIndex.ProjectID)
	if err != nil {
		return nil, err
	}

	if err := s.syncProjectWorkflows(ctx, *currentIndex.ProjectID); err != nil {
		return nil, err
	}

	currentSummary, err := s.hydrateWorkflowSummary(ctx, currentIndex)
	if err != nil {
		return nil, err
	}

	expected := req.ExpectedVersion
	if expected != 0 && int32(currentSummary.Version) != expected {
		return nil, fmt.Errorf("%w: expected %d, found %d", ErrWorkflowVersionConflict, expected, currentSummary.Version)
	}

	updated := proto.Clone(currentSummary).(*basapi.WorkflowSummary)
	updated.Name = strings.TrimSpace(req.Name)
	if updated.Name == "" {
		updated.Name = currentSummary.Name
	}
	updated.Description = strings.TrimSpace(req.Description)
	updated.FolderPath = normalizeFolderPath(req.FolderPath)
	updated.Tags = append([]string(nil), req.Tags...)
	if req.FlowDefinition != nil {
		updated.FlowDefinition = req.FlowDefinition
	}
	updated.Version = currentSummary.Version + 1
	updated.UpdatedAt = autocontracts.NowTimestamp()
	updated.LastChangeDescription = strings.TrimSpace(req.ChangeDescription)
	updated.LastChangeSource = req.Source
	if updated.LastChangeSource == basbase.ChangeSource_CHANGE_SOURCE_UNSPECIFIED {
		updated.LastChangeSource = basbase.ChangeSource_CHANGE_SOURCE_MANUAL
	}
	if updated.LastChangeDescription == "" {
		updated.LastChangeDescription = "Updated workflow"
	}

	absPath, relPath, err := WriteWorkflowSummaryFile(project, updated, currentIndex.FilePath)
	if err != nil {
		return nil, err
	}
	_ = persistWorkflowVersionSnapshot(project, updated)
	s.cacheWorkflowPath(workflowID, absPath, relPath)

	currentIndex.Name = updated.Name
	currentIndex.FolderPath = updated.FolderPath
	currentIndex.FilePath = relPath
	currentIndex.Version = int(updated.Version)
	if err := s.repo.UpdateWorkflow(ctx, currentIndex); err != nil {
		return nil, err
	}

	return &basapi.UpdateWorkflowResponse{
		Workflow:      updated,
		FlowDefinition: updated.FlowDefinition,
	}, nil
}

func (s *WorkflowService) DeleteWorkflow(ctx context.Context, req *basapi.DeleteWorkflowRequest) (*basapi.DeleteWorkflowResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	workflowID, err := uuid.Parse(strings.TrimSpace(req.WorkflowId))
	if err != nil {
		return nil, fmt.Errorf("invalid workflow_id: %w", err)
	}
	index, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if index.ProjectID != nil {
		project, err := s.repo.GetProject(ctx, *index.ProjectID)
		if err == nil {
			abs := filepath.Join(projectWorkflowsDir(project), filepath.FromSlash(index.FilePath))
			_ = os.Remove(abs)
		}
	}

	if err := s.repo.DeleteWorkflow(ctx, workflowID); err != nil {
		return nil, err
	}
	s.removeWorkflowPath(workflowID)

	return &basapi.DeleteWorkflowResponse{
		Success:    true,
		WorkflowId: workflowID.String(),
	}, nil
}

// hydrateWorkflowSummary reads the workflow proto file from disk (source of truth).
func (s *WorkflowService) hydrateWorkflowSummary(ctx context.Context, wf *database.WorkflowIndex) (*basapi.WorkflowSummary, error) {
	if wf == nil {
		return nil, errors.New("workflow is nil")
	}
	if wf.ProjectID == nil {
		return nil, errors.New("workflow project_id missing")
	}
	project, err := s.repo.GetProject(ctx, *wf.ProjectID)
	if err != nil {
		return nil, err
	}

	abs := filepath.Join(projectWorkflowsDir(project), filepath.FromSlash(wf.FilePath))
	snapshot, err := ReadWorkflowSummaryFile(ctx, project, abs)
	if err != nil {
		return nil, err
	}
	if snapshot.Workflow == nil {
		return nil, fmt.Errorf("workflow file %s: missing workflow payload", wf.FilePath)
	}
	// Ensure required identifiers are populated even if legacy file omitted them.
	snapshot.Workflow.ProjectId = wf.ProjectID.String()
	if strings.TrimSpace(snapshot.Workflow.Id) == "" {
		snapshot.Workflow.Id = wf.ID.String()
	}
	if snapshot.Workflow.FlowDefinition == nil {
		snapshot.Workflow.FlowDefinition = &basworkflows.WorkflowDefinitionV2{}
	}
	if snapshot.Workflow.CreatedAt == nil {
		snapshot.Workflow.CreatedAt = autocontracts.NowTimestamp()
	}
	if snapshot.Workflow.UpdatedAt == nil {
		snapshot.Workflow.UpdatedAt = snapshot.Workflow.CreatedAt
	}
	if snapshot.NeedsWrite {
		if _, _, err := WriteWorkflowSummaryFile(project, snapshot.Workflow, wf.FilePath); err != nil && s.log != nil {
			s.log.WithError(err).WithField("workflow_id", wf.ID).Warn("Failed to normalize workflow file to proto format")
		}
	}
	return snapshot.Workflow, nil
}

func (s *WorkflowService) syncProjectsForWorkflowListing(ctx context.Context) error {
	projects, err := s.repo.ListProjects(ctx, 1000, 0)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if project == nil || project.ID == uuid.Nil {
			continue
		}
		if !s.shouldSyncProject(project.ID) {
			continue
		}
		if err := s.syncProjectWorkflows(ctx, project.ID); err != nil {
			if s.log != nil {
				s.log.WithError(err).WithField("project_id", project.ID).Warn("Failed to synchronize workflows before listing")
			}
			continue
		}
		s.recordProjectSync(project.ID)
	}
	return nil
}
