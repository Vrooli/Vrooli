package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// ============================================================================
// ListExports Tests
// ============================================================================

func TestListExports_Success(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	// Create test exports
	workflowID := uuid.New()
	executionID := uuid.New()
	repo.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, WorkflowID: workflowID, StartedAt: time.Now()})
	repo.AddExport(&database.ExportIndex{
		ExecutionID: executionID,
		WorkflowID:  &workflowID,
		Name:        "export-1",
		Format:      "mp4",
		Status:      "completed",
	})
	repo.AddExport(&database.ExportIndex{
		ExecutionID: executionID,
		WorkflowID:  &workflowID,
		Name:        "export-2",
		Format:      "gif",
		Status:      "completed",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports", nil)
	rr := httptest.NewRecorder()

	handler.ListExports(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(response["total"].(float64))
	if total != 2 {
		t.Fatalf("expected 2 exports, got %d", total)
	}
}

func TestListExports_FilterByExecutionID(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	executionID1 := uuid.New()
	executionID2 := uuid.New()
	repo.AddExecution(&database.ExecutionIndex{ID: executionID1, StartedAt: time.Now()})
	repo.AddExecution(&database.ExecutionIndex{ID: executionID2, StartedAt: time.Now()})

	repo.AddExport(&database.ExportIndex{
		ExecutionID: executionID1,
		Name:        "export-for-exec-1",
		Format:      "mp4",
		Status:      "completed",
	})
	repo.AddExport(&database.ExportIndex{
		ExecutionID: executionID2,
		Name:        "export-for-exec-2",
		Format:      "gif",
		Status:      "completed",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports?execution_id="+executionID1.String(), nil)
	rr := httptest.NewRecorder()

	handler.ListExports(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(response["total"].(float64))
	if total != 1 {
		t.Fatalf("expected 1 export for execution, got %d", total)
	}
}

func TestListExports_FilterByWorkflowID(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	executionID := uuid.New()
	repo.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, WorkflowID: workflowID, StartedAt: time.Now()})

	repo.AddExport(&database.ExportIndex{
		ExecutionID: executionID,
		WorkflowID:  &workflowID,
		Name:        "workflow-export",
		Format:      "mp4",
		Status:      "completed",
	})
	repo.AddExport(&database.ExportIndex{
		ExecutionID: uuid.New(),
		Name:        "other-export",
		Format:      "gif",
		Status:      "completed",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports?workflow_id="+workflowID.String(), nil)
	rr := httptest.NewRecorder()

	handler.ListExports(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(response["total"].(float64))
	if total != 1 {
		t.Fatalf("expected 1 export for workflow, got %d", total)
	}
}

func TestListExports_InvalidExecutionID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports?execution_id=invalid-uuid", nil)
	rr := httptest.NewRecorder()

	handler.ListExports(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestListExports_InvalidWorkflowID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports?workflow_id=invalid-uuid", nil)
	rr := httptest.NewRecorder()

	handler.ListExports(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// GetExport Tests
// ============================================================================

func TestGetExport_Success(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	executionID := uuid.New()
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, StartedAt: time.Now()})

	export := &database.ExportIndex{
		ExecutionID: executionID,
		Name:        "test-export",
		Format:      "mp4",
		Status:      "completed",
	}
	repo.AddExport(export)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/"+export.ID.String(), nil)
	req = withURLParam(req, "id", export.ID.String())
	rr := httptest.NewRecorder()

	handler.GetExport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	exportData := response["export"].(map[string]any)
	if exportData["name"] != "test-export" {
		t.Fatalf("expected name 'test-export', got %v", exportData["name"])
	}
}

func TestGetExport_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	exportID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/"+exportID.String(), nil)
	req = withURLParam(req, "id", exportID.String())
	rr := httptest.NewRecorder()

	handler.GetExport(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetExport_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/invalid-uuid", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// CreateExport Tests
// ============================================================================

func TestCreateExport_Success(t *testing.T) {
	handler, _, execSvc, repo, _, _ := createTestHandler()

	executionID := uuid.New()
	workflowID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:         executionID,
		WorkflowID: workflowID,
		Status:     "completed",
	})

	body := CreateExportRequest{
		ExecutionID: executionID.String(),
		WorkflowID:  workflowID.String(),
		Name:        "new-export",
		Format:      "mp4",
		Settings:    map[string]any{"quality": "high"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["status"] != "created" {
		t.Fatalf("expected status 'created', got %v", response["status"])
	}

	// Verify export was created
	exports, _ := repo.ListExports(context.Background(), 10, 0)
	if len(exports) != 1 {
		t.Fatalf("expected 1 export in repo, got %d", len(exports))
	}
}

func TestCreateExport_MissingExecutionID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := CreateExportRequest{
		Name:   "test-export",
		Format: "mp4",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateExport_MissingName(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := CreateExportRequest{
		ExecutionID: uuid.New().String(),
		Format:      "mp4",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateExport_MissingFormat(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := CreateExportRequest{
		ExecutionID: uuid.New().String(),
		Name:        "test-export",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateExport_InvalidFormat(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := CreateExportRequest{
		ExecutionID: uuid.New().String(),
		Name:        "test-export",
		Format:      "invalid-format",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid format, got %d", rr.Code)
	}
}

func TestCreateExport_ExecutionNotFound(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	execSvc.GetExecutionError = database.ErrNotFound

	body := CreateExportRequest{
		ExecutionID: uuid.New().String(),
		Name:        "test-export",
		Format:      "mp4",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestCreateExport_AllFormats(t *testing.T) {
	formats := []string{"mp4", "gif", "json", "html"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			handler, _, execSvc, _, _, _ := createTestHandler()

			executionID := uuid.New()
			execSvc.AddExecution(&database.ExecutionIndex{
				ID:     executionID,
				Status: "completed",
			})

			body := CreateExportRequest{
				ExecutionID: executionID.String(),
				Name:        "test-" + format,
				Format:      format,
			}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.CreateExport(rr, req)

			if rr.Code != http.StatusCreated {
				t.Fatalf("expected status 201 for format %s, got %d: %s", format, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestCreateExport_WithStatus(t *testing.T) {
	handler, _, execSvc, repo, _, _ := createTestHandler()

	executionID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:     executionID,
		Status: "completed",
	})

	body := CreateExportRequest{
		ExecutionID: executionID.String(),
		Name:        "pending-export",
		Format:      "mp4",
		Status:      "pending",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify status was set
	exports, _ := repo.ListExports(context.Background(), 10, 0)
	if len(exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(exports))
	}
	if exports[0].Status != "pending" {
		t.Fatalf("expected status 'pending', got %s", exports[0].Status)
	}
}

func TestCreateExport_InvalidStatus(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	executionID := uuid.New()
	execSvc.AddExecution(&database.ExecutionIndex{
		ID:     executionID,
		Status: "completed",
	})

	body := CreateExportRequest{
		ExecutionID: executionID.String(),
		Name:        "test-export",
		Format:      "mp4",
		Status:      "invalid-status",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid status, got %d", rr.Code)
	}
}

// ============================================================================
// UpdateExport Tests
// ============================================================================

func TestUpdateExport_Success(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	executionID := uuid.New()
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, StartedAt: time.Now()})

	export := &database.ExportIndex{
		ExecutionID: executionID,
		Name:        "original-name",
		Format:      "mp4",
		Status:      "pending",
	}
	repo.AddExport(export)

	body := UpdateExportRequest{
		Name:   "updated-name",
		Status: "completed",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/exports/"+export.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", export.ID.String())
	rr := httptest.NewRecorder()

	handler.UpdateExport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify updates
	updated, _ := repo.GetExport(context.Background(), export.ID)
	if updated.Name != "updated-name" {
		t.Fatalf("expected name 'updated-name', got %s", updated.Name)
	}
	if updated.Status != "completed" {
		t.Fatalf("expected status 'completed', got %s", updated.Status)
	}
}

func TestUpdateExport_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	exportID := uuid.New()
	body := UpdateExportRequest{Name: "new-name"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/exports/"+exportID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", exportID.String())
	rr := httptest.NewRecorder()

	handler.UpdateExport(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestUpdateExport_InvalidStatus(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	executionID := uuid.New()
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, StartedAt: time.Now()})

	export := &database.ExportIndex{
		ExecutionID: executionID,
		Name:        "test-export",
		Format:      "mp4",
		Status:      "pending",
	}
	repo.AddExport(export)

	body := UpdateExportRequest{Status: "invalid-status"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/exports/"+export.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", export.ID.String())
	rr := httptest.NewRecorder()

	handler.UpdateExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid status, got %d", rr.Code)
	}
}

func TestUpdateExport_AllFields(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	executionID := uuid.New()
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, StartedAt: time.Now()})

	export := &database.ExportIndex{
		ExecutionID: executionID,
		Name:        "original",
		Format:      "mp4",
		Status:      "pending",
	}
	repo.AddExport(export)

	fileSize := int64(1024000)
	durationMs := 30000
	frameCount := 900
	body := UpdateExportRequest{
		Name:          "updated",
		StorageURL:    "https://storage.example.com/exports/video.mp4",
		ThumbnailURL:  "https://storage.example.com/exports/thumb.jpg",
		FileSizeBytes: &fileSize,
		DurationMs:    &durationMs,
		FrameCount:    &frameCount,
		AICaption:     "A browser automation workflow in action",
		Status:        "completed",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/exports/"+export.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", export.ID.String())
	rr := httptest.NewRecorder()

	handler.UpdateExport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	updated, _ := repo.GetExport(context.Background(), export.ID)
	if updated.Name != "updated" {
		t.Fatalf("expected name 'updated', got %s", updated.Name)
	}
	if updated.StorageURL != "https://storage.example.com/exports/video.mp4" {
		t.Fatalf("expected storage URL, got %s", updated.StorageURL)
	}
	if *updated.FileSizeBytes != 1024000 {
		t.Fatalf("expected file size 1024000, got %d", *updated.FileSizeBytes)
	}
	if updated.AICaption != "A browser automation workflow in action" {
		t.Fatalf("expected AI caption, got %s", updated.AICaption)
	}
}

// ============================================================================
// DeleteExport Tests
// ============================================================================

func TestDeleteExport_Success(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	executionID := uuid.New()
	repo.AddExecution(&database.ExecutionIndex{ID: executionID, StartedAt: time.Now()})

	export := &database.ExportIndex{
		ExecutionID: executionID,
		Name:        "to-delete",
		Format:      "mp4",
		Status:      "completed",
	}
	repo.AddExport(export)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/exports/"+export.ID.String(), nil)
	req = withURLParam(req, "id", export.ID.String())
	rr := httptest.NewRecorder()

	handler.DeleteExport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify deleted
	_, err := repo.GetExport(context.Background(), export.ID)
	if !errors.Is(err, database.ErrNotFound) {
		t.Fatal("expected export to be deleted")
	}
}

func TestDeleteExport_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	exportID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/exports/"+exportID.String(), nil)
	req = withURLParam(req, "id", exportID.String())
	rr := httptest.NewRecorder()

	handler.DeleteExport(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestDeleteExport_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/exports/invalid-uuid", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.DeleteExport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// GenerateExportCaption Tests
// ============================================================================

func TestGenerateExportCaption_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	exportID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports/"+exportID.String()+"/generate-caption", nil)
	req = withURLParam(req, "id", exportID.String())
	rr := httptest.NewRecorder()

	handler.GenerateExportCaption(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGenerateExportCaption_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports/invalid-uuid/generate-caption", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GenerateExportCaption(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// Note: Full GenerateExportCaption_Success test requires AI service mock
// which would need additional infrastructure. The handler gracefully falls back
// to a default caption when AI is unavailable, so basic functionality is covered.
