package scan_test

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
	"github.com/vrooli/browser-automation-studio/usecases/import/scan"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

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

func (m *mockDirectoryScanner) ScanDirectory(_ context.Context, path string) ([]shared.FileEntry, error) {
	if entries, ok := m.directories[path]; ok {
		return entries, nil
	}
	return []shared.FileEntry{}, nil
}

func (m *mockDirectoryScanner) ScanForPattern(_ context.Context, _ string, _ string, _ int) ([]shared.FileEntry, error) {
	return []shared.FileEntry{}, nil
}

func (m *mockDirectoryScanner) ReadFile(_ context.Context, path string) ([]byte, error) {
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockDirectoryScanner) WriteFile(_ context.Context, path string, content []byte, _ os.FileMode) error {
	m.files[path] = content
	return nil
}

func (m *mockDirectoryScanner) CopyFile(_ context.Context, src, dst string) error {
	if content, ok := m.files[src]; ok {
		m.files[dst] = content
		return nil
	}
	return os.ErrNotExist
}

func (m *mockDirectoryScanner) Exists(_ context.Context, path string) (bool, error) {
	if _, ok := m.files[path]; ok {
		return true, nil
	}
	if _, ok := m.directories[path]; ok {
		return true, nil
	}
	return false, nil
}

func (m *mockDirectoryScanner) IsDir(_ context.Context, path string) (bool, error) {
	_, ok := m.directories[path]
	return ok, nil
}

func (m *mockDirectoryScanner) Stat(_ context.Context, _ string) (os.FileInfo, error) {
	return nil, nil
}

type mockWorkflowIndexer struct{}

func (m *mockWorkflowIndexer) CreateWorkflowIndex(_ context.Context, _ uuid.UUID, _ *shared.WorkflowIndexData) error {
	return nil
}

func (m *mockWorkflowIndexer) GetWorkflowByFilePath(_ context.Context, _ uuid.UUID, _ string) (*shared.WorkflowIndexData, error) {
	return nil, nil
}

func (m *mockWorkflowIndexer) GetWorkflowByID(_ context.Context, _ uuid.UUID) (*shared.WorkflowIndexData, error) {
	return nil, nil
}

func (m *mockWorkflowIndexer) UpdateWorkflowIndex(_ context.Context, _ *shared.WorkflowIndexData) error {
	return nil
}

func (m *mockWorkflowIndexer) ListWorkflowsByProject(_ context.Context, _ uuid.UUID) ([]*shared.WorkflowIndexData, error) {
	return nil, nil
}

func (m *mockWorkflowIndexer) DeleteWorkflowIndex(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockProjectIndexer struct{}

func (m *mockProjectIndexer) GetProjectByID(_ context.Context, _ uuid.UUID) (*shared.ProjectIndexData, error) {
	return nil, os.ErrNotExist
}

func (m *mockProjectIndexer) GetProjectByFolderPath(_ context.Context, _ string) (*shared.ProjectIndexData, error) {
	return nil, nil
}

func (m *mockProjectIndexer) ListProjects(_ context.Context, _ int, _ int) ([]*shared.ProjectIndexData, error) {
	return nil, nil
}

func (m *mockProjectIndexer) CreateProject(_ context.Context, _ *shared.ProjectIndexData) error {
	return nil
}

func TestScanAssets_ReturnsFilesAndFolders(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	scanner := newMockDirectoryScanner()
	projecter := &mockProjectIndexer{}
	workflowIndexer := &mockWorkflowIndexer{}

	assetsPath := "/tmp/assets"
	scanner.directories[assetsPath] = []shared.FileEntry{
		{Name: "Screenshots", Path: "/tmp/assets/Screenshots", IsDir: true},
		{Name: "logo.png", Path: "/tmp/assets/logo.png", IsDir: false, Size: 2048},
		{Name: "notes.pdf", Path: "/tmp/assets/notes.pdf", IsDir: false, Size: 1024},
	}

	service := scan.NewService(scanner, projecter, workflowIndexer, log)
	handler := scan.NewHandler(service, log)

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	body, _ := json.Marshal(map[string]interface{}{
		"mode": "assets",
		"path": assetsPath,
	})
	req := httptest.NewRequest(http.MethodPost, "/fs/scan", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp scan.ScanResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Entries))
	}

	if resp.Entries[0].IsDir && resp.Entries[1].IsDir {
		t.Fatalf("expected a file entry to be included")
	}
}

func TestScanWorkflows_MissingProjectID(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	scanner := newMockDirectoryScanner()
	projecter := &mockProjectIndexer{}
	workflowIndexer := &mockWorkflowIndexer{}

	service := scan.NewService(scanner, projecter, workflowIndexer, log)
	handler := scan.NewHandler(service, log)

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	body, _ := json.Marshal(map[string]interface{}{
		"mode": "workflows",
		"path": "/tmp/workflows",
	})
	req := httptest.NewRequest(http.MethodPost, "/fs/scan", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
