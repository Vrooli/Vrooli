package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services"
)

// mockWorkflowServiceForProjects provides minimal workflow service implementation for project tests
type mockWorkflowServiceForProjects struct {
	createProjectFn          func(ctx context.Context, project *database.Project) error
	getProjectByNameFn       func(ctx context.Context, name string) (*database.Project, error)
	getProjectByFolderPathFn func(ctx context.Context, folderPath string) (*database.Project, error)
	getProjectFn             func(ctx context.Context, id uuid.UUID) (*database.Project, error)
	listProjectsFn           func(ctx context.Context, limit, offset int) ([]*database.Project, error)
	updateProjectFn          func(ctx context.Context, project *database.Project) error
	deleteProjectFn          func(ctx context.Context, id uuid.UUID) error
	getProjectStatsFn        func(ctx context.Context, projectID uuid.UUID) (map[string]any, error)
	getProjectsStatsFn       func(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error)
	deleteProjectWorkflowsFn func(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error
}

func (m *mockWorkflowServiceForProjects) CreateProject(ctx context.Context, project *database.Project) error {
	if m.createProjectFn != nil {
		return m.createProjectFn(ctx, project)
	}
	project.ID = uuid.New()
	return nil
}

func (m *mockWorkflowServiceForProjects) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	if m.getProjectByNameFn != nil {
		return m.getProjectByNameFn(ctx, name)
	}
	return nil, nil
}

func (m *mockWorkflowServiceForProjects) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	if m.getProjectByFolderPathFn != nil {
		return m.getProjectByFolderPathFn(ctx, folderPath)
	}
	return nil, nil
}

func (m *mockWorkflowServiceForProjects) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	if m.getProjectFn != nil {
		return m.getProjectFn(ctx, id)
	}
	return &database.Project{
		ID:          id,
		Name:        "Test Project",
		Description: "Test description",
		FolderPath:  "/tmp/test-project",
	}, nil
}

func (m *mockWorkflowServiceForProjects) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	if m.listProjectsFn != nil {
		return m.listProjectsFn(ctx, limit, offset)
	}
	return []*database.Project{
		{
			ID:          uuid.New(),
			Name:        "Project 1",
			Description: "Description 1",
			FolderPath:  "/tmp/project1",
		},
		{
			ID:          uuid.New(),
			Name:        "Project 2",
			Description: "Description 2",
			FolderPath:  "/tmp/project2",
		},
	}, nil
}

func (m *mockWorkflowServiceForProjects) UpdateProject(ctx context.Context, project *database.Project) error {
	if m.updateProjectFn != nil {
		return m.updateProjectFn(ctx, project)
	}
	return nil
}

func (m *mockWorkflowServiceForProjects) DeleteProject(ctx context.Context, id uuid.UUID) error {
	if m.deleteProjectFn != nil {
		return m.deleteProjectFn(ctx, id)
	}
	return nil
}

func (m *mockWorkflowServiceForProjects) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	if m.getProjectStatsFn != nil {
		return m.getProjectStatsFn(ctx, projectID)
	}
	return map[string]any{
		"workflow_count":  5,
		"execution_count": 10,
	}, nil
}

func (m *mockWorkflowServiceForProjects) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	if m.getProjectsStatsFn != nil {
		return m.getProjectsStatsFn(ctx, projectIDs)
	}
	result := make(map[uuid.UUID]map[string]any, len(projectIDs))
	for _, id := range projectIDs {
		result[id] = map[string]any{
			"workflow_count":  5,
			"execution_count": 10,
		}
	}
	return result, nil
}

func (m *mockWorkflowServiceForProjects) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	if m.deleteProjectWorkflowsFn != nil {
		return m.deleteProjectWorkflowsFn(ctx, projectID, workflowIDs)
	}
	return nil
}

// Stub implementations for other WorkflowService methods (matching handler.go interface)
func (m *mockWorkflowServiceForProjects) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input services.WorkflowUpdateInput) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*services.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*services.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*services.ExecutionTimeline, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*services.ExecutionExportPreview, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *mockWorkflowServiceForProjects) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	return nil
}
func (m *mockWorkflowServiceForProjects) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}

func setupProjectTestHandler(t *testing.T, workflowService WorkflowService) *Handler {
	t.Helper()

	log := logrus.New()
	log.SetOutput(os.Stderr)

	return &Handler{
		log:             log,
		repo:            &mockRepository{},
		workflowService: workflowService,
	}
}

func TestCreateProject(t *testing.T) {
	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] creates project with valid data", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("VROOLI_ROOT", rootDir)
		projectDir := filepath.Join(rootDir, "projects", "test-project")
		service := &mockWorkflowServiceForProjects{}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{
			"name": "Test Project",
			"description": "A test project",
			"folder_path": "` + projectDir + `"
		}`

		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if _, ok := response["project_id"]; !ok {
			t.Errorf("response missing 'project_id' field, got: %v", response)
		}
	})

	t.Run("[REQ:BAS-PROJECT-CREATE-VALIDATION] rejects empty name", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("VROOLI_ROOT", rootDir)
		projectDir := filepath.Join(rootDir, "projects", "test-project")
		service := &mockWorkflowServiceForProjects{}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{
			"name": "",
			"folder_path": "` + projectDir + `"
		}`

		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("[REQ:BAS-PROJECT-CREATE-VALIDATION] rejects empty folder path", func(t *testing.T) {
		service := &mockWorkflowServiceForProjects{}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{
			"name": "Test Project",
			"folder_path": ""
		}`

		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("[REQ:BAS-PROJECT-CREATE-VALIDATION] rejects duplicate project name", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("VROOLI_ROOT", rootDir)
		projectDir := filepath.Join(rootDir, "projects", "existing-project")
		service := &mockWorkflowServiceForProjects{
			getProjectByNameFn: func(ctx context.Context, name string) (*database.Project, error) {
				return &database.Project{ID: uuid.New(), Name: name}, nil
			},
		}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{
			"name": "Existing Project",
			"folder_path": "` + projectDir + `"
		}`

		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})

	t.Run("returns 500 when name uniqueness check fails", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("VROOLI_ROOT", rootDir)
		projectDir := filepath.Join(rootDir, "projects", "error-name")
		service := &mockWorkflowServiceForProjects{
			getProjectByNameFn: func(ctx context.Context, name string) (*database.Project, error) {
				return nil, errors.New("db down")
			},
		}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{"name": "Test", "folder_path": "` + projectDir + `"}`
		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("returns 500 when folder uniqueness check fails", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("VROOLI_ROOT", rootDir)
		projectDir := filepath.Join(rootDir, "projects", "error-folder")
		service := &mockWorkflowServiceForProjects{
			getProjectByFolderPathFn: func(ctx context.Context, folderPath string) (*database.Project, error) {
				return nil, errors.New("db down")
			},
		}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{"name": "Test", "folder_path": "` + projectDir + `"}`
		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("rejects folder paths outside project root", func(t *testing.T) {
		rootDir := t.TempDir()
		outsideDir := t.TempDir()
		t.Setenv("VROOLI_ROOT", rootDir)
		service := &mockWorkflowServiceForProjects{}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{
			"name": "Test Project",
			"folder_path": "` + filepath.Join(outsideDir, "project") + `"
		}`

		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("[REQ:BAS-PROJECT-CREATE-VALIDATION] rejects invalid JSON", func(t *testing.T) {
		service := &mockWorkflowServiceForProjects{}
		handler := setupProjectTestHandler(t, service)

		reqBody := `{invalid json`

		req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateProject(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestGetProject(t *testing.T) {
	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] retrieves project by ID", func(t *testing.T) {
		projectID := uuid.New()
		service := &mockWorkflowServiceForProjects{
			getProjectFn: func(ctx context.Context, id uuid.UUID) (*database.Project, error) {
				return &database.Project{
					ID:          id,
					Name:        "Retrieved Project",
					Description: "Test description",
					FolderPath:  "/tmp/test",
				}, nil
			},
		}
		handler := setupProjectTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", projectID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetProject(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response database.Project
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Name != "Retrieved Project" {
			t.Errorf("expected name 'Retrieved Project', got %s", response.Name)
		}
	})

	t.Run("[REQ:BAS-PROJECT-CREATE-VALIDATION] rejects invalid UUID", func(t *testing.T) {
		service := &mockWorkflowServiceForProjects{}
		handler := setupProjectTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/projects/invalid-uuid", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		handler.GetProject(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestListProjects(t *testing.T) {
	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] lists all projects", func(t *testing.T) {
		projectsReturned := []*database.Project{
			{ID: uuid.New(), Name: "Project A", FolderPath: "/tmp/a"},
			{ID: uuid.New(), Name: "Project B", FolderPath: "/tmp/b"},
		}
		service := &mockWorkflowServiceForProjects{
			listProjectsFn: func(ctx context.Context, limit, offset int) ([]*database.Project, error) {
				return projectsReturned, nil
			},
			getProjectsStatsFn: func(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
				if len(projectIDs) != len(projectsReturned) {
					t.Fatalf("expected %d project IDs, got %d", len(projectsReturned), len(projectIDs))
				}
				stats := make(map[uuid.UUID]map[string]any, len(projectIDs))
				for _, id := range projectIDs {
					stats[id] = map[string]any{"workflow_count": 42}
				}
				return stats, nil
			},
		}
		handler := setupProjectTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/projects", nil)
		w := httptest.NewRecorder()

		handler.ListProjects(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response struct {
			Projects []ProjectWithStats `json:"projects"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(response.Projects) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(response.Projects))
		}
		for _, project := range response.Projects {
			count, ok := project.Stats["workflow_count"].(float64)
			if !ok || count != 42 {
				t.Fatalf("expected aggregated stats to be included")
			}
		}
	})

	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] respects pagination parameters", func(t *testing.T) {
		service := &mockWorkflowServiceForProjects{
			listProjectsFn: func(ctx context.Context, limit, offset int) ([]*database.Project, error) {
				if limit != 10 || offset != 5 {
					t.Errorf("expected limit=10, offset=5, got limit=%d, offset=%d", limit, offset)
				}
				return []*database.Project{}, nil
			},
		}
		handler := setupProjectTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/projects?limit=10&offset=5", nil)
		w := httptest.NewRecorder()

		handler.ListProjects(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("returns 500 when bulk stats query fails", func(t *testing.T) {
		service := &mockWorkflowServiceForProjects{
			getProjectsStatsFn: func(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
				return nil, errors.New("boom")
			},
		}
		handler := setupProjectTestHandler(t, service)

		req := httptest.NewRequest("GET", "/api/v1/projects", nil)
		w := httptest.NewRecorder()
		handler.ListProjects(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}
