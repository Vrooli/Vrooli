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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services"
)

// mockWorkflowServiceForWorkflows provides workflow service implementation for workflow handler tests
type mockWorkflowServiceForWorkflows struct {
	createWorkflowFn          func(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error)
	getWorkflowFn             func(ctx context.Context, id uuid.UUID) (*database.Workflow, error)
	deleteWorkflowFn          func(ctx context.Context, id uuid.UUID) error
	listWorkflowsByProjectFn  func(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error)
	executeWorkflowFn         func(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
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
func (m *mockWorkflowServiceForWorkflows) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *mockWorkflowServiceForWorkflows) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input services.WorkflowUpdateInput) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*services.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*services.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*services.ExecutionTimeline, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*services.ExecutionExportPreview, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForWorkflows) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	return nil
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
				return nil, &services.AIWorkflowError{
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
}
