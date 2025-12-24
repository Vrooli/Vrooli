package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestNormalizeProjectRelPath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPath  string
		wantValid bool
	}{
		{"empty string", "", "", false},
		{"dot only", ".", "", false},
		{"valid path", "workflows/test.json", "workflows/test.json", true},
		{"leading slash", "/workflows/test.json", "workflows/test.json", true},
		{"trailing slash", "workflows/test/", "workflows/test", true},
		{"backslashes", "workflows\\test.json", "workflows/test.json", true},
		{"parent directory", "../outside", "", false},
		{"double parent", "../../outside", "", false},
		// Note: absolute paths are normalized by stripping leading slash
		{"absolute path", "/absolute/path", "absolute/path", true},
		{"whitespace", "  workflows/test.json  ", "workflows/test.json", true},
		{"nested path", "workflows/folder/subfolder/test.json", "workflows/folder/subfolder/test.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotValid := normalizeProjectRelPath(tt.input)
			if gotValid != tt.wantValid {
				t.Errorf("normalizeProjectRelPath(%q) valid = %v, want %v", tt.input, gotValid, tt.wantValid)
			}
			if gotValid && gotPath != tt.wantPath {
				t.Errorf("normalizeProjectRelPath(%q) = %q, want %q", tt.input, gotPath, tt.wantPath)
			}
		})
	}
}

func TestSafeJoinProjectPath(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		relPath     string
		wantErr     bool
	}{
		{"valid join", "/home/user/project", "workflows/test.json", false},
		{"empty root", "", "workflows/test.json", true},
		{"dot root", ".", "workflows/test.json", true},
		{"empty rel path", "/home/user/project", "", true},
		{"traversal attempt", "/home/user/project", "../outside", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := safeJoinProjectPath(tt.projectRoot, tt.relPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("safeJoinProjectPath(%q, %q) error = %v, wantErr %v", tt.projectRoot, tt.relPath, err, tt.wantErr)
			}
		})
	}
}

func TestWorkflowsDir(t *testing.T) {
	tests := []struct {
		name    string
		project *database.ProjectIndex
		want    string
	}{
		{"nil project", nil, "workflows"},
		{"empty folder path", &database.ProjectIndex{FolderPath: ""}, "workflows"},
		{"whitespace folder path", &database.ProjectIndex{FolderPath: "   "}, "workflows"},
		{"valid folder path", &database.ProjectIndex{FolderPath: "/home/user/project"}, filepath.Join("/home/user/project", "workflows")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workflowsDir(tt.project)
			if got != tt.want {
				t.Errorf("workflowsDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorkflowFolderPathFromRelPath(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		want    string
	}{
		{"file at root", "workflows/test.json", "/"},
		{"file in subfolder", "workflows/folder/test.json", "/folder"},
		{"deeply nested", "workflows/a/b/c/test.json", "/a/b/c"},
		{"no workflows prefix", "test.json", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workflowFolderPathFromRelPath(tt.relPath)
			if got != tt.want {
				t.Errorf("workflowFolderPathFromRelPath(%q) = %q, want %q", tt.relPath, got, tt.want)
			}
		})
	}
}

// ============================================================================
// GetProjectFileTree Tests
// ============================================================================

func TestGetProjectFileTree_InvalidProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/invalid-uuid/files", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetProjectFileTree(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetProjectFileTree_ProjectNotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/files", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.GetProjectFileTree(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetProjectFileTree_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	// Create temp directory for project
	tmpDir, err := os.MkdirTemp("", "project")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workflows directory
	os.MkdirAll(filepath.Join(tmpDir, "workflows"), 0755)

	projectID := uuid.New()
	project := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: tmpDir,
	}
	repo.AddProject(project)
	catalogSvc.AddProject(project)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/files", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.GetProjectFileTree(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ProjectFileTreeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should have at least the workflows folder
	hasWorkflowsFolder := false
	for _, entry := range response.Entries {
		if entry.Path == "workflows" && entry.Kind == ProjectEntryKindFolder {
			hasWorkflowsFolder = true
			break
		}
	}
	if !hasWorkflowsFolder {
		t.Error("expected workflows folder in entries")
	}
}

func TestGetProjectFileTree_WithWorkflows(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "project")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "workflows"), 0755)

	projectID := uuid.New()
	project := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: tmpDir,
	}
	repo.AddProject(project)
	catalogSvc.AddProject(project)

	// Add a workflow
	workflowID := uuid.New()
	repo.AddWorkflow(&database.WorkflowIndex{
		ID:         workflowID,
		ProjectID:  &projectID,
		Name:       "test-workflow",
		FilePath:   "test.workflow.json",
		FolderPath: "/",
		Version:    1,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/files", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.GetProjectFileTree(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response ProjectFileTreeResponse
	json.Unmarshal(rr.Body.Bytes(), &response)

	// Should have workflow entry
	hasWorkflow := false
	for _, entry := range response.Entries {
		if entry.Kind == ProjectEntryKindWorkflowFile && entry.WorkflowID != nil {
			hasWorkflow = true
			break
		}
	}
	if !hasWorkflow {
		t.Error("expected workflow file in entries")
	}
}

// ============================================================================
// ReadProjectFile Tests
// ============================================================================

func TestReadProjectFile_InvalidProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/invalid-uuid/files/read?path=workflows/test.json", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ReadProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestReadProjectFile_InvalidPath(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/files/read?path=", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.ReadProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestReadProjectFile_NonWorkflowPath(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: "/tmp/test",
	})

	// Path not under workflows/ should be rejected
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/files/read?path=assets/image.png", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.ReadProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for non-workflow path, got %d", rr.Code)
	}
}

func TestReadProjectFile_ProjectNotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/files/read?path=workflows/test.json", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.ReadProjectFile(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

// ============================================================================
// MkdirProjectPath Tests
// ============================================================================

func TestMkdirProjectPath_InvalidProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := MkdirProjectPathRequest{Path: "new-folder"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/invalid-uuid/files/mkdir", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.MkdirProjectPath(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestMkdirProjectPath_InvalidJSON(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/mkdir", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.MkdirProjectPath(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestMkdirProjectPath_InvalidPath(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	body := MkdirProjectPathRequest{Path: ""} // Empty path
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/mkdir", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.MkdirProjectPath(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestMkdirProjectPath_ProjectNotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	body := MkdirProjectPathRequest{Path: "new-folder"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/mkdir", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.MkdirProjectPath(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestMkdirProjectPath_Success(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "project")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: tmpDir,
	})

	body := MkdirProjectPathRequest{Path: "new-folder"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/mkdir", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.MkdirProjectPath(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify directory was created
	newDir := filepath.Join(tmpDir, "new-folder")
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

// ============================================================================
// WriteProjectWorkflowFile Tests
// ============================================================================

func TestWriteProjectWorkflowFile_InvalidProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := WriteProjectWorkflowFileRequest{
		Path: "workflows/test.workflow.json",
		Workflow: ProjectWorkflowFileWrite{
			Name:           "test",
			FlowDefinition: map[string]any{},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/invalid-uuid/files/write", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.WriteProjectWorkflowFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestWriteProjectWorkflowFile_InvalidPath(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	// Path not under workflows/
	body := WriteProjectWorkflowFileRequest{
		Path: "assets/test.json",
		Workflow: ProjectWorkflowFileWrite{
			Name:           "test",
			FlowDefinition: map[string]any{},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/write", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.WriteProjectWorkflowFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestWriteProjectWorkflowFile_InvalidExtension(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	// Wrong extension
	body := WriteProjectWorkflowFileRequest{
		Path: "workflows/test.json", // Should be .workflow.json
		Workflow: ProjectWorkflowFileWrite{
			Name:           "test",
			FlowDefinition: map[string]any{},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/write", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.WriteProjectWorkflowFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// MoveProjectFile Tests
// ============================================================================

func TestMoveProjectFile_InvalidProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := MoveProjectFileRequest{
		FromPath: "workflows/old.json",
		ToPath:   "workflows/new.json",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/invalid-uuid/files/move", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.MoveProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestMoveProjectFile_InvalidFromPath(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	body := MoveProjectFileRequest{
		FromPath: "", // Invalid
		ToPath:   "workflows/new.json",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/move", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.MoveProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestMoveProjectFile_InvalidToPath(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	body := MoveProjectFileRequest{
		FromPath: "workflows/old.json",
		ToPath:   "", // Invalid
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/move", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.MoveProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestMoveProjectFile_ProjectNotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	body := MoveProjectFileRequest{
		FromPath: "workflows/old.json",
		ToPath:   "workflows/new.json",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/move", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.MoveProjectFile(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

// ============================================================================
// DeleteProjectFile Tests
// ============================================================================

func TestDeleteProjectFile_InvalidProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := DeleteProjectFileRequest{Path: "workflows/test.json"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/invalid-uuid/files/delete", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.DeleteProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestDeleteProjectFile_InvalidPath(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	projectID := uuid.New()
	repo.AddProject(&database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: "/tmp/test",
	})

	body := DeleteProjectFileRequest{Path: ""} // Invalid
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/delete", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.DeleteProjectFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestDeleteProjectFile_ProjectNotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	body := DeleteProjectFileRequest{Path: "workflows/test.json"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/delete", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.DeleteProjectFile(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestDeleteProjectFile_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	// Create temp directory with a file
	tmpDir, err := os.MkdirTemp("", "project")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test-file.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	projectID := uuid.New()
	project := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: tmpDir,
	}
	repo.AddProject(project)
	catalogSvc.AddProject(project)

	body := DeleteProjectFileRequest{Path: "test-file.txt"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/delete", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.DeleteProjectFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

// ============================================================================
// ResyncProjectFiles Tests
// ============================================================================

func TestResyncProjectFiles_InvalidProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/invalid-uuid/files/resync", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ResyncProjectFiles(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestResyncProjectFiles_ProjectNotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/resync", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.ResyncProjectFiles(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestResyncProjectFiles_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "project")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "workflows"), 0755)

	projectID := uuid.New()
	project := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test",
		FolderPath: tmpDir,
	}
	repo.AddProject(project)
	catalogSvc.AddProject(project)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/resync", nil)
	req = withURLParam(req, "id", projectID.String())
	rr := httptest.NewRecorder()

	handler.ResyncProjectFiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ResyncProjectFilesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.ProjectID != projectID.String() {
		t.Errorf("expected project ID %s, got %s", projectID.String(), response.ProjectID)
	}
	if response.ProjectRoot != tmpDir {
		t.Errorf("expected project root %s, got %s", tmpDir, response.ProjectRoot)
	}
}

// ============================================================================
// Struct Tests
// ============================================================================

func TestProjectEntryKind_Values(t *testing.T) {
	if ProjectEntryKindFolder != "folder" {
		t.Errorf("expected folder, got %s", ProjectEntryKindFolder)
	}
	if ProjectEntryKindWorkflowFile != "workflow_file" {
		t.Errorf("expected workflow_file, got %s", ProjectEntryKindWorkflowFile)
	}
	if ProjectEntryKindAssetFile != "asset_file" {
		t.Errorf("expected asset_file, got %s", ProjectEntryKindAssetFile)
	}
}

func TestProjectEntry_JSONSerialization(t *testing.T) {
	workflowID := "test-id"
	entry := ProjectEntry{
		ID:         "wf:123",
		ProjectID:  "project-123",
		Path:       "workflows/test.json",
		Kind:       ProjectEntryKindWorkflowFile,
		WorkflowID: &workflowID,
		Metadata:   map[string]any{"version": 1},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ProjectEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != entry.ID {
		t.Errorf("expected ID %s, got %s", entry.ID, decoded.ID)
	}
	if decoded.Kind != entry.Kind {
		t.Errorf("expected Kind %s, got %s", entry.Kind, decoded.Kind)
	}
	if decoded.WorkflowID == nil || *decoded.WorkflowID != workflowID {
		t.Errorf("expected WorkflowID %s, got %v", workflowID, decoded.WorkflowID)
	}
}
