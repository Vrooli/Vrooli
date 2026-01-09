package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

// ============================================================================
// GetExecutionRecordedTraces Tests
// ============================================================================

func TestGetExecutionRecordedTraces_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-traces", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedTraces(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response executionTracesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.ExecutionID != executionID.String() {
		t.Errorf("expected execution_id %q, got %q", executionID.String(), response.ExecutionID)
	}

	// Verify it's called (default returns empty slice)
	_ = execSvc
}

func TestGetExecutionRecordedTraces_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/invalid-uuid/recorded-traces", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedTraces(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetExecutionRecordedTraces_ServiceError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.GetExecutionTraceArtifactsError = errors.New("trace retrieval failed")

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-traces", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedTraces(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestGetExecutionRecordedTraces_WithTraces(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	size1 := int64(12345)
	size2 := int64(67890)
	execSvc.ExecutionTraceArtifacts = []workflow.ExecutionFileArtifact{
		{
			ArtifactID:  "trace-1",
			StorageURL:  "/artifacts/traces/trace-1.zip",
			SizeBytes:   &size1,
			ContentType: "application/zip",
			Label:       "trace-1.zip",
		},
		{
			ArtifactID:  "trace-2",
			StorageURL:  "/artifacts/traces/trace-2.zip",
			SizeBytes:   &size2,
			ContentType: "application/zip",
			Label:       "trace-2.zip",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-traces", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedTraces(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response executionTracesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(response.Traces) != 2 {
		t.Errorf("expected 2 traces, got %d", len(response.Traces))
	}

	if response.Traces[0].ArtifactID != "trace-1" {
		t.Errorf("expected first trace artifact_id 'trace-1', got %q", response.Traces[0].ArtifactID)
	}
}
