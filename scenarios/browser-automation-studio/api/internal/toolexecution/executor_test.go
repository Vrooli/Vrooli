package toolexecution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/testutil/mocks"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

// --- Tests ---

func TestExecute_UnknownTool(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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

	mock := &mocks.WorkflowExecutionService{
		ExecuteWorkflowAPIFunc: func(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
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
		CatalogService:   &mocks.WorkflowCatalogService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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

	mock := &mocks.WorkflowExecutionService{
		GetExecutionFunc: func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
			return &database.ExecutionIndex{
				ID:         executionID,
				WorkflowID: workflowID,
				Status:     database.ExecutionStatusRunning,
				StartedAt:  startedAt,
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mocks.WorkflowCatalogService{},
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
	mock := &mocks.WorkflowExecutionService{
		GetExecutionFunc: func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
			return nil, errors.New("not found")
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mocks.WorkflowCatalogService{},
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

	mock := &mocks.WorkflowExecutionService{
		StopExecutionFunc: func(ctx context.Context, id uuid.UUID) error {
			stopCalled = true
			if id != executionID {
				t.Errorf("expected execution_id %s, got %s", executionID.String(), id.String())
			}
			return nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mocks.WorkflowCatalogService{},
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

	mock := &mocks.WorkflowCatalogService{
		ListWorkflowsFunc: func(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
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
		ExecutionService: &mocks.WorkflowExecutionService{},
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

	mock := &mocks.WorkflowExecutionService{
		ListExecutionsFunc: func(ctx context.Context, wfID *uuid.UUID, projID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
			return []*database.ExecutionIndex{
				{ID: executionID, WorkflowID: workflowID, Status: database.ExecutionStatusCompleted},
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   &mocks.WorkflowCatalogService{},
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

	mock := &mocks.WorkflowCatalogService{
		CreateWorkflowFunc: func(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
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
		ExecutionService: &mocks.WorkflowExecutionService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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

	mock := &mocks.WorkflowCatalogService{
		CreateProjectFunc: func(ctx context.Context, project *database.ProjectIndex, description string) error {
			createCalled = true
			if project.Name != "Test Project" {
				t.Errorf("expected name 'Test Project', got %s", project.Name)
			}
			return nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   mock,
		ExecutionService: &mocks.WorkflowExecutionService{},
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

	mock := &mocks.WorkflowCatalogService{
		ListProjectsFunc: func(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error) {
			return []*database.ProjectIndex{
				{ID: projectID, Name: "Test Project"},
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{
		CatalogService:   mock,
		ExecutionService: &mocks.WorkflowExecutionService{},
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

	mock := &mocks.WorkflowExecutionService{
		GetExecutionTimelineFunc: func(ctx context.Context, id uuid.UUID) (*workflow.ExecutionTimeline, error) {
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
		CatalogService:   &mocks.WorkflowCatalogService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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
		CatalogService:   &mocks.WorkflowCatalogService{},
		ExecutionService: &mocks.WorkflowExecutionService{},
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
