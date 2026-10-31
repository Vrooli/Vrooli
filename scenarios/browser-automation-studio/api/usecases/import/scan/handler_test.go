package scan_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/testutil/mocks"
	"github.com/vrooli/browser-automation-studio/usecases/import/scan"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

func TestScanAssets_ReturnsFilesAndFolders(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	scanner := mocks.NewImportDirectoryScanner()
	projecter := mocks.NewImportProjectIndexer()
	workflowIndexer := mocks.NewImportWorkflowIndexer()

	assetsPath := "/tmp/assets"
	scanner.PutDirectory(assetsPath, []shared.FileEntry{
		{Name: "Screenshots", Path: "/tmp/assets/Screenshots", IsDir: true},
		{Name: "logo.png", Path: "/tmp/assets/logo.png", IsDir: false, Size: 2048},
		{Name: "notes.pdf", Path: "/tmp/assets/notes.pdf", IsDir: false, Size: 1024},
	})

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

	scanner := mocks.NewImportDirectoryScanner()
	projecter := mocks.NewImportProjectIndexer()
	workflowIndexer := mocks.NewImportWorkflowIndexer()

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
