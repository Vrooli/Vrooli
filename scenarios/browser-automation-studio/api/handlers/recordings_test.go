package handlers

import (
	"context"
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
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
)

// ============================================================================
// Mock Recording Service
// ============================================================================

// MockRecordingService implements IngestionServiceInterface for testing
type MockRecordingService struct {
	ImportArchiveFunc  func(ctx context.Context, archivePath string, opts archiveingestion.IngestionOptions) (*archiveingestion.IngestionResult, error)
	ImportArchiveError error
	ImportArchiveResult *archiveingestion.IngestionResult
}

func NewMockRecordingService() *MockRecordingService {
	return &MockRecordingService{
		ImportArchiveResult: &archiveingestion.IngestionResult{
			Execution: &database.ExecutionIndex{
				ID:     uuid.New(),
				Status: "completed",
			},
			Workflow: &database.WorkflowIndex{
				ID:   uuid.New(),
				Name: "imported-workflow",
			},
			Project: &database.ProjectIndex{
				ID:   uuid.New(),
				Name: "imported-project",
			},
			FrameCount: 10,
			AssetCount: 10,
			DurationMs: 5000,
		},
	}
}

func (m *MockRecordingService) ImportArchive(ctx context.Context, archivePath string, opts archiveingestion.IngestionOptions) (*archiveingestion.IngestionResult, error) {
	if m.ImportArchiveFunc != nil {
		return m.ImportArchiveFunc(ctx, archivePath, opts)
	}
	if m.ImportArchiveError != nil {
		return nil, m.ImportArchiveError
	}
	return m.ImportArchiveResult, nil
}

// Compile-time interface check
var _ archiveingestion.IngestionServiceInterface = (*MockRecordingService)(nil)

// ============================================================================
// Helper to create handler with recording service
// ============================================================================

func createTestHandlerWithRecordingService() (*Handler, *MockRecordingService) {
	repo := NewMockRepository()
	hub := NewMockHub()
	catalogSvc := NewMockCatalogService()
	execSvc := NewMockExecutionService()
	storageMock := NewMockStorage()
	recordingSvc := NewMockRecordingService()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		catalogService:   catalogSvc,
		executionService: execSvc,
		repo:             repo,
		wsHub:            hub,
		storage:          storageMock,
		recordingService: recordingSvc,
		log:              log,
	}

	return handler, recordingSvc
}

// ============================================================================
// ImportRecording Tests
// ============================================================================

func TestImportRecording_ServiceUnavailable(t *testing.T) {
	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingService = nil // Make service unavailable

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/import", nil)
	rr := httptest.NewRecorder()

	handler.ImportRecording(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when service unavailable, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestImportRecording_EmptyBody(t *testing.T) {
	handler, recordingSvc := createTestHandlerWithRecordingService()

	// Configure mock to return an error for empty archives (simulates actual service behavior)
	recordingSvc.ImportArchiveError = errors.New("empty archive")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/import", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/octet-stream")
	rr := httptest.NewRecorder()

	handler.ImportRecording(rr, req)

	// Should fail because the recording service rejects empty archives
	if rr.Code == http.StatusCreated {
		t.Fatalf("expected error for empty body, got success with status %d", rr.Code)
	}
}

// ============================================================================
// ServeRecordingAsset Tests
// ============================================================================

func TestServeRecordingAsset_StorageUnavailable(t *testing.T) {
	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingsRoot = "" // Make storage unavailable

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/assets/"+executionID.String()+"/screenshot.png", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionID", executionID.String())
	rctx.URLParams.Add("*", "screenshot.png")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeRecordingAsset(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when storage unavailable, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServeRecordingAsset_InvalidExecutionID(t *testing.T) {
	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingsRoot = "/tmp/recordings"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/assets/invalid-uuid/screenshot.png", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionID", "invalid-uuid")
	rctx.URLParams.Add("*", "screenshot.png")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeRecordingAsset(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestServeRecordingAsset_MissingAssetPath(t *testing.T) {
	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingsRoot = "/tmp/recordings"

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/assets/"+executionID.String()+"/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionID", executionID.String())
	rctx.URLParams.Add("*", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeRecordingAsset(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing asset path, got %d", rr.Code)
	}
}

func TestServeRecordingAsset_PathTraversalAttempt(t *testing.T) {
	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingsRoot = "/tmp/recordings"

	executionID := uuid.New()
	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../etc/passwd"},
		{"double parent", "../../etc/passwd"},
		{"embedded traversal", "subdir/../../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/assets/"+executionID.String()+"/"+tt.path, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("executionID", executionID.String())
			rctx.URLParams.Add("*", tt.path)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			handler.ServeRecordingAsset(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for path traversal attempt, got %d", rr.Code)
			}
		})
	}
}

func TestServeRecordingAsset_Success(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "recordings-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingsRoot = tmpDir

	executionID := uuid.New()

	// Create execution directory and test file
	execDir := filepath.Join(tmpDir, executionID.String())
	if err := os.MkdirAll(execDir, 0755); err != nil {
		t.Fatalf("failed to create exec dir: %v", err)
	}

	testContent := []byte("fake-png-content")
	testFile := filepath.Join(execDir, "screenshot.png")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/assets/"+executionID.String()+"/screenshot.png", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionID", executionID.String())
	rctx.URLParams.Add("*", "screenshot.png")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeRecordingAsset(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify content was served
	if rr.Body.Len() == 0 {
		t.Fatal("expected response body to contain file content")
	}
}

func TestServeRecordingAsset_NotFound(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "recordings-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingsRoot = tmpDir

	executionID := uuid.New()

	// Create execution directory but no file
	execDir := filepath.Join(tmpDir, executionID.String())
	if err := os.MkdirAll(execDir, 0755); err != nil {
		t.Fatalf("failed to create exec dir: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/assets/"+executionID.String()+"/nonexistent.png", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionID", executionID.String())
	rctx.URLParams.Add("*", "nonexistent.png")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeRecordingAsset(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for nonexistent file, got %d", rr.Code)
	}
}

func TestServeRecordingAsset_CacheHeaders(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "recordings-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	handler, _ := createTestHandlerWithRecordingService()
	handler.recordingsRoot = tmpDir

	executionID := uuid.New()

	// Create execution directory and test file
	execDir := filepath.Join(tmpDir, executionID.String())
	if err := os.MkdirAll(execDir, 0755); err != nil {
		t.Fatalf("failed to create exec dir: %v", err)
	}

	testFile := filepath.Join(execDir, "cached.png")
	if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/assets/"+executionID.String()+"/cached.png", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionID", executionID.String())
	rctx.URLParams.Add("*", "cached.png")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ServeRecordingAsset(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Fatal("expected Cache-Control header to be set")
	}
	if !strings.Contains(cacheControl, "public") {
		t.Fatalf("expected Cache-Control to contain 'public', got %s", cacheControl)
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestParseRecordingOptions_ValidProjectID(t *testing.T) {
	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/import?project_id="+projectID.String(), nil)

	opts, err := parseRecordingOptions(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.ProjectID == nil {
		t.Fatal("expected project ID to be set")
	}
	if *opts.ProjectID != projectID {
		t.Fatalf("expected project ID %s, got %s", projectID, *opts.ProjectID)
	}
}

func TestParseRecordingOptions_InvalidProjectID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/import?project_id=invalid-uuid", nil)

	_, err := parseRecordingOptions(req)
	if err == nil {
		t.Fatal("expected error for invalid project ID")
	}
}

func TestParseRecordingOptions_ValidWorkflowID(t *testing.T) {
	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/import?workflow_id="+workflowID.String(), nil)

	opts, err := parseRecordingOptions(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.WorkflowID == nil {
		t.Fatal("expected workflow ID to be set")
	}
	if *opts.WorkflowID != workflowID {
		t.Fatalf("expected workflow ID %s, got %s", workflowID, *opts.WorkflowID)
	}
}

func TestParseRecordingOptions_Names(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/import?project_name=MyProject&workflow_name=MyWorkflow", nil)

	opts, err := parseRecordingOptions(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.ProjectName != "MyProject" {
		t.Fatalf("expected project name 'MyProject', got %s", opts.ProjectName)
	}
	if opts.WorkflowName != "MyWorkflow" {
		t.Fatalf("expected workflow name 'MyWorkflow', got %s", opts.WorkflowName)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{"first non-empty", []string{"", "second", "third"}, "second"},
		{"all empty", []string{"", "", ""}, ""},
		{"first is non-empty", []string{"first", "second"}, "first"},
		{"whitespace only", []string{"  ", "second"}, "second"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstNonEmpty(tt.values...)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
