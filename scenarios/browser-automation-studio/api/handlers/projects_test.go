package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// ============================================================================
// CreateProject Tests
// ============================================================================

func TestCreateProject_Success(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Create a temp directory for the project
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test-project")

	body := `{
		"name": "Test Project",
		"folder_path": "` + projectPath + `",
		"description": "A test project"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateProject(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateProject_MissingName(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{
		"folder_path": "/tmp/test-project"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateProject(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["code"] != "MISSING_REQUIRED_FIELD" {
		t.Errorf("expected error code MISSING_REQUIRED_FIELD, got %v", response["code"])
	}
}

func TestCreateProject_MissingFolderPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{
		"name": "Test Project"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateProject(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateProject_DuplicateName(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	// Add existing project
	catalogSvc.AddProject(&database.ProjectIndex{
		Name:       "Existing Project",
		FolderPath: "/tmp/existing",
	})

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "new-project")

	body := `{
		"name": "Existing Project",
		"folder_path": "` + projectPath + `"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateProject(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateProject_DuplicateFolderPath(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Add existing project with the same folder path
	catalogSvc.AddProject(&database.ProjectIndex{
		Name:       "Existing Project",
		FolderPath: projectPath,
	})

	body := `{
		"name": "New Project",
		"folder_path": "` + projectPath + `"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateProject(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateProject_InvalidJSON(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateProject(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateProject_WithPreset(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "test-project")

	body := `{
		"name": "Test Project",
		"folder_path": "` + projectPath + `",
		"preset": "recommended"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateProject(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify preset folders were created
	expectedFolders := []string{
		"workflows/actions",
		"workflows/flows",
		"workflows/cases",
		"assets",
	}
	for _, folder := range expectedFolders {
		folderPath := filepath.Join(projectPath, folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			t.Errorf("expected folder %q to be created", folder)
		}
	}
}

// ============================================================================
// GetProject Tests
// ============================================================================

func TestGetProject_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	catalogSvc.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String(), nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.GetProject(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetProject_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/invalid-uuid", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetProject(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.GetProjectError = database.ErrNotFound

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String(), nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.GetProject(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

// ============================================================================
// ListProjects Tests
// ============================================================================

func TestListProjects_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	// Add some projects
	for i := 0; i < 3; i++ {
		catalogSvc.AddProject(&database.ProjectIndex{
			Name:       "Project " + string(rune('A'+i)),
			FolderPath: "/tmp/project-" + string(rune('a'+i)),
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rr := httptest.NewRecorder()

	handler.ListProjects(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListProjects_Empty(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rr := httptest.NewRecorder()

	handler.ListProjects(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestListProjects_DatabaseError(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.ListProjectsError = errors.New("database error")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rr := httptest.NewRecorder()

	handler.ListProjects(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// UpdateProject Tests
// ============================================================================

func TestUpdateProject_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	catalogSvc.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "Original Name",
		FolderPath: "/tmp/test",
	})

	body := `{
		"name": "Updated Name"
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.UpdateProject(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateProject_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{"name": "Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/invalid-uuid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.UpdateProject(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateProject_NotFound(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.GetProjectError = database.ErrNotFound

	projectID := uuid.New()
	body := `{"name": "Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.UpdateProject(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

// ============================================================================
// DeleteProject Tests
// ============================================================================

func TestDeleteProject_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	catalogSvc.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String(), nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.DeleteProject(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify the service was called
	if !catalogSvc.DeleteProjectCalled {
		t.Error("expected DeleteProject to be called")
	}
}

func TestDeleteProject_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/invalid-uuid", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.DeleteProject(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestDeleteProject_ServiceError(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.DeleteProjectError = errors.New("failed to delete")

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String(), nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.DeleteProject(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// GetProjectWorkflows Tests
// ============================================================================

func TestGetProjectWorkflows_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	catalogSvc.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/tmp/test",
	})

	// Add some workflows
	for i := 0; i < 3; i++ {
		catalogSvc.AddWorkflow(&database.WorkflowIndex{
			ProjectID: &projectID,
			Name:      "Workflow " + string(rune('A'+i)),
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/workflows", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.GetProjectWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetProjectWorkflows_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/invalid-uuid/workflows", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetProjectWorkflows(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// BulkDeleteProjectWorkflows Tests
// ============================================================================

func TestBulkDeleteProjectWorkflows_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	catalogSvc.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/tmp/test",
	})

	workflowIDs := []uuid.UUID{uuid.New(), uuid.New()}
	for _, wfID := range workflowIDs {
		catalogSvc.AddWorkflow(&database.WorkflowIndex{
			ID:        wfID,
			ProjectID: &projectID,
			Name:      "Workflow",
		})
	}

	body := `{
		"workflow_ids": ["` + workflowIDs[0].String() + `", "` + workflowIDs[1].String() + `"]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/workflows/bulk-delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.BulkDeleteProjectWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["status"] != "deleted" {
		t.Errorf("expected status 'deleted', got %v", response["status"])
	}
	if response["deleted_count"].(float64) != 2 {
		t.Errorf("expected deleted_count 2, got %v", response["deleted_count"])
	}
}

func TestBulkDeleteProjectWorkflows_EmptyList(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	body := `{"workflow_ids": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/workflows/bulk-delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.BulkDeleteProjectWorkflows(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty list, got %d", rr.Code)
	}
}

func TestBulkDeleteProjectWorkflows_InvalidWorkflowID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	body := `{"workflow_ids": ["invalid-uuid"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/workflows/bulk-delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.BulkDeleteProjectWorkflows(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid workflow ID, got %d", rr.Code)
	}
}

// ============================================================================
// ExecuteAllProjectWorkflows Tests
// ============================================================================

func TestExecuteAllProjectWorkflows_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	catalogSvc.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/tmp/test",
	})

	// Add some workflows
	for i := 0; i < 2; i++ {
		catalogSvc.AddWorkflow(&database.WorkflowIndex{
			ProjectID: &projectID,
			Name:      "Workflow " + string(rune('A'+i)),
		})
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/execute-all", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.ExecuteAllProjectWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteAllProjectWorkflows_NoWorkflows(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	catalogSvc.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/execute-all", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.ExecuteAllProjectWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["message"] != "No workflows found in project" {
		t.Errorf("expected message about no workflows, got %v", response["message"])
	}
}

func TestExecuteAllProjectWorkflows_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/invalid-uuid/execute-all", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ExecuteAllProjectWorkflows(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
