//go:build legacydb
// +build legacydb

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

// scheduleTestRepo is a mock repository for schedule tests
type scheduleTestRepo struct {
	mockRepository
	createScheduleFn        func(ctx context.Context, schedule *database.WorkflowSchedule) error
	getScheduleFn           func(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error)
	listSchedulesFn         func(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.WorkflowSchedule, error)
	updateScheduleFn        func(ctx context.Context, schedule *database.WorkflowSchedule) error
	deleteScheduleFn        func(ctx context.Context, id uuid.UUID) error
	updateScheduleLastRunFn func(ctx context.Context, id uuid.UUID, lastRun time.Time) error
}

func (r *scheduleTestRepo) CreateSchedule(ctx context.Context, schedule *database.WorkflowSchedule) error {
	if r.createScheduleFn != nil {
		return r.createScheduleFn(ctx, schedule)
	}
	return nil
}

func (r *scheduleTestRepo) GetSchedule(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
	if r.getScheduleFn != nil {
		return r.getScheduleFn(ctx, id)
	}
	return nil, database.ErrNotFound
}

func (r *scheduleTestRepo) ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.WorkflowSchedule, error) {
	if r.listSchedulesFn != nil {
		return r.listSchedulesFn(ctx, workflowID, activeOnly, limit, offset)
	}
	return nil, nil
}

func (r *scheduleTestRepo) UpdateSchedule(ctx context.Context, schedule *database.WorkflowSchedule) error {
	if r.updateScheduleFn != nil {
		return r.updateScheduleFn(ctx, schedule)
	}
	return nil
}

func (r *scheduleTestRepo) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if r.deleteScheduleFn != nil {
		return r.deleteScheduleFn(ctx, id)
	}
	return nil
}

func (r *scheduleTestRepo) UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
	if r.updateScheduleLastRunFn != nil {
		return r.updateScheduleLastRunFn(ctx, id, lastRun)
	}
	return nil
}

// scheduleTestWorkflowService is a mock workflow service for schedule tests
type scheduleTestWorkflowService struct {
	workflowServiceMock
	getWorkflowFn     func(ctx context.Context, id uuid.UUID) (*database.Workflow, error)
	executeWorkflowFn func(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
}

func (s *scheduleTestWorkflowService) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	if s.getWorkflowFn != nil {
		return s.getWorkflowFn(ctx, id)
	}
	return nil, database.ErrNotFound
}

func (s *scheduleTestWorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	if s.executeWorkflowFn != nil {
		return s.executeWorkflowFn(ctx, workflowID, parameters)
	}
	return nil, nil
}

func withScheduleRouteContext(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.RouteContext(r.Context())
	if ctx == nil {
		ctx = chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
	}
	for k, v := range params {
		ctx.URLParams.Add(k, v)
	}
	return r
}

func newScheduleTestHandler(repo *scheduleTestRepo, svc *scheduleTestWorkflowService) *Handler {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return &Handler{
		repo:             repo,
		workflowCatalog:  svc,
		executionService: svc,
		exportService:    svc,
		log:              logger,
	}
}

func TestCreateScheduleSuccess(t *testing.T) {
	workflowID := uuid.New()

	repo := &scheduleTestRepo{
		createScheduleFn: func(ctx context.Context, schedule *database.WorkflowSchedule) error {
			if schedule.WorkflowID != workflowID {
				t.Errorf("expected workflow_id %s, got %s", workflowID, schedule.WorkflowID)
			}
			if schedule.Name != "Daily Test" {
				t.Errorf("expected name 'Daily Test', got '%s'", schedule.Name)
			}
			if schedule.CronExpression != "0 9 * * *" {
				t.Errorf("expected cron '0 9 * * *', got '%s'", schedule.CronExpression)
			}
			if schedule.Timezone != "America/New_York" {
				t.Errorf("expected timezone 'America/New_York', got '%s'", schedule.Timezone)
			}
			// Assign ID to simulate DB behavior
			schedule.ID = uuid.New()
			return nil
		},
	}

	svc := &scheduleTestWorkflowService{
		getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
			return &database.Workflow{
				ID:   workflowID,
				Name: "Test Workflow",
			}, nil
		},
	}

	handler := newScheduleTestHandler(repo, svc)

	reqBody := `{
		"name": "Daily Test",
		"cron_expression": "0 9 * * *",
		"timezone": "America/New_York"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = withScheduleRouteContext(req, map[string]string{"workflowID": workflowID.String()})

	resp := httptest.NewRecorder()
	handler.CreateSchedule(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "created" {
		t.Errorf("expected status 'created', got '%v'", body["status"])
	}
}

func TestCreateScheduleMissingName(t *testing.T) {
	workflowID := uuid.New()
	repo := &scheduleTestRepo{}
	svc := &scheduleTestWorkflowService{}
	handler := newScheduleTestHandler(repo, svc)

	reqBody := `{
		"cron_expression": "0 9 * * *"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = withScheduleRouteContext(req, map[string]string{"workflowID": workflowID.String()})

	resp := httptest.NewRecorder()
	handler.CreateSchedule(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestCreateScheduleInvalidCron(t *testing.T) {
	workflowID := uuid.New()
	repo := &scheduleTestRepo{}
	svc := &scheduleTestWorkflowService{
		getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
			return &database.Workflow{ID: workflowID, Name: "Test"}, nil
		},
	}
	handler := newScheduleTestHandler(repo, svc)

	reqBody := `{
		"name": "Bad Schedule",
		"cron_expression": "not-valid"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = withScheduleRouteContext(req, map[string]string{"workflowID": workflowID.String()})

	resp := httptest.NewRecorder()
	handler.CreateSchedule(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateScheduleInvalidTimezone(t *testing.T) {
	workflowID := uuid.New()
	repo := &scheduleTestRepo{}
	svc := &scheduleTestWorkflowService{
		getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
			return &database.Workflow{ID: workflowID, Name: "Test"}, nil
		},
	}
	handler := newScheduleTestHandler(repo, svc)

	reqBody := `{
		"name": "Bad Timezone",
		"cron_expression": "0 9 * * *",
		"timezone": "Invalid/Timezone"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = withScheduleRouteContext(req, map[string]string{"workflowID": workflowID.String()})

	resp := httptest.NewRecorder()
	handler.CreateSchedule(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateScheduleWorkflowNotFound(t *testing.T) {
	workflowID := uuid.New()
	repo := &scheduleTestRepo{}
	svc := &scheduleTestWorkflowService{
		getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
			return nil, database.ErrNotFound
		},
	}
	handler := newScheduleTestHandler(repo, svc)

	reqBody := `{
		"name": "Test",
		"cron_expression": "0 9 * * *"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID.String()+"/schedules", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = withScheduleRouteContext(req, map[string]string{"workflowID": workflowID.String()})

	resp := httptest.NewRecorder()
	handler.CreateSchedule(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetScheduleSuccess(t *testing.T) {
	scheduleID := uuid.New()
	workflowID := uuid.New()

	repo := &scheduleTestRepo{
		getScheduleFn: func(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
			if id != scheduleID {
				t.Errorf("expected schedule_id %s, got %s", scheduleID, id)
			}
			return &database.WorkflowSchedule{
				ID:             scheduleID,
				WorkflowID:     workflowID,
				Name:           "Daily Test",
				CronExpression: "0 9 * * *",
				Timezone:       "UTC",
				IsActive:       true,
			}, nil
		},
	}

	svc := &scheduleTestWorkflowService{
		getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
			return &database.Workflow{ID: workflowID, Name: "Test Workflow"}, nil
		},
	}

	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/"+scheduleID.String(), nil)
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.GetSchedule(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetScheduleNotFound(t *testing.T) {
	scheduleID := uuid.New()
	repo := &scheduleTestRepo{
		getScheduleFn: func(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
			return nil, database.ErrNotFound
		},
	}
	svc := &scheduleTestWorkflowService{}
	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/"+scheduleID.String(), nil)
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.GetSchedule(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}
}

func TestListSchedulesSuccess(t *testing.T) {
	workflowID := uuid.New()
	schedule1 := uuid.New()
	schedule2 := uuid.New()

	repo := &scheduleTestRepo{
		listSchedulesFn: func(ctx context.Context, wfID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.WorkflowSchedule, error) {
			if wfID != nil && *wfID != workflowID {
				t.Errorf("expected workflow_id %s, got %s", workflowID, *wfID)
			}
			return []*database.WorkflowSchedule{
				{ID: schedule1, WorkflowID: workflowID, Name: "Schedule 1", IsActive: true},
				{ID: schedule2, WorkflowID: workflowID, Name: "Schedule 2", IsActive: false},
			}, nil
		},
	}
	svc := &scheduleTestWorkflowService{}
	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/schedules", nil)
	req = withScheduleRouteContext(req, map[string]string{"workflowID": workflowID.String()})

	resp := httptest.NewRecorder()
	handler.ListWorkflowSchedules(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	schedules, ok := body["schedules"].([]any)
	if !ok {
		t.Fatal("expected schedules array in response")
	}
	if len(schedules) != 2 {
		t.Errorf("expected 2 schedules, got %d", len(schedules))
	}
}

func TestUpdateScheduleSuccess(t *testing.T) {
	scheduleID := uuid.New()
	workflowID := uuid.New()

	repo := &scheduleTestRepo{
		getScheduleFn: func(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
			return &database.WorkflowSchedule{
				ID:             scheduleID,
				WorkflowID:     workflowID,
				Name:           "Old Name",
				CronExpression: "0 9 * * *",
				Timezone:       "UTC",
				IsActive:       true,
			}, nil
		},
		updateScheduleFn: func(ctx context.Context, schedule *database.WorkflowSchedule) error {
			if schedule.Name != "New Name" {
				t.Errorf("expected name 'New Name', got '%s'", schedule.Name)
			}
			return nil
		},
	}
	svc := &scheduleTestWorkflowService{
		getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
			return &database.Workflow{ID: workflowID, Name: "Test"}, nil
		},
	}
	handler := newScheduleTestHandler(repo, svc)

	reqBody := `{"name": "New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.UpdateSchedule(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteScheduleSuccess(t *testing.T) {
	scheduleID := uuid.New()
	repo := &scheduleTestRepo{
		deleteScheduleFn: func(ctx context.Context, id uuid.UUID) error {
			if id != scheduleID {
				t.Errorf("expected schedule_id %s, got %s", scheduleID, id)
			}
			return nil
		},
	}
	svc := &scheduleTestWorkflowService{}
	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/"+scheduleID.String(), nil)
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.DeleteSchedule(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteScheduleNotFound(t *testing.T) {
	scheduleID := uuid.New()
	repo := &scheduleTestRepo{
		deleteScheduleFn: func(ctx context.Context, id uuid.UUID) error {
			return database.ErrNotFound
		},
	}
	svc := &scheduleTestWorkflowService{}
	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/"+scheduleID.String(), nil)
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.DeleteSchedule(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}
}

func TestToggleScheduleSuccess(t *testing.T) {
	scheduleID := uuid.New()
	workflowID := uuid.New()

	initiallyActive := true
	repo := &scheduleTestRepo{
		getScheduleFn: func(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
			return &database.WorkflowSchedule{
				ID:             scheduleID,
				WorkflowID:     workflowID,
				Name:           "Test",
				CronExpression: "0 9 * * *",
				Timezone:       "UTC",
				IsActive:       initiallyActive,
			}, nil
		},
		updateScheduleFn: func(ctx context.Context, schedule *database.WorkflowSchedule) error {
			// After toggle, is_active should be false
			if schedule.IsActive != !initiallyActive {
				t.Errorf("expected is_active to be %v, got %v", !initiallyActive, schedule.IsActive)
			}
			return nil
		},
	}
	svc := &scheduleTestWorkflowService{
		getWorkflowFn: func(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
			return &database.Workflow{ID: workflowID, Name: "Test"}, nil
		},
	}
	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+scheduleID.String()+"/toggle", nil)
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.ToggleSchedule(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// After toggle, should be inactive (opposite of initial)
	if body["is_active"] != false {
		t.Errorf("expected is_active to be false, got %v", body["is_active"])
	}
}

func TestTriggerScheduleSuccess(t *testing.T) {
	scheduleID := uuid.New()
	workflowID := uuid.New()
	executionID := uuid.New()

	repo := &scheduleTestRepo{
		getScheduleFn: func(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
			return &database.WorkflowSchedule{
				ID:             scheduleID,
				WorkflowID:     workflowID,
				Name:           "Test Schedule",
				CronExpression: "0 9 * * *",
				Timezone:       "UTC",
				IsActive:       true,
			}, nil
		},
		updateScheduleLastRunFn: func(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
			return nil
		},
	}

	svc := &scheduleTestWorkflowService{
		executeWorkflowFn: func(ctx context.Context, wfID uuid.UUID, params map[string]any) (*database.Execution, error) {
			if wfID != workflowID {
				t.Errorf("expected workflow_id %s, got %s", workflowID, wfID)
			}
			// Verify trigger metadata is passed
			if params["_trigger_type"] != "manual_schedule_trigger" {
				t.Errorf("expected _trigger_type 'manual_schedule_trigger', got '%v'", params["_trigger_type"])
			}
			return &database.Execution{
				ID:         executionID,
				WorkflowID: workflowID,
				Status:     "pending",
			}, nil
		},
	}
	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+scheduleID.String()+"/trigger", nil)
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.TriggerSchedule(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["execution_id"] != executionID.String() {
		t.Errorf("expected execution_id %s, got %v", executionID, body["execution_id"])
	}
	if body["status"] != "triggered" {
		t.Errorf("expected status 'triggered', got '%v'", body["status"])
	}
}

func TestTriggerScheduleNotFound(t *testing.T) {
	scheduleID := uuid.New()
	repo := &scheduleTestRepo{
		getScheduleFn: func(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
			return nil, database.ErrNotFound
		},
	}
	svc := &scheduleTestWorkflowService{}
	handler := newScheduleTestHandler(repo, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+scheduleID.String()+"/trigger", nil)
	req = withScheduleRouteContext(req, map[string]string{"scheduleID": scheduleID.String()})

	resp := httptest.NewRecorder()
	handler.TriggerSchedule(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}
}

// Test helper function validation
func TestValidateCronExpression(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"valid 5-field cron", "0 9 * * *", false},
		{"valid 6-field cron", "0 0 9 * * *", false},
		{"empty cron", "", true},
		{"invalid - too few fields", "* *", true},
		{"invalid - too many fields", "* * * * * * *", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCronExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCronExpression(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

// Test formatRelativeTime function
func TestFormatRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		input    *time.Time
		contains string
	}{
		{"nil time", nil, ""},
		{"future time", ptrTime(time.Now().Add(2 * time.Hour)), "in"},
		{"past time", ptrTime(time.Now().Add(-2 * time.Hour)), "ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRelativeTime(tt.input)
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("formatRelativeTime() = %q, want containing %q", result, tt.contains)
			}
			if tt.contains == "" && result != "" {
				t.Errorf("formatRelativeTime() = %q, want empty string", result)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// Ensure interface compliance
var _ workflow.CatalogService = (*scheduleTestWorkflowService)(nil)
var _ workflow.ExecutionService = (*scheduleTestWorkflowService)(nil)
