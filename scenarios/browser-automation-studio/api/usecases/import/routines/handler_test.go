package routines_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/usecases/import/routines"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

// mockDirectoryScanner implements shared.DirectoryScanner for testing.
type mockDirectoryScanner struct {
	files       map[string][]byte
	directories map[string][]shared.FileEntry
}

func newMockDirectoryScanner() *mockDirectoryScanner {
	return &mockDirectoryScanner{
		files:       make(map[string][]byte),
		directories: make(map[string][]shared.FileEntry),
	}
}

func (m *mockDirectoryScanner) ScanDirectory(ctx context.Context, path string) ([]shared.FileEntry, error) {
	if entries, ok := m.directories[path]; ok {
		return entries, nil
	}
	return []shared.FileEntry{}, nil
}

func (m *mockDirectoryScanner) ScanForPattern(ctx context.Context, root string, pattern string, maxDepth int) ([]shared.FileEntry, error) {
	return []shared.FileEntry{}, nil
}

func (m *mockDirectoryScanner) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockDirectoryScanner) WriteFile(ctx context.Context, path string, content []byte, perm os.FileMode) error {
	m.files[path] = content
	return nil
}

func (m *mockDirectoryScanner) CopyFile(ctx context.Context, src, dst string) error {
	if content, ok := m.files[src]; ok {
		m.files[dst] = content
		return nil
	}
	return os.ErrNotExist
}

func (m *mockDirectoryScanner) Exists(ctx context.Context, path string) (bool, error) {
	if _, ok := m.files[path]; ok {
		return true, nil
	}
	if _, ok := m.directories[path]; ok {
		return true, nil
	}
	return false, nil
}

func (m *mockDirectoryScanner) IsDir(ctx context.Context, path string) (bool, error) {
	_, ok := m.directories[path]
	return ok, nil
}

func (m *mockDirectoryScanner) Stat(ctx context.Context, path string) (os.FileInfo, error) {
	return nil, nil
}

// mockWorkflowIndexer implements shared.WorkflowIndexer for testing.
type mockWorkflowIndexer struct {
	workflows map[uuid.UUID]*shared.WorkflowIndexData
}

func newMockWorkflowIndexer() *mockWorkflowIndexer {
	return &mockWorkflowIndexer{
		workflows: make(map[uuid.UUID]*shared.WorkflowIndexData),
	}
}

func (m *mockWorkflowIndexer) CreateWorkflowIndex(ctx context.Context, projectID uuid.UUID, workflow *shared.WorkflowIndexData) error {
	m.workflows[workflow.ID] = workflow
	return nil
}

func (m *mockWorkflowIndexer) GetWorkflowByFilePath(ctx context.Context, projectID uuid.UUID, filePath string) (*shared.WorkflowIndexData, error) {
	for _, w := range m.workflows {
		if w.FilePath == filePath && w.ProjectID == projectID {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockWorkflowIndexer) GetWorkflowByID(ctx context.Context, id uuid.UUID) (*shared.WorkflowIndexData, error) {
	if w, ok := m.workflows[id]; ok {
		return w, nil
	}
	return nil, nil
}

func (m *mockWorkflowIndexer) UpdateWorkflowIndex(ctx context.Context, workflow *shared.WorkflowIndexData) error {
	m.workflows[workflow.ID] = workflow
	return nil
}

func (m *mockWorkflowIndexer) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID) ([]*shared.WorkflowIndexData, error) {
	var result []*shared.WorkflowIndexData
	for _, w := range m.workflows {
		if w.ProjectID == projectID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWorkflowIndexer) DeleteWorkflowIndex(ctx context.Context, id uuid.UUID) error {
	delete(m.workflows, id)
	return nil
}

// mockProjectIndexer implements shared.ProjectIndexer for testing.
type mockProjectIndexer struct {
	projects map[uuid.UUID]*shared.ProjectIndexData
}

func newMockProjectIndexer() *mockProjectIndexer {
	return &mockProjectIndexer{
		projects: make(map[uuid.UUID]*shared.ProjectIndexData),
	}
}

func (m *mockProjectIndexer) GetProjectByID(ctx context.Context, id uuid.UUID) (*shared.ProjectIndexData, error) {
	if p, ok := m.projects[id]; ok {
		return p, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockProjectIndexer) GetProjectByFolderPath(ctx context.Context, folderPath string) (*shared.ProjectIndexData, error) {
	for _, p := range m.projects {
		if p.FolderPath == folderPath {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockProjectIndexer) ListProjects(ctx context.Context, limit, offset int) ([]*shared.ProjectIndexData, error) {
	var result []*shared.ProjectIndexData
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockProjectIndexer) CreateProject(ctx context.Context, project *shared.ProjectIndexData) error {
	m.projects[project.ID] = project
	return nil
}

func setupTestHandler(t *testing.T) (*routines.Handler, *mockDirectoryScanner, *mockWorkflowIndexer, *mockProjectIndexer) {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel) // Quiet logs during tests

	scanner := newMockDirectoryScanner()
	indexer := newMockWorkflowIndexer()
	projecter := newMockProjectIndexer()

	service := routines.NewService(scanner, indexer, projecter, log)
	handler := routines.NewHandler(service, log)

	return handler, scanner, indexer, projecter
}

func TestInspectRoutine_InvalidProjectID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/projects/invalid-uuid/routines/inspect", bytes.NewBufferString(`{"file_path": "/test.workflow.json"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestInspectRoutine_MissingFilePath(t *testing.T) {
	handler, _, _, projecter := setupTestHandler(t)

	projectID := uuid.New()
	projecter.projects[projectID] = &shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/projects/test",
	}

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/routines/inspect", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestInspectRoutine_FileNotFound(t *testing.T) {
	handler, _, _, projecter := setupTestHandler(t)

	projectID := uuid.New()
	projecter.projects[projectID] = &shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/projects/test",
	}

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	body := `{"file_path": "/projects/test/workflows/missing.workflow.json"}`
	req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/routines/inspect", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp routines.InspectRoutineResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Exists {
		t.Error("expected Exists to be false for missing file")
	}
}

func TestInspectRoutine_ValidWorkflow(t *testing.T) {
	handler, scanner, _, projecter := setupTestHandler(t)

	projectID := uuid.New()
	projecter.projects[projectID] = &shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/projects/test",
	}

	// Add a valid workflow file
	workflowContent := `{
		"id": "test-workflow-id",
		"name": "Test Workflow",
		"version": 1,
		"nodes": [],
		"edges": []
	}`
	scanner.files["/projects/test/workflows/test.workflow.json"] = []byte(workflowContent)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	body := `{"file_path": "/projects/test/workflows/test.workflow.json"}`
	req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/routines/inspect", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp routines.InspectRoutineResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Exists {
		t.Error("expected Exists to be true")
	}
	if !resp.IsValid {
		t.Error("expected IsValid to be true")
	}
}

func TestScanRoutines_EmptyDirectory(t *testing.T) {
	handler, scanner, _, projecter := setupTestHandler(t)

	projectID := uuid.New()
	projecter.projects[projectID] = &shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/projects/test",
	}

	// Add workflows directory (empty)
	scanner.directories["/projects/test/workflows"] = []shared.FileEntry{}

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/routines/scan", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp routines.ScanRoutinesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(resp.Entries))
	}
}

func TestImportRoutine_InvalidProjectID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/projects/invalid-uuid/routines/import", bytes.NewBufferString(`{"file_path": "/test.workflow.json"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
