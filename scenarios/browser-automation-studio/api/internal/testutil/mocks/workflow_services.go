package mocks

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// WorkflowCatalogService is a function-backed fake for tests that need only a
// small part of workflow.CatalogService. Unconfigured methods return stable
// defaults for commonly used read/write paths and explicit "not implemented"
// errors for paths the test did not arrange.
type WorkflowCatalogService struct {
	CheckHealthFunc           func() string
	CheckAutomationHealthFunc func(ctx context.Context) (bool, error)
	CreateProjectFunc         func(ctx context.Context, project *database.ProjectIndex, description string) error
	ListProjectsFunc          func(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error)
	CreateWorkflowFunc        func(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error)
	UpdateWorkflowFunc        func(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error)
	ListWorkflowsFunc         func(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error)
	GetWorkflowFunc           func(ctx context.Context, id uuid.UUID) (*basapi.WorkflowSummary, error)
}

func (m *WorkflowCatalogService) CheckHealth() string {
	if m.CheckHealthFunc != nil {
		return m.CheckHealthFunc()
	}
	return "ok"
}

func (m *WorkflowCatalogService) CheckAutomationHealth(ctx context.Context) (bool, error) {
	if m.CheckAutomationHealthFunc != nil {
		return m.CheckAutomationHealthFunc(ctx)
	}
	return true, nil
}

func (m *WorkflowCatalogService) CreateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, project, description)
	}
	return nil
}

func (m *WorkflowCatalogService) GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) GetProjectByName(ctx context.Context, name string) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) UpdateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	return errors.New("not implemented")
}

func (m *WorkflowCatalogService) DeleteProject(ctx context.Context, id uuid.UUID, deleteFiles bool) error {
	return errors.New("not implemented")
}

func (m *WorkflowCatalogService) ListProjects(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx, limit, offset)
	}
	return []*database.ProjectIndex{}, nil
}

func (m *WorkflowCatalogService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (*database.ProjectStats, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *WorkflowCatalogService) EnsureSeedProject(ctx context.Context) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) HydrateProject(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) CreateWorkflow(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	if m.CreateWorkflowFunc != nil {
		return m.CreateWorkflowFunc(ctx, req)
	}
	return &basapi.CreateWorkflowResponse{
		Workflow: &basapi.WorkflowSummary{Id: uuid.New().String(), Name: req.Name},
	}, nil
}

func (m *WorkflowCatalogService) GetWorkflowAPI(ctx context.Context, req *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) UpdateWorkflow(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error) {
	if m.UpdateWorkflowFunc != nil {
		return m.UpdateWorkflowFunc(ctx, req)
	}
	return &basapi.UpdateWorkflowResponse{
		Workflow: &basapi.WorkflowSummary{Id: req.GetWorkflowId(), Version: 2},
	}, nil
}

func (m *WorkflowCatalogService) DeleteWorkflow(ctx context.Context, req *basapi.DeleteWorkflowRequest) (*basapi.DeleteWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) ListWorkflows(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc(ctx, req)
	}
	return &basapi.ListWorkflowsResponse{Workflows: []*basapi.WorkflowSummary{}, Total: 0}, nil
}

func (m *WorkflowCatalogService) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error) {
	if m.GetWorkflowFunc != nil {
		return m.GetWorkflowFunc(ctx, workflowID)
	}
	return nil, errors.New("not found")
}

func (m *WorkflowCatalogService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string, projectRoot string) (*basapi.WorkflowSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) ListWorkflowVersionsAPI(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowVersionList, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) GetWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32) (*basapi.WorkflowVersion, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) RestoreWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32, changeDescription string) (*basapi.RestoreWorkflowVersionResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowCatalogService) SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *WorkflowCatalogService) ModifyWorkflowAPI(ctx context.Context, workflowID uuid.UUID, prompt string, current *basworkflows.WorkflowDefinitionV2) (*basapi.UpdateWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}

// WorkflowExecutionService is a function-backed fake for tests that exercise
// workflow.ExecutionService through a narrow subset of methods.
type WorkflowExecutionService struct {
	ExecuteWorkflowAPIFunc   func(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error)
	GetExecutionFunc         func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	GetExecutionTimelineFunc func(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionTimeline, error)
	StopExecutionFunc        func(ctx context.Context, executionID uuid.UUID) error
	ListExecutionsFunc       func(ctx context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error)
}

func (m *WorkflowExecutionService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) ExecuteWorkflowAPI(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
	if m.ExecuteWorkflowAPIFunc != nil {
		return m.ExecuteWorkflowAPIFunc(ctx, req)
	}
	return &basapi.ExecuteWorkflowResponse{
		ExecutionId: uuid.New().String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_PENDING,
	}, nil
}

func (m *WorkflowExecutionService) ExecuteWorkflowAPIWithOptions(ctx context.Context, req *basapi.ExecuteWorkflowRequest, opts *workflow.ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error) {
	return m.ExecuteWorkflowAPI(ctx, req)
}

func (m *WorkflowExecutionService) ExecuteAdhocWorkflowAPI(ctx context.Context, req *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *workflow.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	if m.StopExecutionFunc != nil {
		return m.StopExecutionFunc(ctx, executionID)
	}
	return nil
}

func (m *WorkflowExecutionService) ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	if m.ListExecutionsFunc != nil {
		return m.ListExecutionsFunc(ctx, workflowID, projectID, limit, offset)
	}
	return []*database.ExecutionIndex{}, nil
}

func (m *WorkflowExecutionService) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if m.GetExecutionFunc != nil {
		return m.GetExecutionFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *WorkflowExecutionService) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	return errors.New("not implemented")
}

func (m *WorkflowExecutionService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) GetExecutionVideoArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionVideoArtifact, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) GetExecutionTraceArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) GetExecutionHarArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) HydrateExecutionProto(ctx context.Context, execIndex *database.ExecutionIndex) (*basexecution.Execution, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionTimeline, error) {
	if m.GetExecutionTimelineFunc != nil {
		return m.GetExecutionTimelineFunc(ctx, executionID)
	}
	return &workflow.ExecutionTimeline{Frames: []workflow.TimelineFrame{}}, nil
}

func (m *WorkflowExecutionService) GetExecutionTimelineProto(ctx context.Context, executionID uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) GetExecutionReplayPackage(ctx context.Context, executionID uuid.UUID) (*basevidence.ReplayPackage, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	return nil, errors.New("not implemented")
}

func (m *WorkflowExecutionService) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	return errors.New("not implemented")
}

var (
	_ workflow.CatalogService   = (*WorkflowCatalogService)(nil)
	_ workflow.ExecutionService = (*WorkflowExecutionService)(nil)
)
