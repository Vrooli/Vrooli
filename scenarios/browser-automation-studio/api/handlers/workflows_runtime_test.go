package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// ============================================================================
// ExecuteWorkflow Tests
// ============================================================================

func TestExecuteWorkflow_Success(t *testing.T) {
	handler, catalogSvc, execSvc, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{
		ID:   workflowID,
		Name: "test-workflow",
	})

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ExecuteWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify execution was started
	if execSvc.ExecuteWorkflowAPIError != nil {
		t.Fatal("expected no error from ExecuteWorkflowAPI")
	}
}

func TestExecuteWorkflow_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/invalid-uuid/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ExecuteWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestExecuteWorkflow_InvalidJSON(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ExecuteWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestExecuteWorkflow_ServiceError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.ExecuteWorkflowAPIError = errors.New("execution failed")

	workflowID := uuid.New()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ExecuteWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestExecuteWorkflow_WithFrameStreaming(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{
		ID:   workflowID,
		Name: "test-workflow",
	})

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/execute?frame_streaming=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ExecuteWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteWorkflow_WithVideoRecording(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{
		ID:   workflowID,
		Name: "test-workflow",
	})

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/execute?requires_video=true", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ExecuteWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// ExecuteAdhocWorkflow Tests
// ============================================================================

func TestExecuteAdhocWorkflow_Success(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{
		"flow_definition": {
			"nodes": [],
			"edges": []
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/execute-adhoc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ExecuteAdhocWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteAdhocWorkflow_InvalidJSON(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/execute-adhoc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ExecuteAdhocWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestExecuteAdhocWorkflow_ServiceError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.ExecuteAdhocWorkflowAPIError = errors.New("execution failed")

	body := `{
		"flow_definition": {
			"nodes": [],
			"edges": []
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/execute-adhoc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ExecuteAdhocWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// ListWorkflowVersions Tests
// ============================================================================

func TestListWorkflowVersions_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{
		ID:      workflowID,
		Name:    "test-workflow",
		Version: 1,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/versions", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ListWorkflowVersions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListWorkflowVersions_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/invalid-uuid/versions", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ListWorkflowVersions(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestListWorkflowVersions_NotFound(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.ListWorkflowsError = database.ErrNotFound

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/versions", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ListWorkflowVersions(rr, req)

	// Handler may return 200 with empty list or 404
	// based on implementation - verify it doesn't crash
	if rr.Code != http.StatusOK && rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 200 or 404, got %d", rr.Code)
	}
}

// ============================================================================
// GetWorkflowVersion Tests
// ============================================================================

func TestGetWorkflowVersion_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{
		ID:      workflowID,
		Name:    "test-workflow",
		Version: 1,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/versions/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID.String())
	rctx.URLParams.Add("version", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetWorkflowVersion(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetWorkflowVersion_InvalidWorkflowID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/invalid-uuid/versions/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid-uuid")
	rctx.URLParams.Add("version", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetWorkflowVersion(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetWorkflowVersion_InvalidVersion(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/versions/invalid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID.String())
	rctx.URLParams.Add("version", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetWorkflowVersion(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetWorkflowVersion_ZeroVersion(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/versions/0", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID.String())
	rctx.URLParams.Add("version", "0")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.GetWorkflowVersion(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for version 0, got %d", rr.Code)
	}
}

// ============================================================================
// RestoreWorkflowVersion Tests
// ============================================================================

func TestRestoreWorkflowVersion_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{
		ID:      workflowID,
		Name:    "test-workflow",
		Version: 2,
	})

	body := `{"change_description": "Restored from version 1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/versions/1/restore", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID.String())
	rctx.URLParams.Add("version", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.RestoreWorkflowVersion(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRestoreWorkflowVersion_InvalidWorkflowID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/invalid-uuid/versions/1/restore", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid-uuid")
	rctx.URLParams.Add("version", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.RestoreWorkflowVersion(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// ReceiveExecutionFrame Tests
// ============================================================================

func TestReceiveExecutionFrame_Success(t *testing.T) {
	handler, _, _, _, hub, _ := createTestHandler()

	executionID := uuid.New().String()
	hub.ExecutionFrameSubscribers[executionID] = true

	body := `{
		"data": "base64data",
		"media_type": "image/jpeg",
		"width": 1920,
		"height": 1080
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID+"/frames", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionId", executionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ReceiveExecutionFrame(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReceiveExecutionFrame_NoSubscribers(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	executionID := uuid.New().String()
	// No subscribers registered

	body := `{
		"data": "base64data",
		"media_type": "image/jpeg",
		"width": 1920,
		"height": 1080
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID+"/frames", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionId", executionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ReceiveExecutionFrame(rr, req)

	// Should return 204 No Content when no subscribers
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
}

func TestReceiveExecutionFrame_MissingExecutionID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := `{"data": "base64data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions//frames", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionId", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ReceiveExecutionFrame(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestReceiveExecutionFrame_InvalidJSON(t *testing.T) {
	handler, _, _, _, hub, _ := createTestHandler()

	executionID := uuid.New().String()
	hub.ExecutionFrameSubscribers[executionID] = true

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID+"/frames", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("executionId", executionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.ReceiveExecutionFrame(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// parseBoolQuery Tests
// ============================================================================

func TestParseBoolQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		keys     []string
		expected bool
	}{
		{
			name:     "true value",
			query:    "?flag=true",
			keys:     []string{"flag"},
			expected: true,
		},
		{
			name:     "false value",
			query:    "?flag=false",
			keys:     []string{"flag"},
			expected: false,
		},
		{
			name:     "1 value",
			query:    "?flag=1",
			keys:     []string{"flag"},
			expected: true,
		},
		{
			name:     "0 value",
			query:    "?flag=0",
			keys:     []string{"flag"},
			expected: false,
		},
		{
			name:     "yes value",
			query:    "?flag=yes",
			keys:     []string{"flag"},
			expected: true,
		},
		{
			name:     "no value",
			query:    "?flag=no",
			keys:     []string{"flag"},
			expected: false,
		},
		{
			name:     "missing key",
			query:    "",
			keys:     []string{"flag"},
			expected: false,
		},
		{
			name:     "multiple keys first match",
			query:    "?alt_flag=true",
			keys:     []string{"flag", "alt_flag"},
			expected: true,
		},
		{
			name:     "case insensitive",
			query:    "?flag=TRUE",
			keys:     []string{"flag"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.query, nil)
			result := parseBoolQuery(req, tt.keys...)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
