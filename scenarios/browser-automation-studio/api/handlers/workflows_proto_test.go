package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

// ============================================================================
// CreateWorkflow Tests
// ============================================================================

func TestCreateWorkflow_Success(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	// CreateWorkflowRequest proto fields: project_id, name, folder_path, flow_definition, ai_prompt
	body := map[string]any{
		"name":        "test-workflow",
		"folder_path": "/workflows",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateWorkflow(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify service was called
	if catalogSvc.CreateWorkflowError != nil {
		t.Error("unexpected create workflow error")
	}
}

func TestCreateWorkflow_InvalidJSON(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateWorkflow_NameConflict(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.CreateWorkflowError = workflow.ErrWorkflowNameConflict

	body := map[string]any{
		"name": "existing-workflow",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateWorkflow(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateWorkflow_InternalError(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.CreateWorkflowError = errors.New("database error")

	body := map[string]any{
		"name": "test-workflow",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// ListWorkflows Tests
// ============================================================================

func TestListWorkflows_Success(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	rr := httptest.NewRecorder()

	handler.ListWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListWorkflows_WithPagination(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?limit=10&offset=5", nil)
	rr := httptest.NewRecorder()

	handler.ListWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestListWorkflows_WithFolderPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?folder_path=/my/folder", nil)
	rr := httptest.NewRecorder()

	handler.ListWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestListWorkflows_WithProjectID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	projectID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?project_id="+projectID.String(), nil)
	rr := httptest.NewRecorder()

	handler.ListWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestListWorkflows_ServiceError(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.ListWorkflowsError = errors.New("database error")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	rr := httptest.NewRecorder()

	handler.ListWorkflows(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// GetWorkflow Tests
// ============================================================================

func TestGetWorkflow_Success(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String(), nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetWorkflow_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/invalid-uuid", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.GetWorkflowAPIError = database.ErrNotFound

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String(), nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflow(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetWorkflow_WithVersion(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"?version=2", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflow(rr, req)

	// Should work with version parameter
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestGetWorkflow_InvalidVersion(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	// Invalid version (not a number) should be ignored
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"?version=invalid", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflow(rr, req)

	// Invalid version is ignored, should still return 200
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 (invalid version ignored), got %d", rr.Code)
	}
}

func TestGetWorkflow_ServiceError(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.GetWorkflowAPIError = errors.New("database error")

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String(), nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// UpdateWorkflow Tests
// ============================================================================

func TestUpdateWorkflow_Success(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	body := map[string]any{
		"name":        "updated-workflow",
		"description": "Updated description",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workflows/"+workflowID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.UpdateWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateWorkflow_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := map[string]any{"name": "updated"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workflows/invalid-uuid", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.UpdateWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateWorkflow_InvalidJSON(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workflows/"+workflowID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.UpdateWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateWorkflow_VersionConflict(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.UpdateWorkflowError = workflow.ErrWorkflowVersionConflict

	workflowID := uuid.New()
	body := map[string]any{"name": "updated"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workflows/"+workflowID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.UpdateWorkflow(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateWorkflow_ServiceError(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.UpdateWorkflowError = errors.New("database error")

	workflowID := uuid.New()
	body := map[string]any{"name": "updated"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workflows/"+workflowID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.UpdateWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// DeleteWorkflow Tests
// ============================================================================

func TestDeleteWorkflow_Success(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+workflowID.String(), nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.DeleteWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteWorkflow_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/invalid-uuid", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.DeleteWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestDeleteWorkflow_NotFound(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.DeleteWorkflowError = database.ErrNotFound

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+workflowID.String(), nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.DeleteWorkflow(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestDeleteWorkflow_ServiceError(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	catalogSvc.DeleteWorkflowError = errors.New("database error")

	workflowID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+workflowID.String(), nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.DeleteWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}
