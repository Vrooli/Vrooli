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
// CreateSchedule Tests
// ============================================================================

func TestCreateSchedule_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	// Create a workflow to schedule
	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{
		ID:   workflowID,
		Name: "test-workflow",
	})

	body := CreateScheduleRequest{
		Name:           "daily-test",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.CreateSchedule(rr, req)

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
	if response["schedule_id"] == nil {
		t.Fatal("expected schedule_id in response")
	}

	// Verify schedule was created in repo
	schedules, _ := repo.ListSchedules(context.Background(), &workflowID, false, 10, 0)
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule in repo, got %d", len(schedules))
	}
}

func TestCreateSchedule_InvalidWorkflowID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := CreateScheduleRequest{
		Name:           "test-schedule",
		CronExpression: "0 0 * * *",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/invalid-uuid/schedules", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "workflowID", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.CreateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateSchedule_MissingName(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test"})

	body := CreateScheduleRequest{
		CronExpression: "0 0 * * *",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.CreateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing name, got %d", rr.Code)
	}
}

func TestCreateSchedule_MissingCronExpression(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test"})

	body := CreateScheduleRequest{
		Name: "test-schedule",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.CreateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing cron expression, got %d", rr.Code)
	}
}

func TestCreateSchedule_WorkflowNotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()

	body := CreateScheduleRequest{
		Name:           "test-schedule",
		CronExpression: "0 0 * * *",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.CreateSchedule(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for workflow not found, got %d", rr.Code)
	}
}

func TestCreateSchedule_InvalidTimezone(t *testing.T) {
	handler, catalogSvc, _, _, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test"})

	body := CreateScheduleRequest{
		Name:           "test-schedule",
		CronExpression: "0 0 * * *",
		Timezone:       "Invalid/Timezone",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.CreateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid timezone, got %d", rr.Code)
	}
}

func TestCreateSchedule_WithParameters(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	isActive := false
	body := CreateScheduleRequest{
		Name:           "parameterized-schedule",
		CronExpression: "*/5 * * * *",
		Timezone:       "America/New_York",
		Parameters:     map[string]any{"url": "https://example.com", "timeout": 30},
		IsActive:       &isActive,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.CreateSchedule(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify parameters were stored
	schedules, _ := repo.ListSchedules(context.Background(), &workflowID, false, 10, 0)
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}

	if schedules[0].IsActive {
		t.Fatal("expected schedule to be inactive")
	}

	params, _ := schedules[0].GetParameters()
	if params["url"] != "https://example.com" {
		t.Fatalf("expected url parameter, got %v", params)
	}
}

// ============================================================================
// ListWorkflowSchedules Tests
// ============================================================================

func TestListWorkflowSchedules_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	// Add schedules
	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "schedule-1",
		CronExpression: "0 0 * * *",
		IsActive:       true,
	})
	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "schedule-2",
		CronExpression: "0 12 * * *",
		IsActive:       false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/schedules", nil)
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ListWorkflowSchedules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(response["total"].(float64))
	if total != 2 {
		t.Fatalf("expected 2 schedules, got %d", total)
	}
}

func TestListWorkflowSchedules_ActiveOnly(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "active-schedule",
		CronExpression: "0 0 * * *",
		IsActive:       true,
	})
	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "inactive-schedule",
		CronExpression: "0 12 * * *",
		IsActive:       false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/schedules?active_only=true", nil)
	req = withURLParam(req, "workflowID", workflowID.String())
	rr := httptest.NewRecorder()

	handler.ListWorkflowSchedules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(response["total"].(float64))
	if total != 1 {
		t.Fatalf("expected 1 active schedule, got %d", total)
	}
}

func TestListWorkflowSchedules_InvalidWorkflowID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/invalid-uuid/schedules", nil)
	req = withURLParam(req, "workflowID", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ListWorkflowSchedules(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// ListAllSchedules Tests
// ============================================================================

func TestListAllSchedules_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID1 := uuid.New()
	workflowID2 := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID1, Name: "workflow-1"})
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID2, Name: "workflow-2"})

	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     workflowID1,
		Name:           "schedule-1",
		CronExpression: "0 0 * * *",
		IsActive:       true,
	})
	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     workflowID2,
		Name:           "schedule-2",
		CronExpression: "0 12 * * *",
		IsActive:       true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	rr := httptest.NewRecorder()

	handler.ListAllSchedules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(response["total"].(float64))
	if total != 2 {
		t.Fatalf("expected 2 schedules, got %d", total)
	}
}

func TestListAllSchedules_WithWorkflowFilter(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "schedule-1",
		CronExpression: "0 0 * * *",
		IsActive:       true,
	})
	repo.AddSchedule(&database.ScheduleIndex{
		WorkflowID:     uuid.New(),
		Name:           "schedule-2",
		CronExpression: "0 12 * * *",
		IsActive:       true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules?workflow_id="+workflowID.String(), nil)
	rr := httptest.NewRecorder()

	handler.ListAllSchedules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(response["total"].(float64))
	if total != 1 {
		t.Fatalf("expected 1 schedule for workflow, got %d", total)
	}
}

// ============================================================================
// GetSchedule Tests
// ============================================================================

func TestGetSchedule_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "test-schedule",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/"+schedule.ID.String(), nil)
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.GetSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	scheduleData := response["schedule"].(map[string]any)
	if scheduleData["name"] != "test-schedule" {
		t.Fatalf("expected name 'test-schedule', got %v", scheduleData["name"])
	}
}

func TestGetSchedule_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	scheduleID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/"+scheduleID.String(), nil)
	req = withURLParam(req, "scheduleID", scheduleID.String())
	rr := httptest.NewRecorder()

	handler.GetSchedule(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetSchedule_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/invalid-uuid", nil)
	req = withURLParam(req, "scheduleID", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// UpdateSchedule Tests
// ============================================================================

func TestUpdateSchedule_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "original-name",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	newName := "updated-name"
	isActive := false
	body := UpdateScheduleRequest{
		Name:     &newName,
		IsActive: &isActive,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/"+schedule.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.UpdateSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify updates
	updated, _ := repo.GetSchedule(context.Background(), schedule.ID)
	if updated.Name != "updated-name" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}
	if updated.IsActive {
		t.Fatal("expected schedule to be inactive")
	}
}

func TestUpdateSchedule_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	scheduleID := uuid.New()
	newName := "new-name"
	body := UpdateScheduleRequest{Name: &newName}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/"+scheduleID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "scheduleID", scheduleID.String())
	rr := httptest.NewRecorder()

	handler.UpdateSchedule(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestUpdateSchedule_InvalidCronExpression(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "test-schedule",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	invalidCron := "invalid cron"
	body := UpdateScheduleRequest{CronExpression: &invalidCron}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/"+schedule.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.UpdateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid cron, got %d", rr.Code)
	}
}

func TestUpdateSchedule_InvalidTimezone(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "test-schedule",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	invalidTz := "Invalid/Zone"
	body := UpdateScheduleRequest{Timezone: &invalidTz}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/"+schedule.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.UpdateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid timezone, got %d", rr.Code)
	}
}

// ============================================================================
// DeleteSchedule Tests
// ============================================================================

func TestDeleteSchedule_Success(t *testing.T) {
	handler, _, _, repo, _, _ := createTestHandler()

	schedule := &database.ScheduleIndex{
		WorkflowID:     uuid.New(),
		Name:           "to-delete",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/"+schedule.ID.String(), nil)
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.DeleteSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify deleted
	_, err := repo.GetSchedule(context.Background(), schedule.ID)
	if !errors.Is(err, database.ErrNotFound) {
		t.Fatal("expected schedule to be deleted")
	}
}

func TestDeleteSchedule_InvalidUUID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/invalid-uuid", nil)
	req = withURLParam(req, "scheduleID", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.DeleteSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

// ============================================================================
// ToggleSchedule Tests
// ============================================================================

func TestToggleSchedule_Success(t *testing.T) {
	handler, catalogSvc, _, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "toggle-test",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedule.ID.String()+"/toggle", nil)
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.ToggleSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify toggled
	updated, _ := repo.GetSchedule(context.Background(), schedule.ID)
	if updated.IsActive {
		t.Fatal("expected schedule to be toggled to inactive")
	}

	// Toggle again
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedule.ID.String()+"/toggle", nil)
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr = httptest.NewRecorder()

	handler.ToggleSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	updated, _ = repo.GetSchedule(context.Background(), schedule.ID)
	if !updated.IsActive {
		t.Fatal("expected schedule to be toggled back to active")
	}
}

func TestToggleSchedule_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	scheduleID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+scheduleID.String()+"/toggle", nil)
	req = withURLParam(req, "scheduleID", scheduleID.String())
	rr := httptest.NewRecorder()

	handler.ToggleSchedule(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

// ============================================================================
// TriggerSchedule Tests
// ============================================================================

func TestTriggerSchedule_Success(t *testing.T) {
	handler, catalogSvc, execSvc, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "trigger-test",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	// Add an execution that will be returned by ExecuteWorkflow
	execSvc.AddExecution(&database.ExecutionIndex{
		WorkflowID: workflowID,
		Status:     "running",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedule.ID.String()+"/trigger", nil)
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.TriggerSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["status"] != "triggered" {
		t.Fatalf("expected status 'triggered', got %v", response["status"])
	}
	if response["execution_id"] == nil {
		t.Fatal("expected execution_id in response")
	}
}

func TestTriggerSchedule_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	scheduleID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+scheduleID.String()+"/trigger", nil)
	req = withURLParam(req, "scheduleID", scheduleID.String())
	rr := httptest.NewRecorder()

	handler.TriggerSchedule(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestTriggerSchedule_ExecutionFails(t *testing.T) {
	handler, catalogSvc, execSvc, repo, _, _ := createTestHandler()

	workflowID := uuid.New()
	catalogSvc.AddWorkflow(&database.WorkflowIndex{ID: workflowID, Name: "test-workflow"})

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           "trigger-fail-test",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	repo.AddSchedule(schedule)

	execSvc.ExecuteWorkflowError = errors.New("execution failed")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedule.ID.String()+"/trigger", nil)
	req = withURLParam(req, "scheduleID", schedule.ID.String())
	rr := httptest.NewRecorder()

	handler.TriggerSchedule(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestFormatRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		time     *time.Time
		contains string
	}{
		{
			name:     "nil time",
			time:     nil,
			contains: "",
		},
		{
			name:     "future time (hours)",
			time:     timePtr(time.Now().Add(2 * time.Hour)),
			contains: "in",
		},
		{
			name:     "past time (minutes)",
			time:     timePtr(time.Now().Add(-30 * time.Minute)),
			contains: "ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRelativeTime(tt.time)
			if tt.contains != "" && !containsStr(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
			if tt.contains == "" && result != "" {
				t.Errorf("expected empty string, got %q", result)
			}
		})
	}
}

func TestGetLastRunStatus(t *testing.T) {
	tests := []struct {
		name     string
		lastRun  *time.Time
		expected string
	}{
		{
			name:     "nil last run",
			lastRun:  nil,
			expected: "never",
		},
		{
			name:     "with last run",
			lastRun:  timePtr(time.Now().Add(-1 * time.Hour)),
			expected: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLastRunStatus(tt.lastRun)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// withURLParam helper is already defined in executions_test.go
// Reusing it here since it's in the same package

// createTestHandler helper is already defined in executions_test.go
// Reusing it here since it's in the same package
