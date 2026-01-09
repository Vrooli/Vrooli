package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// createTestHandler creates a handler with all mock dependencies for testing.
func createTestHandler() (*Handler, *MockCatalogService, *MockExecutionService, *MockRepository, *MockHub, *MockStorage) {
	repo := NewMockRepository()
	hub := NewMockHub()
	catalogSvc := NewMockCatalogService()
	execSvc := NewMockExecutionService()
	storageMock := NewMockStorage()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel) // Suppress logs during tests

	handler := &Handler{
		catalogService:   catalogSvc,
		executionService: execSvc,
		repo:             repo,
		wsHub:            hub,
		storage:          storageMock,
		log:              log,
	}

	return handler, catalogSvc, execSvc, repo, hub, storageMock
}

// withURLParam adds a chi URL parameter to the request context
func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ============================================================================
// GetExecution Tests
// ============================================================================

func TestGetExecution_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	workflowID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:         executionID,
		WorkflowID: workflowID,
		Status:     "completed",
		StartedAt:  time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String(), nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecution(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetExecution_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/invalid-uuid", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetExecution(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetExecution_NotFound(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.GetExecutionError = database.ErrNotFound

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String(), nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecution(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetExecution_HydrateError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:     executionID,
		Status: "completed",
	})
	execSvc.HydrateExecutionProtoError = errors.New("hydration failed")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String(), nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecution(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// ListExecutions Tests
// ============================================================================

func TestListExecutions_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	// Add some executions
	workflowID := uuid.New()
	for i := 0; i < 3; i++ {
		execSvc.AddExecution(&database.ExecutionIndex{
			ID:         uuid.New(),
			WorkflowID: workflowID,
			Status:     "completed",
			StartedAt:  time.Now(),
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	rr := httptest.NewRecorder()

	handler.ListExecutions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListExecutions_WithWorkflowIDFilter(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     "completed",
		StartedAt:  time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions?workflow_id="+workflowID.String(), nil)
	rr := httptest.NewRecorder()

	handler.ListExecutions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestListExecutions_DatabaseError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.ListExecutionsError = errors.New("database error")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	rr := httptest.NewRecorder()

	handler.ListExecutions(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestListExecutions_HydrateError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.AddExecution(&database.ExecutionIndex{
		ID:        uuid.New(),
		Status:    "completed",
		StartedAt: time.Now(),
	})
	execSvc.HydrateExecutionProtoError = errors.New("hydration failed")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	rr := httptest.NewRecorder()

	handler.ListExecutions(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// StopExecution Tests
// ============================================================================

func TestStopExecution_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:     executionID,
		Status: "running",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/stop", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.StopExecution(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !execSvc.StopExecutionCalled {
		t.Fatal("expected StopExecution to be called")
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["status"] != "stopped" {
		t.Fatalf("expected status 'stopped', got %v", response["status"])
	}
}

func TestStopExecution_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/invalid-uuid/stop", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.StopExecution(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestStopExecution_ServiceError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.StopExecutionError = errors.New("failed to stop")

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/stop", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.StopExecution(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// ResumeExecution Tests
// ============================================================================

func TestResumeExecution_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:     executionID,
		Status: "interrupted",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/resume", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.ResumeExecution(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !execSvc.ResumeExecutionCalled {
		t.Fatal("expected ResumeExecution to be called")
	}
}

func TestResumeExecution_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/invalid-uuid/resume", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ResumeExecution(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestResumeExecution_CannotBeResumed(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.ResumeExecutionError = errors.New("execution cannot be resumed: not in resumable state")

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/resume", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.ResumeExecution(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for non-resumable execution, got %d", rr.Code)
	}
}

func TestResumeExecution_ServiceError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.ResumeExecutionError = errors.New("internal error")

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/resume", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.ResumeExecution(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// GetExecutionScreenshots Tests
// ============================================================================

func TestGetExecutionScreenshots_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:     executionID,
		Status: "completed",
	})

	// Provide mock screenshots to avoid artifact gap detection
	execSvc.ExecutionScreenshots = []*basexecution.ExecutionScreenshot{
		{
			StepIndex: 0,
			NodeId:    uuid.New().String(),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/screenshots", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionScreenshots(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetExecutionScreenshots_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/invalid-uuid/screenshots", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetExecutionScreenshots(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetExecutionScreenshots_ServiceError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.GetExecutionScreenshotsError = errors.New("storage error")

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/screenshots", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionScreenshots(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// GetExecutionTimeline Tests
// ============================================================================

func TestGetExecutionTimeline_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:         executionID,
		Status:     "completed",
		ResultPath: "/tmp/test/result.json",
	})

	// Provide mock timeline proto with entries to avoid artifact gap detection
	execSvc.ExecutionTimelineProto = &bastimeline.ExecutionTimeline{
		ExecutionId: executionID.String(),
		Entries: []*bastimeline.TimelineEntry{
			{
				Id:          uuid.New().String(),
				SequenceNum: 0,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/timeline", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionTimeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetExecutionTimeline_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/invalid-uuid/timeline", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetExecutionTimeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// Pagination Tests
// ============================================================================

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		defaultLimit   int
		defaultOffset  int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "no params uses defaults",
			queryString:    "",
			defaultLimit:   100,
			defaultOffset:  0,
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "explicit limit and offset",
			queryString:    "?limit=50&offset=10",
			defaultLimit:   100,
			defaultOffset:  0,
			expectedLimit:  50,
			expectedOffset: 10,
		},
		{
			name:           "negative values ignored",
			queryString:    "?limit=-10&offset=-5",
			defaultLimit:   100,
			defaultOffset:  0,
			expectedLimit:  100,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/executions"+tt.queryString, nil)
			limit, offset := parsePaginationParams(req, tt.defaultLimit, tt.defaultOffset)

			if limit != tt.expectedLimit {
				t.Errorf("expected limit %d, got %d", tt.expectedLimit, limit)
			}
			if offset != tt.expectedOffset {
				t.Errorf("expected offset %d, got %d", tt.expectedOffset, offset)
			}
		})
	}
}
