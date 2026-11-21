package routines_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/testutil/mocks"
	"github.com/vrooli/browser-automation-studio/usecases/import/routines"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

func setupTestHandler(t *testing.T) (*routines.Handler, *mocks.ImportDirectoryScanner, *mocks.ImportWorkflowIndexer, *mocks.ImportProjectIndexer) {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel) // Quiet logs during tests

	scanner := mocks.NewImportDirectoryScanner()
	indexer := mocks.NewImportWorkflowIndexer()
	projecter := mocks.NewImportProjectIndexer()

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
	projecter.PutProject(&shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/projects/test",
	})

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
	projecter.PutProject(&shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/projects/test",
	})

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
	projecter.PutProject(&shared.ProjectIndexData{
		ID:         projectID,
		Name:       "Test Project",
		FolderPath: "/projects/test",
	})

	// Add a valid workflow file
	workflowContent := `{
		"id": "test-workflow-id",
		"name": "Test Workflow",
		"version": 1,
		"nodes": [],
		"edges": []
	}`
	scanner.PutFile("/projects/test/workflows/test.workflow.json", []byte(workflowContent))

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
