package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/logutil"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
)

// mockWorkflowServiceForWorkflows provides workflow service implementation for workflow handler tests
type mockWorkflowServiceForWorkflows struct {
	createWorkflowFn         func(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error)
	getWorkflowFn            func(ctx context.Context, id uuid.UUID) (*database.Workflow, error)
	deleteWorkflowFn         func(ctx context.Context, id uuid.UUID) error
	listWorkflowsByProjectFn func(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error)
	executeWorkflowFn        func(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
	getExecutionFn           func(ctx context.Context, executionID uuid.UUID) (*database.Execution, error)
	listWorkflowsFn          func(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error)
	updateWorkflowFn         func(ctx context.Context, workflowID uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error)
	modifyWorkflowFn         func(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error)
}

func (m *mockWorkflowServiceForWorkflows) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	if m.createWorkflowFn != nil {
		return m.createWorkflowFn(ctx, projectID, name, folderPath, flowDefinition, aiPrompt)
	}
	workflow := &database.Workflow{
		ID:             uuid.New(),
		Name:           name,
		FolderPath:     folderPath,
		FlowDefinition: database.JSONMap(flowDefinition),
	}
	if projectID != nil {
		workflow.ProjectID = projectID
	}
	return workflow, nil
}

func (m *mockWorkflowServiceForWorkflows) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	if m.getWorkflowFn != nil {
		return m.getWorkflowFn(ctx, id)
	}
	return &database.Workflow{
		ID:         id,
		Name:       "Test Workflow",
		FolderPath: "/workflows",
	}, nil
}

func (m *mockWorkflowServiceForWorkflows) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	if m.deleteWorkflowFn != nil {
		return m.deleteWorkflowFn(ctx, id)
	}
	return nil
}

func (m *mockWorkflowServiceForWorkflows) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	if m.listWorkflowsByProjectFn != nil {
		return m.listWorkflowsByProjectFn(ctx, projectID, limit, offset)
	}
	return []*database.Workflow{
		{ID: uuid.New(), Name: "Workflow 1", FolderPath: "/workflows"},
		{ID: uuid.New(), Name: "Workflow 2", FolderPath: "/workflows"},
	}, nil
}

func (m *mockWorkflowServiceForWorkflows) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	if m.executeWorkflowFn != nil {
		return m.executeWorkflowFn(ctx, workflowID, parameters)
	}
	return &database.Execution{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     "pending",
	}, nil
}

// Stub implementations matching full WorkflowService interface from handler.go
func (m *mockWorkflowServiceForWorkflows) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockWorkflowServiceForWorkflows) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockWorkflowServiceForWorkflows) DeleteProject(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockWorkflowServiceForWorkflows) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, nil
}

func (m *mockWorkflowServiceForWorkflows) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	return map[uuid.UUID]map[string]any{}, nil
}
func (m *mockWorkflowServiceForWorkflows) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *mockWorkflowServiceForWorkflows) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	if m.listWorkflowsFn != nil {
		return m.listWorkflowsFn(ctx, folderPath, limit, offset)
	}
	return []*database.Workflow{
		{ID: uuid.New(), Name: "Workflow 1", FolderPath: folderPath},
		{ID: uuid.New(), Name: "Workflow 2", FolderPath: folderPath},
	}, nil
}
func (m *mockWorkflowServiceForWorkflows) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error) {
	if m.updateWorkflowFn != nil {
		return m.updateWorkflowFn(ctx, workflowID, input)
	}
	return &database.Workflow{
		ID:          workflowID,
		Name:        input.Name,
		Description: input.Description,
		FolderPath:  input.FolderPath,
		Tags:        input.Tags,
	}, nil
}
func (m *mockWorkflowServiceForWorkflows) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*workflow.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*workflow.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
	if m.modifyWorkflowFn != nil {
		return m.modifyWorkflowFn(ctx, workflowID, prompt, currentFlow)
	}
	return &database.Workflow{
		ID:   workflowID,
		Name: "Modified Workflow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{},
			"edges": []any{},
		},
	}, nil
}
func (m *mockWorkflowServiceForWorkflows) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*export.ExecutionTimeline, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error) {
	if m.getExecutionFn != nil {
		return m.getExecutionFn(ctx, executionID)
	}
	return &database.Execution{ID: executionID, Status: "completed"}, nil
}
func (m *mockWorkflowServiceForWorkflows) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	return nil
}

func (m *mockWorkflowServiceForWorkflows) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	return errors.New("not implemented")
}

func (m *mockWorkflowServiceForWorkflows) CheckAutomationHealth(ctx context.Context) (bool, error) {
	return true, nil
}

func setupWorkflowTestHandler(t *testing.T, workflowService WorkflowService) *Handler {
	t.Helper()

	log := logrus.New()
	log.SetOutput(os.Stderr)

	return &Handler{
		log:             log,
		repo:            &mockRepository{},
		workflowService: workflowService,
	}
}

func TestCreateWorkflow(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] creates workflow with valid data", func(t *testing.T) {
		projectID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"project_id": "` + projectID.String() + `",
			"name": "Test Workflow",
			"folder_path": "/workflows",
			"flow_definition": {
				"nodes": [],
				"edges": []
			}
		}`

		req := httptest.NewRequest("POST", "/api/v1/workflows/create", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateWorkflow(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var response database.Workflow
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Name != "Test Workflow" {
			t.Errorf("expected name 'Test Workflow', got %s", response.Name)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] rejects empty workflow name", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"name": "",
			"folder_path": "/workflows"
		}`

		req := httptest.NewRequest("POST", "/api/v1/workflows/create", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] creates workflow from AI prompt", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{
			createWorkflowFn: func(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
				if aiPrompt == "" {
					t.Error("expected non-empty AI prompt")
				}
				return &database.Workflow{
					ID:         uuid.New(),
					Name:       name,
					FolderPath: folderPath,
					FlowDefinition: database.JSONMap{
						"nodes": []any{},
						"edges": []any{},
					},
				}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"name": "AI Generated Workflow",
			"folder_path": "/workflows",
			"ai_prompt": "Create a workflow to visit google.com and take a screenshot"
		}`

		req := httptest.NewRequest("POST", "/api/v1/workflows/create", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateWorkflow(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] rejects duplicate workflow name", func(t *testing.T) {
		projectID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			createWorkflowFn: func(ctx context.Context, projID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
				return nil, services.ErrWorkflowNameConflict
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"project_id": "` + projectID.String() + `",
			"name": "Existing Workflow",
			"folder_path": "/workflows"
		}`

		req := httptest.NewRequest("POST", "/api/v1/workflows/create", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateWorkflow(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles AI generation errors", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{
			createWorkflowFn: func(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
				return nil, &workflow.AIWorkflowError{
					Reason: "Invalid prompt: too vague",
				}
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"name": "AI Workflow",
			"folder_path": "/workflows",
			"ai_prompt": "do something"
		}`

		req := httptest.NewRequest("POST", "/api/v1/workflows/create", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Invalid prompt") {
			t.Errorf("expected error message about invalid prompt, got: %s", body)
		}
	})
}

func TestGetWorkflow(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] retrieves workflow by ID", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
				return &database.Workflow{
					ID:         id,
					Name:       "Retrieved Workflow",
					FolderPath: "/workflows",
				}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/workflows/"+workflowID.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetWorkflow(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response database.Workflow
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Name != "Retrieved Workflow" {
			t.Errorf("expected name 'Retrieved Workflow', got %s", response.Name)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns 404 for non-existent workflow", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
				return nil, errors.New("workflow not found")
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/workflows/"+workflowID.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetWorkflow(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestExecuteWorkflow(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] executes workflow and returns execution ID", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			executeWorkflowFn: func(ctx context.Context, wfID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
				if wfID != workflowID {
					t.Errorf("expected workflow ID %s, got %s", workflowID, wfID)
				}
				return &database.Execution{
					ID:         uuid.New(),
					WorkflowID: wfID,
					Status:     "running",
				}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"parameters": {"url": "https://example.com"},
			"wait_for_completion": false
		}`

		req := httptest.NewRequest("POST", "/api/v1/workflows/"+workflowID.String()+"/execute", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ExecuteWorkflow(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if _, ok := response["execution_id"]; !ok {
			t.Error("response missing 'execution_id' field")
		}
	})

	t.Run("wait_for_completion returns terminal status", func(t *testing.T) {
		workflowID := uuid.New()
		executionID := uuid.New()
		completedAt := time.Now().UTC()
		polls := 0
		service := &mockWorkflowServiceForWorkflows{
			executeWorkflowFn: func(ctx context.Context, wfID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
				if wfID != workflowID {
					t.Fatalf("unexpected workflow id: %s", wfID)
				}
				return &database.Execution{ID: executionID, WorkflowID: wfID, Status: "pending"}, nil
			},
			getExecutionFn: func(ctx context.Context, execID uuid.UUID) (*database.Execution, error) {
				if execID != executionID {
					t.Fatalf("unexpected execution id: %s", execID)
				}
				polls++
				if polls >= 2 {
					return &database.Execution{
						ID:          executionID,
						WorkflowID:  workflowID,
						Status:      "completed",
						CompletedAt: &completedAt,
					}, nil
				}
				return &database.Execution{ID: executionID, WorkflowID: workflowID, Status: "running"}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"wait_for_completion": true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/execute", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ExecuteWorkflow(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["status"] != "completed" {
			t.Fatalf("expected completed status, got %v", response["status"])
		}
		completedAtValue, ok := response["completed_at"].(string)
		if !ok {
			t.Fatalf("expected completed_at string, got %T", response["completed_at"])
		}
		if completedAtValue != completedAt.UTC().Format(time.RFC3339Nano) {
			t.Fatalf("expected completed_at %s, got %s", completedAt.UTC().Format(time.RFC3339Nano), completedAtValue)
		}
	})

	t.Run("wait_for_completion respects request deadline", func(t *testing.T) {
		workflowID := uuid.New()
		executionID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			executeWorkflowFn: func(ctx context.Context, wfID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
				return &database.Execution{ID: executionID, WorkflowID: wfID, Status: "pending"}, nil
			},
			getExecutionFn: func(ctx context.Context, execID uuid.UUID) (*database.Execution, error) {
				return &database.Execution{ID: execID, WorkflowID: workflowID, Status: "running"}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"wait_for_completion": true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/execute", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()
		req = req.WithContext(timeoutCtx)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ExecuteWorkflow(w, req)

		if w.Code != http.StatusRequestTimeout {
			t.Fatalf("expected status 408, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestListWorkflows(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] lists workflows by folder path", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{
			listWorkflowsFn: func(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
				// Verify folder path filtering
				if folderPath != "/my/folder" {
					t.Errorf("expected folder path '/my/folder', got '%s'", folderPath)
				}
				if limit != 100 || offset != 0 {
					t.Errorf("expected limit=100, offset=0, got limit=%d, offset=%d", limit, offset)
				}
				return []*database.Workflow{
					{ID: uuid.New(), Name: "Workflow 1", FolderPath: "/my/folder"},
					{ID: uuid.New(), Name: "Workflow 2", FolderPath: "/my/folder"},
				}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/workflows?folder_path=/my/folder", nil)
		w := httptest.NewRecorder()

		handler.ListWorkflows(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		workflows, ok := response["workflows"].([]any)
		if !ok {
			t.Fatal("response missing 'workflows' field or wrong type")
		}
		if len(workflows) != 2 {
			t.Errorf("expected 2 workflows, got %d", len(workflows))
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] honors pagination query parameters", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{
			listWorkflowsFn: func(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
				if limit != 25 || offset != 50 {
					t.Fatalf("expected limit=25, offset=50, got %d/%d", limit, offset)
				}
				return []*database.Workflow{}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/workflows?limit=25&offset=50", nil)
		w := httptest.NewRecorder()

		handler.ListWorkflows(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] lists all workflows when no folder specified", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{
			listWorkflowsFn: func(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
				if folderPath != "" {
					t.Errorf("expected empty folder path, got '%s'", folderPath)
				}
				return []*database.Workflow{
					{ID: uuid.New(), Name: "Workflow 1", FolderPath: "/folder1"},
					{ID: uuid.New(), Name: "Workflow 2", FolderPath: "/folder2"},
					{ID: uuid.New(), Name: "Workflow 3", FolderPath: "/folder1"},
				}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		w := httptest.NewRecorder()

		handler.ListWorkflows(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		workflows, ok := response["workflows"].([]any)
		if !ok {
			t.Fatal("response missing 'workflows' field or wrong type")
		}
		if len(workflows) != 3 {
			t.Errorf("expected 3 workflows, got %d", len(workflows))
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] handles service errors", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{
			listWorkflowsFn: func(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
				return nil, errors.New("database connection failed")
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		w := httptest.NewRecorder()

		handler.ListWorkflows(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}

func TestUpdateWorkflow(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-VERSION] updates workflow with valid changes", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			updateWorkflowFn: func(ctx context.Context, id uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error) {
				if id != workflowID {
					t.Errorf("expected workflow ID %s, got %s", workflowID, id)
				}
				if input.Name != "Updated Workflow" {
					t.Errorf("expected name 'Updated Workflow', got '%s'", input.Name)
				}
				if input.Description != "New description" {
					t.Errorf("expected description 'New description', got '%s'", input.Description)
				}
				return &database.Workflow{
					ID:          id,
					Name:        input.Name,
					Description: input.Description,
					FolderPath:  input.FolderPath,
					Tags:        input.Tags,
				}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"name": "Updated Workflow",
			"description": "New description",
			"folder_path": "/updated/path",
			"tags": ["tag1", "tag2"]
		}`

		req := httptest.NewRequest("PUT", "/api/v1/workflows/"+workflowID.String(), strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWorkflow(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["name"] != "Updated Workflow" {
			t.Errorf("expected name 'Updated Workflow' in response, got '%v'", response["name"])
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-VERSION] rejects empty name", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"name": "", "folder_path": "/workflows"}`
		req := httptest.NewRequest("PUT", "/api/v1/workflows/"+workflowID.String(), strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-VERSION] handles version conflict", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			updateWorkflowFn: func(ctx context.Context, id uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error) {
				return nil, services.ErrWorkflowVersionConflict
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"name": "Updated", "folder_path": "/workflows", "expected_version": 5}`
		req := httptest.NewRequest("PUT", "/api/v1/workflows/"+workflowID.String(), strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWorkflow(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-VERSION] rejects invalid workflow ID", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"name": "Updated", "folder_path": "/workflows"}`
		req := httptest.NewRequest("PUT", "/api/v1/workflows/not-a-uuid", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "not-a-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-VERSION] handles invalid workflow definition", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			updateWorkflowFn: func(ctx context.Context, id uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error) {
				return nil, errors.New("invalid workflow definition: missing required fields")
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"name": "Updated", "folder_path": "/workflows", "flow_definition": {"invalid": true}}`
		req := httptest.NewRequest("PUT", "/api/v1/workflows/"+workflowID.String(), strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.UpdateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestModifyWorkflow(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-MODIFY] modifies workflow via AI prompt", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			modifyWorkflowFn: func(ctx context.Context, id uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
				if id != workflowID {
					t.Errorf("expected workflow ID %s, got %s", workflowID, id)
				}
				if prompt != "Add a screenshot step after navigation" {
					t.Errorf("expected prompt 'Add a screenshot step after navigation', got '%s'", prompt)
				}
				return &database.Workflow{
					ID:   id,
					Name: "Modified Workflow",
					FlowDefinition: database.JSONMap{
						"nodes": []any{
							map[string]any{"id": "1", "type": "navigate"},
							map[string]any{"id": "2", "type": "screenshot"},
						},
						"edges": []any{},
					},
				}, nil
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{
			"modification_prompt": "Add a screenshot step after navigation",
			"current_flow": {"nodes": [{"id": "1", "type": "navigate"}], "edges": []}
		}`

		req := httptest.NewRequest("POST", "/api/v1/workflows/"+workflowID.String()+"/modify", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ModifyWorkflow(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["modification_note"] != "ai" {
			t.Errorf("expected modification_note 'ai', got '%v'", response["modification_note"])
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-MODIFY] rejects empty modification prompt", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"modification_prompt": "", "current_flow": {"nodes": [], "edges": []}}`
		req := httptest.NewRequest("POST", "/api/v1/workflows/"+workflowID.String()+"/modify", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ModifyWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-MODIFY] handles AI service errors", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{
			modifyWorkflowFn: func(ctx context.Context, id uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
				return nil, errors.New("AI service unavailable")
			},
		}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"modification_prompt": "Add step", "current_flow": {"nodes": [], "edges": []}}`
		req := httptest.NewRequest("POST", "/api/v1/workflows/"+workflowID.String()+"/modify", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ModifyWorkflow(w, req)

		// Handler returns 502 Bad Gateway for AI service errors
		if w.Code != http.StatusBadGateway {
			t.Errorf("expected status %d, got %d", http.StatusBadGateway, w.Code)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-MODIFY] rejects invalid workflow ID", func(t *testing.T) {
		service := &mockWorkflowServiceForWorkflows{}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{"modification_prompt": "Add step"}`
		req := httptest.NewRequest("POST", "/api/v1/workflows/not-a-uuid/modify", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "not-a-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ModifyWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-MODIFY] rejects malformed JSON", func(t *testing.T) {
		workflowID := uuid.New()
		service := &mockWorkflowServiceForWorkflows{}
		handler := setupWorkflowTestHandler(t, service)

		reqBody := `{invalid json`
		req := httptest.NewRequest("POST", "/api/v1/workflows/"+workflowID.String()+"/modify", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", workflowID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.ModifyWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}
