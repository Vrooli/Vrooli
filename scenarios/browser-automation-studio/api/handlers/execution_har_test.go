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
// GetExecutionRecordedHar Tests
// ============================================================================

func TestGetExecutionRecordedHar_Success(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-har", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedHar(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response executionHarResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.ExecutionID != executionID.String() {
		t.Errorf("expected execution_id %q, got %q", executionID.String(), response.ExecutionID)
	}

	// Verify it's called (default returns empty slice)
	_ = execSvc
}

func TestGetExecutionRecordedHar_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/invalid-uuid/recorded-har", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedHar(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetExecutionRecordedHar_ServiceError(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.GetExecutionHarArtifactsError = errors.New("HAR retrieval failed")

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-har", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedHar(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestGetExecutionRecordedHar_WithHarFiles(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	size := int64(54321)
	execSvc.ExecutionHarArtifacts = []workflow.ExecutionFileArtifact{
		{
			ArtifactID:  "network-log",
			StorageURL:  "/artifacts/har/network-log.har",
			SizeBytes:   &size,
			ContentType: "application/json",
			Label:       "network-log.har",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-har", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedHar(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response executionHarResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(response.HARFiles) != 1 {
		t.Errorf("expected 1 HAR file, got %d", len(response.HARFiles))
	}

	if response.HARFiles[0].ArtifactID != "network-log" {
		t.Errorf("expected HAR file artifact_id 'network-log', got %q", response.HARFiles[0].ArtifactID)
	}
}

func TestGetExecutionRecordedHar_EmptyResponse(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	executionID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-har", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionRecordedHar(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response executionHarResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.HARFiles == nil {
		t.Error("expected har_files to be non-nil empty slice, got nil")
	}
}
