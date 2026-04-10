package toolexecution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// mockCatalogService implements workflow.CatalogService for testing.
type mockCatalogService struct {
	listWorkflowsFn   func(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error)
	createWorkflowFn  func(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error)
	updateWorkflowFn  func(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error)
	createProjectFn   func(ctx context.Context, project *database.ProjectIndex, description string) error
	listProjectsFn    func(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error)
	getWorkflowFn     func(ctx context.Context, id uuid.UUID) (*basapi.WorkflowSummary, error)
	checkHealthFn     func() string
	checkAutoHealthFn func(ctx context.Context) (bool, error)
}

// Verify mockCatalogService implements CatalogService interface
var _ workflow.CatalogService = (*mockCatalogService)(nil)

func (m *mockCatalogService) CheckHealth() string {
	if m.checkHealthFn != nil {
		return m.checkHealthFn()
	}
	return "ok"
}

func (m *mockCatalogService) CheckAutomationHealth(ctx context.Context) (bool, error) {
	if m.checkAutoHealthFn != nil {
		return m.checkAutoHealthFn(ctx)
	}
	return true, nil
}

func (m *mockCatalogService) CreateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	if m.createProjectFn != nil {
		return m.createProjectFn(ctx, project, description)
	}
	return nil
}

func (m *mockCatalogService) GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) GetProjectByName(ctx context.Context, name string) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) UpdateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	return errors.New("not implemented")
}

func (m *mockCatalogService) DeleteProject(ctx context.Context, id uuid.UUID, deleteFiles bool) error {
	return errors.New("not implemented")
}

func (m *mockCatalogService) ListProjects(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error) {
	if m.listProjectsFn != nil {
		return m.listProjectsFn(ctx, limit, offset)
	}
	return []*database.ProjectIndex{}, nil
}

func (m *mockCatalogService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (*database.ProjectStats, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockCatalogService) EnsureSeedProject(ctx context.Context) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) HydrateProject(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) CreateWorkflow(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	if m.createWorkflowFn != nil {
		return m.createWorkflowFn(ctx, req)
	}
	return &basapi.CreateWorkflowResponse{
		Workflow: &basapi.WorkflowSummary{Id: uuid.New().String(), Name: req.Name},
	}, nil
}

func (m *mockCatalogService) GetWorkflowAPI(ctx context.Context, req *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) UpdateWorkflow(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error) {
	if m.updateWorkflowFn != nil {
		return m.updateWorkflowFn(ctx, req)
	}
	return &basapi.UpdateWorkflowResponse{
		Workflow: &basapi.WorkflowSummary{Id: req.GetWorkflowId(), Version: 2},
	}, nil
}

func (m *mockCatalogService) DeleteWorkflow(ctx context.Context, req *basapi.DeleteWorkflowRequest) (*basapi.DeleteWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) ListWorkflows(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
	if m.listWorkflowsFn != nil {
		return m.listWorkflowsFn(ctx, req)
	}
	return &basapi.ListWorkflowsResponse{Workflows: []*basapi.WorkflowSummary{}, Total: 0}, nil
}

func (m *mockCatalogService) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error) {
	if m.getWorkflowFn != nil {
		return m.getWorkflowFn(ctx, workflowID)
	}
	return nil, errors.New("not found")
}

func (m *mockCatalogService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string, projectRoot string) (*basapi.WorkflowSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) ListWorkflowVersionsAPI(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowVersionList, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) GetWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32) (*basapi.WorkflowVersion, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) RestoreWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32, changeDescription string) (*basapi.RestoreWorkflowVersionResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalogService) SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockCatalogService) ModifyWorkflowAPI(ctx context.Context, workflowID uuid.UUID, prompt string, current *basworkflows.WorkflowDefinitionV2) (*basapi.UpdateWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}

// mockExecutionService implements workflow.ExecutionService for testing.
type mockExecutionService struct {
	executeWorkflowAPIFn    func(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error)
	getExecutionFn          func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	getExecutionTimelineFn  func(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionTimeline, error)
	stopExecutionFn         func(ctx context.Context, executionID uuid.UUID) error
	listExecutionsFn        func(ctx context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error)
}

// Verify mockExecutionService implements ExecutionService interface
var _ workflow.ExecutionService = (*mockExecutionService)(nil)

func (m *mockExecutionService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) ExecuteWorkflowAPI(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
	if m.executeWorkflowAPIFn != nil {
		return m.executeWorkflowAPIFn(ctx, req)
	}
	return &basapi.ExecuteWorkflowResponse{
		ExecutionId: uuid.New().String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_PENDING,
	}, nil
}

func (m *mockExecutionService) ExecuteWorkflowAPIWithOptions(ctx context.Context, req *basapi.ExecuteWorkflowRequest, opts *workflow.ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error) {
	return m.ExecuteWorkflowAPI(ctx, req)
}

func (m *mockExecutionService) ExecuteAdhocWorkflowAPI(ctx context.Context, req *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *workflow.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	if m.stopExecutionFn != nil {
		return m.stopExecutionFn(ctx, executionID)
	}
	return nil
}

func (m *mockExecutionService) ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	if m.listExecutionsFn != nil {
		return m.listExecutionsFn(ctx, workflowID, projectID, limit, offset)
	}
	return []*database.ExecutionIndex{}, nil
}

func (m *mockExecutionService) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if m.getExecutionFn != nil {
		return m.getExecutionFn(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockExecutionService) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	return errors.New("not implemented")
}

func (m *mockExecutionService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) GetExecutionVideoArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionVideoArtifact, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) GetExecutionTraceArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) GetExecutionHarArtifacts(ctx context.Context, executionID uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) HydrateExecutionProto(ctx context.Context, execIndex *database.ExecutionIndex) (*basexecution.Execution, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionTimeline, error) {
	if m.getExecutionTimelineFn != nil {
		return m.getExecutionTimelineFn(ctx, executionID)
	}
	return &workflow.ExecutionTimeline{Frames: []workflow.TimelineFrame{}}, nil
}

func (m *mockExecutionService) GetExecutionTimelineProto(ctx context.Context, executionID uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	return nil, errors.New("not implemented")
}

func (m *mockExecutionService) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	return errors.New("not implemented")
}

// --- Tests ---

func TestExecute_UnknownTool(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "unknown_tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for unknown tool")
	}
	if result.Code != CodeUnknownTool {
		t.Errorf("expected code %q, got %q", CodeUnknownTool, result.Code)
	}
}

func TestExecuteWorkflow_Success(t *testing.T) {
	workflowID := uuid.New()
	executionID := uuid.New()

	mock := &mockExecutionService{
		executeWorkflowAPIFn: func(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
			if req.WorkflowId != workflowID.String() {
				t.Errorf("expected workflow_id %s, got %s", workflowID.String(), req.WorkflowId)
			}
			return &basapi.ExecuteWorkflowResponse{
				ExecutionId: executionID.String(),
				Status:      basbase.ExecutionStatus_EXECUTION_STATUS_PENDING,
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: mock,
	})

	result, err := executor.Execute(context.Background(), "execute_workflow", map[string]interface{}{
		"workflow_id": workflowID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !result.IsAsync {
		t.Error("execute_workflow should return async result")
	}
	if result.RunID != executionID.String() {
		t.Errorf("expected execution_id %s, got %s", executionID.String(), result.RunID)
	}
}

func TestExecuteWorkflow_MissingWorkflowID(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "execute_workflow", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing workflow_id")
	}
	if result.Code != CodeInvalidArgs {
		t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
	}
}

func TestExecuteWorkflow_InvalidWorkflowID(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "execute_workflow", map[string]interface{}{
		"workflow_id": "not-a-valid-uuid",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for invalid workflow_id")
	}
	if result.Code != CodeInvalidArgs {
		t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
	}
}

func TestGetExecution_Success(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()
	startedAt := time.Now()

	mock := &mockExecutionService{
		getExecutionFn: func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
			return &database.ExecutionIndex{
				ID:         executionID,
				WorkflowID: workflowID,
				Status:     database.ExecutionStatusRunning,
				StartedAt:  startedAt,
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: mock,
	})

	result, err := executor.Execute(context.Background(), "get_execution", map[string]interface{}{
		"execution_id": executionID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	if data["status"] != database.ExecutionStatusRunning {
		t.Errorf("expected status 'running', got %v", data["status"])
	}
}

func TestGetExecution_NotFound(t *testing.T) {
	mock := &mockExecutionService{
		getExecutionFn: func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
			return nil, errors.New("not found")
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: mock,
	})

	result, err := executor.Execute(context.Background(), "get_execution", map[string]interface{}{
		"execution_id": uuid.New().String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for not found")
	}
	if result.Code != CodeNotFound {
		t.Errorf("expected code %q, got %q", CodeNotFound, result.Code)
	}
}

func TestStopExecution_Success(t *testing.T) {
	executionID := uuid.New()
	stopCalled := false

	mock := &mockExecutionService{
		stopExecutionFn: func(ctx context.Context, id uuid.UUID) error {
			stopCalled = true
			if id != executionID {
				t.Errorf("expected execution_id %s, got %s", executionID.String(), id.String())
			}
			return nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: mock,
	})

	result, err := executor.Execute(context.Background(), "stop_execution", map[string]interface{}{
		"execution_id": executionID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !stopCalled {
		t.Error("stopExecution was not called")
	}
}

func TestListWorkflows_Success(t *testing.T) {
	workflowID := uuid.New()

	mock := &mockCatalogService{
		listWorkflowsFn: func(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
			return &basapi.ListWorkflowsResponse{
				Workflows: []*basapi.WorkflowSummary{
					{Id: workflowID.String(), Name: "Test Workflow"},
				},
				Total: 1,
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   mock,
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "list_workflows", map[string]interface{}{
		"limit": 50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	workflows, ok := data["workflows"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected workflows field to be a slice")
	}
	if len(workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(workflows))
	}
}

func TestListExecutions_Success(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	mock := &mockExecutionService{
		listExecutionsFn: func(ctx context.Context, wfID *uuid.UUID, projID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
			return []*database.ExecutionIndex{
				{ID: executionID, WorkflowID: workflowID, Status: database.ExecutionStatusCompleted},
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: mock,
	})

	result, err := executor.Execute(context.Background(), "list_executions", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

func TestCreateWorkflow_Success(t *testing.T) {
	workflowID := uuid.New()

	mock := &mockCatalogService{
		createWorkflowFn: func(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
			if req.Name != "Test Workflow" {
				t.Errorf("expected name 'Test Workflow', got %s", req.Name)
			}
			return &basapi.CreateWorkflowResponse{
				Workflow: &basapi.WorkflowSummary{Id: workflowID.String(), Name: req.Name},
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   mock,
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "create_workflow", map[string]interface{}{
		"name": "Test Workflow",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	if data["workflow_id"] != workflowID.String() {
		t.Errorf("expected workflow_id %s, got %v", workflowID.String(), data["workflow_id"])
	}
}

func TestCreateWorkflow_MissingName(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "create_workflow", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing name")
	}
	if result.Code != CodeInvalidArgs {
		t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
	}
}

func TestCreateProject_Success(t *testing.T) {
	createCalled := false

	mock := &mockCatalogService{
		createProjectFn: func(ctx context.Context, project *database.ProjectIndex, description string) error {
			createCalled = true
			if project.Name != "Test Project" {
				t.Errorf("expected name 'Test Project', got %s", project.Name)
			}
			return nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   mock,
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "create_project", map[string]interface{}{
		"name":        "Test Project",
		"description": "A test project",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !createCalled {
		t.Error("createProject was not called")
	}
}

func TestListProjects_Success(t *testing.T) {
	projectID := uuid.New()

	mock := &mockCatalogService{
		listProjectsFn: func(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error) {
			return []*database.ProjectIndex{
				{ID: projectID, Name: "Test Project"},
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   mock,
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "list_projects", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	projects, ok := data["projects"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected projects field to be a slice")
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
}

func TestGetExecutionTimeline_Success(t *testing.T) {
	executionID := uuid.New()

	mock := &mockExecutionService{
		getExecutionTimelineFn: func(ctx context.Context, id uuid.UUID) (*workflow.ExecutionTimeline, error) {
			return &workflow.ExecutionTimeline{
				ExecutionID: executionID,
				Frames: []workflow.TimelineFrame{
					{StepIndex: 0, StepType: "navigate", Status: "completed", Success: true, DurationMs: 100},
					{StepIndex: 1, StepType: "click", Status: "completed", Success: true, DurationMs: 50},
				},
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: mock,
	})

	result, err := executor.Execute(context.Background(), "get_execution_timeline", map[string]interface{}{
		"execution_id": executionID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	if data["total"] != 2 {
		t.Errorf("expected total 2, got %v", data["total"])
	}
}

func TestValidateWorkflow_Success(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "validate_workflow", map[string]interface{}{
		"definition": map[string]interface{}{"steps": []interface{}{}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

func TestValidateWorkflow_MissingDefinition(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	result, err := executor.Execute(context.Background(), "validate_workflow", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing definition")
	}
	if result.Code != CodeInvalidArgs {
		t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
	}
}

func TestInvalidExecutionID(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	tools := []string{"get_execution", "get_execution_timeline", "stop_execution"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tool, map[string]interface{}{
				"execution_id": "not-a-valid-uuid",
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Success {
				t.Error("expected failure for invalid execution_id")
			}
			if result.Code != CodeInvalidArgs {
				t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
			}
		})
	}
}

// Test recording tools return appropriate error (not implemented)
func TestRecordingTools_NotImplemented(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	tools := []struct {
		name string
		args map[string]interface{}
	}{
		{"create_recording_session", map[string]interface{}{"url": "https://example.com"}},
		{"get_recorded_actions", map[string]interface{}{"session_id": uuid.New().String()}},
		{"generate_workflow_from_recording", map[string]interface{}{"session_id": uuid.New().String(), "workflow_name": "Test"}},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tc.name, tc.args)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Recording tools should return an error indicating they require configuration
			if result.Success {
				t.Error("expected failure - recording service not configured")
			}
		})
	}
}

// Test AI tools return appropriate error (not implemented)
func TestAITools_NotImplemented(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mockCatalogService{},
		ExecutionService: &mockExecutionService{},
	})

	tools := []struct {
		name string
		args map[string]interface{}
	}{
		{"ai_analyze_elements", map[string]interface{}{"url": "https://example.com", "query": "find button"}},
		{"ai_navigate", map[string]interface{}{"url": "https://example.com", "goal": "login"}},
		{"get_dom_tree", map[string]interface{}{"url": "https://example.com"}},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tc.name, tc.args)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// AI tools should return an error indicating they require configuration
			if result.Success {
				t.Error("expected failure - AI service not configured")
			}
		})
	}
}
