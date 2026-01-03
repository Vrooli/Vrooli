package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/scheduler"
)

// CreateScheduleRequest represents the request to create a schedule
type CreateScheduleRequest struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	CronExpression string         `json:"cron_expression"`
	Timezone       string         `json:"timezone,omitempty"`
	Parameters     map[string]any `json:"parameters,omitempty"`
	IsActive       *bool          `json:"is_active,omitempty"`
}

// UpdateScheduleRequest represents the request to update a schedule
type UpdateScheduleRequest struct {
	Name           *string        `json:"name,omitempty"`
	Description    *string        `json:"description,omitempty"`
	CronExpression *string        `json:"cron_expression,omitempty"`
	Timezone       *string        `json:"timezone,omitempty"`
	Parameters     map[string]any `json:"parameters,omitempty"`
	IsActive       *bool          `json:"is_active,omitempty"`
}

// WorkflowScheduleResponse matches the UI contract while using ScheduleIndex as the stored index.
type WorkflowScheduleResponse struct {
	ID             uuid.UUID      `json:"id"`
	WorkflowID     uuid.UUID      `json:"workflow_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	CronExpression string         `json:"cron_expression"`
	Timezone       string         `json:"timezone"`
	IsActive       bool           `json:"is_active"`
	Parameters     map[string]any `json:"parameters,omitempty"`
	NextRunAt      *time.Time     `json:"next_run_at,omitempty"`
	LastRunAt      *time.Time     `json:"last_run_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at,omitempty"`
	WorkflowName   string         `json:"workflow_name,omitempty"`
	NextRunHuman   string         `json:"next_run_human,omitempty"`
	LastRunStatus  string         `json:"last_run_status,omitempty"`
}

func scheduleToResponse(schedule *database.ScheduleIndex, workflowName string) *WorkflowScheduleResponse {
	if schedule == nil {
		return nil
	}
	params, _ := schedule.GetParameters()
	return &WorkflowScheduleResponse{
		ID:             schedule.ID,
		WorkflowID:     schedule.WorkflowID,
		Name:           schedule.Name,
		Description:    "",
		CronExpression: schedule.CronExpression,
		Timezone:       schedule.Timezone,
		IsActive:       schedule.IsActive,
		Parameters:     params,
		NextRunAt:      schedule.NextRunAt,
		LastRunAt:      schedule.LastRunAt,
		CreatedAt:      schedule.CreatedAt,
		UpdatedAt:      schedule.UpdatedAt,
		WorkflowName:   workflowName,
		NextRunHuman:   formatRelativeTime(schedule.NextRunAt),
		LastRunStatus:  getLastRunStatus(schedule.LastRunAt),
	}
}

// CreateSchedule handles POST /api/v1/workflows/{workflowID}/schedules
func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "workflowID", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	var req CreateScheduleRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode create schedule request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	if strings.TrimSpace(req.CronExpression) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "cron_expression"}))
		return
	}

	// Validate field lengths
	if err := validateName(req.Name); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "name", "error": err.Error()}))
		return
	}
	if err := validateCronExpressionLength(req.CronExpression); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "cron_expression", "error": err.Error()}))
		return
	}
	if req.Timezone != "" {
		if err := validateTimezoneLength(req.Timezone); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "timezone", "error": err.Error()}))
			return
		}
	}

	// Validate cron expression
	if err := validateCronExpression(req.CronExpression); err != nil {
		h.respondError(w, ErrInvalidCronExpression.WithDetails(map[string]string{
			"cron_expression": req.CronExpression,
			"error":           err.Error(),
		}))
		return
	}

	// Validate and default timezone
	timezone := "UTC"
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			h.respondError(w, ErrInvalidTimezone.WithDetails(map[string]string{
				"timezone": req.Timezone,
				"error":    err.Error(),
			}))
			return
		}
		timezone = req.Timezone
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Verify workflow exists
	workflow, err := h.catalogService.GetWorkflow(ctx, workflowID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": workflowID.String()}))
			return
		}
		h.log.WithError(err).WithField("workflow_id", workflowID).Error("Failed to verify workflow")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_workflow"}))
		return
	}

	// Default is_active to true
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           strings.TrimSpace(req.Name),
		CronExpression: req.CronExpression,
		Timezone:       timezone,
		IsActive:       isActive,
	}
	_ = schedule.SetParameters(req.Parameters)

	// Calculate next run time
	nextRun, err := calculateNextRun(req.CronExpression, timezone)
	if err == nil && !nextRun.IsZero() {
		schedule.NextRunAt = &nextRun
	}

	if err := h.repo.CreateSchedule(ctx, schedule); err != nil {
		h.log.WithError(err).Error("Failed to create schedule")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "create_schedule"}))
		return
	}

	resp := scheduleToResponse(schedule, workflow.GetName())
	if resp != nil {
		resp.LastRunStatus = "never"
	}

	h.respondSuccess(w, http.StatusCreated, map[string]any{
		"schedule_id": schedule.ID,
		"status":      "created",
		"schedule":    resp,
	})
}

// ListWorkflowSchedules handles GET /api/v1/workflows/{workflowID}/schedules
func (h *Handler) ListWorkflowSchedules(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "workflowID", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, 0, 0)
	activeOnly := r.URL.Query().Get("active_only") == "true"

	schedules, err := h.repo.ListSchedules(ctx, &workflowID, activeOnly, limit, offset)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", workflowID).Error("Failed to list workflow schedules")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_schedules"}))
		return
	}

	workflowName := ""
	if wf, wfErr := h.catalogService.GetWorkflow(ctx, workflowID); wfErr == nil && wf != nil {
		workflowName = wf.GetName()
	}

	// Build response with computed fields
	responses := make([]*WorkflowScheduleResponse, len(schedules))
	for i, s := range schedules {
		responses[i] = scheduleToResponse(s, workflowName)
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"schedules": responses,
		"total":     len(responses),
	})
}

// ListAllSchedules handles GET /api/v1/schedules
func (h *Handler) ListAllSchedules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, 0, 0)
	activeOnly := r.URL.Query().Get("active_only") == "true"

	// Check for workflow_id filter in query params
	var workflowID *uuid.UUID
	if wfIDStr := strings.TrimSpace(r.URL.Query().Get("workflow_id")); wfIDStr != "" {
		id, err := uuid.Parse(wfIDStr)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowID)
			return
		}
		workflowID = &id
	}

	schedules, err := h.repo.ListSchedules(ctx, workflowID, activeOnly, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list schedules")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_schedules"}))
		return
	}

	// Build response with computed fields
	responses := make([]*WorkflowScheduleResponse, len(schedules))
	for i, s := range schedules {
		workflowName := ""
		if wf, wfErr := h.catalogService.GetWorkflow(ctx, s.WorkflowID); wfErr == nil && wf != nil {
			workflowName = wf.GetName()
		}
		responses[i] = scheduleToResponse(s, workflowName)
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"schedules": responses,
		"total":     len(responses),
	})
}

// GetSchedule handles GET /api/v1/schedules/{scheduleID}
func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, ok := h.parseUUIDParam(w, r, "scheduleID", ErrInvalidScheduleID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	schedule, err := h.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrScheduleNotFound)
			return
		}
		h.log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_schedule"}))
		return
	}

	// Get workflow name
	var workflowName string
	workflow, wfErr := h.catalogService.GetWorkflow(ctx, schedule.WorkflowID)
	if wfErr == nil && workflow != nil {
		workflowName = workflow.GetName()
	}

	resp := scheduleToResponse(schedule, workflowName)

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"schedule": resp,
	})
}

// UpdateSchedule handles PATCH /api/v1/schedules/{scheduleID}
func (h *Handler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, ok := h.parseUUIDParam(w, r, "scheduleID", ErrInvalidScheduleID)
	if !ok {
		return
	}

	var req UpdateScheduleRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode update schedule request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get existing schedule
	schedule, err := h.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrScheduleNotFound)
			return
		}
		h.log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule for update")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_schedule"}))
		return
	}

	// Validate field lengths if provided
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		if err := validateName(*req.Name); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "name", "error": err.Error()}))
			return
		}
	}
	if req.CronExpression != nil {
		if err := validateCronExpressionLength(*req.CronExpression); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "cron_expression", "error": err.Error()}))
			return
		}
	}
	if req.Timezone != nil {
		if err := validateTimezoneLength(*req.Timezone); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "timezone", "error": err.Error()}))
			return
		}
	}

	// Apply updates
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		schedule.Name = strings.TrimSpace(*req.Name)
	}
	if req.CronExpression != nil {
		if err := validateCronExpression(*req.CronExpression); err != nil {
			h.respondError(w, ErrInvalidCronExpression.WithDetails(map[string]string{
				"cron_expression": *req.CronExpression,
				"error":           err.Error(),
			}))
			return
		}
		schedule.CronExpression = *req.CronExpression

		// Recalculate next run time
		timezone := schedule.Timezone
		if req.Timezone != nil {
			timezone = *req.Timezone
		}
		nextRun, calcErr := calculateNextRun(*req.CronExpression, timezone)
		if calcErr == nil && !nextRun.IsZero() {
			schedule.NextRunAt = &nextRun
		}
	}
	if req.Timezone != nil {
		if _, err := time.LoadLocation(*req.Timezone); err != nil {
			h.respondError(w, ErrInvalidTimezone.WithDetails(map[string]string{
				"timezone": *req.Timezone,
				"error":    err.Error(),
			}))
			return
		}
		schedule.Timezone = *req.Timezone

		// Recalculate next run time with new timezone
		nextRun, calcErr := calculateNextRun(schedule.CronExpression, *req.Timezone)
		if calcErr == nil && !nextRun.IsZero() {
			schedule.NextRunAt = &nextRun
		}
	}
	if req.Parameters != nil {
		_ = schedule.SetParameters(req.Parameters)
	}
	if req.IsActive != nil {
		schedule.IsActive = *req.IsActive
	}

	if err := h.repo.UpdateSchedule(ctx, schedule); err != nil {
		h.log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to update schedule")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_schedule"}))
		return
	}

	// Get workflow name
	var workflowName string
	workflow, wfErr := h.catalogService.GetWorkflow(ctx, schedule.WorkflowID)
	if wfErr == nil && workflow != nil {
		workflowName = workflow.GetName()
	}

	resp := scheduleToResponse(schedule, workflowName)

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"schedule_id": schedule.ID,
		"status":      "updated",
		"schedule":    resp,
	})
}

// DeleteSchedule handles DELETE /api/v1/schedules/{scheduleID}
func (h *Handler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, ok := h.parseUUIDParam(w, r, "scheduleID", ErrInvalidScheduleID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.repo.DeleteSchedule(ctx, scheduleID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrScheduleNotFound)
			return
		}
		h.log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to delete schedule")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_schedule"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"schedule_id": scheduleID,
		"status":      "deleted",
	})
}

// ToggleSchedule handles POST /api/v1/schedules/{scheduleID}/toggle
func (h *Handler) ToggleSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, ok := h.parseUUIDParam(w, r, "scheduleID", ErrInvalidScheduleID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get existing schedule
	schedule, err := h.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrScheduleNotFound)
			return
		}
		h.log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule for toggle")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_schedule"}))
		return
	}

	// Toggle the active state
	schedule.IsActive = !schedule.IsActive

	// Update next_run_at based on new state
	if schedule.IsActive {
		nextRun, calcErr := calculateNextRun(schedule.CronExpression, schedule.Timezone)
		if calcErr == nil && !nextRun.IsZero() {
			schedule.NextRunAt = &nextRun
		}
	}

	if err := h.repo.UpdateSchedule(ctx, schedule); err != nil {
		h.log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to toggle schedule")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_schedule"}))
		return
	}

	// Get workflow name
	var workflowName string
	workflow, wfErr := h.catalogService.GetWorkflow(ctx, schedule.WorkflowID)
	if wfErr == nil && workflow != nil {
		workflowName = workflow.GetName()
	}

	resp := scheduleToResponse(schedule, workflowName)

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"schedule_id": schedule.ID,
		"is_active":   schedule.IsActive,
		"schedule":    resp,
	})
}

// TriggerSchedule handles POST /api/v1/schedules/{scheduleID}/trigger
// This manually triggers a scheduled workflow execution immediately
func (h *Handler) TriggerSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, ok := h.parseUUIDParam(w, r, "scheduleID", ErrInvalidScheduleID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExecutionCompletionTimeout)
	defer cancel()

	// Get the schedule
	schedule, err := h.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrScheduleNotFound)
			return
		}
		h.log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule for trigger")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_schedule"}))
		return
	}

	// Execute the workflow with schedule parameters
	params := make(map[string]any)
	if parsed, err := schedule.GetParameters(); err == nil {
		for k, v := range parsed {
			params[k] = v
		}
	}
	// Add metadata about the trigger
	params["_trigger_type"] = "manual_schedule_trigger"
	params["_schedule_id"] = scheduleID.String()
	params["_schedule_name"] = schedule.Name

	execution, err := h.executionService.ExecuteWorkflow(ctx, schedule.WorkflowID, params)
	if err != nil {
		h.log.WithError(err).WithFields(map[string]any{
			"schedule_id": scheduleID,
			"workflow_id": schedule.WorkflowID,
		}).Error("Failed to trigger scheduled workflow")
		h.respondError(w, ErrWorkflowExecutionFailed.WithDetails(map[string]string{
			"schedule_id": scheduleID.String(),
			"workflow_id": schedule.WorkflowID.String(),
			"error":       err.Error(),
		}))
		return
	}

	// Update last_run_at
	now := time.Now()
	if updateErr := h.repo.UpdateScheduleLastRun(ctx, scheduleID, now); updateErr != nil {
		h.log.WithError(updateErr).WithField("schedule_id", scheduleID).Warn("Failed to update schedule last_run_at after trigger")
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"schedule_id":  scheduleID,
		"execution_id": execution.ID,
		"workflow_id":  schedule.WorkflowID,
		"status":       "triggered",
		"triggered_at": now,
	})
}

// Helper functions

// validateCronExpression validates a cron expression string.
func validateCronExpression(expr string) error {
	return scheduler.ValidateCronExpression(expr)
}

// calculateNextRun calculates the next run time for a cron expression.
func calculateNextRun(cronExpr string, timezone string) (time.Time, error) {
	return scheduler.CalculateNextRun(cronExpr, timezone)
}

// formatRelativeTime formats a time as a human-readable relative string.
func formatRelativeTime(t *time.Time) string {
	if t == nil {
		return ""
	}

	now := time.Now()
	diff := t.Sub(now)

	if diff < 0 {
		// Past time
		diff = -diff
		if diff < time.Minute {
			return "just now"
		}
		if diff < time.Hour {
			mins := int(diff.Minutes())
			if mins == 1 {
				return "1 minute ago"
			}
			return formatDurationHuman(mins, "minute") + " ago"
		}
		if diff < 24*time.Hour {
			hours := int(diff.Hours())
			if hours == 1 {
				return "1 hour ago"
			}
			return formatDurationHuman(hours, "hour") + " ago"
		}
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return formatDurationHuman(days, "day") + " ago"
	}

	// Future time
	if diff < time.Minute {
		return "in less than a minute"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "in 1 minute"
		}
		return "in " + formatDurationHuman(mins, "minute")
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "in 1 hour"
		}
		return "in " + formatDurationHuman(hours, "hour")
	}
	days := int(diff.Hours() / 24)
	if days == 1 {
		return "in 1 day"
	}
	return "in " + formatDurationHuman(days, "day")
}

// formatDurationHuman formats a count and unit into a human-readable string.
func formatDurationHuman(count int, unit string) string {
	if count == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(count) + " " + unit + "s"
}

// getLastRunStatus returns a string describing the last run status.
func getLastRunStatus(lastRun *time.Time) string {
	if lastRun == nil {
		return "never"
	}
	// For now just return "success" - actual status tracking will come with scheduler service
	return "success"
}

// ScheduleOccurrence represents a single projected run of a schedule
type ScheduleOccurrence struct {
	ScheduleID     uuid.UUID `json:"schedule_id"`
	ScheduleName   string    `json:"schedule_name"`
	WorkflowID     uuid.UUID `json:"workflow_id"`
	WorkflowName   string    `json:"workflow_name"`
	RunAt          time.Time `json:"run_at"`
	IsActive       bool      `json:"is_active"`
	CronExpression string    `json:"cron_expression"`
	Timezone       string    `json:"timezone"`
}

// ScheduleAggregate represents aggregated info for high-frequency schedules
type ScheduleAggregate struct {
	ScheduleID     uuid.UUID `json:"schedule_id"`
	ScheduleName   string    `json:"schedule_name"`
	TotalRuns      int       `json:"total_runs"`
	Truncated      bool      `json:"truncated"`
	CronExpression string    `json:"cron_expression"`
}

// GetScheduleOccurrences handles GET /api/v1/schedules/occurrences
// Returns projected schedule runs for a date range, with aggregation for high-frequency schedules
func (h *Handler) GetScheduleOccurrences(w http.ResponseWriter, r *http.Request) {
	// Parse required query params
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "start and end query parameters are required",
		}))
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "start",
			"error": "must be valid RFC3339 timestamp",
		}))
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "end",
			"error": "must be valid RFC3339 timestamp",
		}))
		return
	}

	if end.Before(start) {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "end must be after start",
		}))
		return
	}

	// Cap the range to prevent excessive computation (max 1 year)
	maxRange := 365 * 24 * time.Hour
	if end.Sub(start) > maxRange {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "date range cannot exceed 1 year",
		}))
		return
	}

	// Parse optional params
	maxPerSchedule := 100
	if maxStr := r.URL.Query().Get("max_per_schedule"); maxStr != "" {
		if parsed, err := strconv.Atoi(maxStr); err == nil && parsed > 0 && parsed <= 1000 {
			maxPerSchedule = parsed
		}
	}

	// Optional workflow filter
	var workflowIDFilter *uuid.UUID
	if wfIDStr := strings.TrimSpace(r.URL.Query().Get("workflow_id")); wfIDStr != "" {
		id, err := uuid.Parse(wfIDStr)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowID)
			return
		}
		workflowIDFilter = &id
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get all active schedules
	schedules, err := h.repo.ListSchedules(ctx, workflowIDFilter, true, 0, 0)
	if err != nil {
		h.log.WithError(err).Error("Failed to list schedules for occurrences")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_schedules"}))
		return
	}

	// Cache workflow names
	workflowNames := make(map[uuid.UUID]string)
	for _, s := range schedules {
		if _, ok := workflowNames[s.WorkflowID]; !ok {
			if wf, wfErr := h.catalogService.GetWorkflow(ctx, s.WorkflowID); wfErr == nil && wf != nil {
				workflowNames[s.WorkflowID] = wf.GetName()
			}
		}
	}

	// Calculate occurrences for each schedule
	var occurrences []ScheduleOccurrence
	aggregates := make(map[string]ScheduleAggregate)

	for _, s := range schedules {
		// Calculate runs within the date range
		runs, err := calculateOccurrencesInRange(s.CronExpression, s.Timezone, start, end, maxPerSchedule+1)
		if err != nil {
			h.log.WithError(err).WithField("schedule_id", s.ID).Warn("Failed to calculate occurrences for schedule")
			continue
		}

		workflowName := workflowNames[s.WorkflowID]

		// Check if this is a high-frequency schedule (more than max per schedule)
		if len(runs) > maxPerSchedule {
			// Count total runs more accurately for the aggregate
			totalRuns := len(runs)
			// For very high frequency schedules, estimate the total
			if totalRuns > maxPerSchedule {
				// Use remaining runs to estimate
				totalRuns = estimateTotalRuns(s.CronExpression, start, end)
			}

			aggregates[s.ID.String()] = ScheduleAggregate{
				ScheduleID:     s.ID,
				ScheduleName:   s.Name,
				TotalRuns:      totalRuns,
				Truncated:      true,
				CronExpression: s.CronExpression,
			}
			// Still include first few occurrences for week/day view
			runs = runs[:min(maxPerSchedule, len(runs))]
		}

		for _, runAt := range runs {
			occurrences = append(occurrences, ScheduleOccurrence{
				ScheduleID:     s.ID,
				ScheduleName:   s.Name,
				WorkflowID:     s.WorkflowID,
				WorkflowName:   workflowName,
				RunAt:          runAt,
				IsActive:       s.IsActive,
				CronExpression: s.CronExpression,
				Timezone:       s.Timezone,
			})
		}
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"occurrences": occurrences,
		"aggregates":  aggregates,
		"total":       len(occurrences),
		"range": map[string]string{
			"start": start.Format(time.RFC3339),
			"end":   end.Format(time.RFC3339),
		},
	})
}

// calculateOccurrencesInRange calculates schedule occurrences within a date range
func calculateOccurrencesInRange(cronExpr, timezone string, start, end time.Time, maxRuns int) ([]time.Time, error) {
	schedule, err := scheduler.ParseCronExpression(cronExpr)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	var runs []time.Time
	current := start.In(loc)

	for len(runs) < maxRuns {
		next := schedule.Next(current)
		if next.After(end) {
			break
		}
		runs = append(runs, next.UTC())
		current = next
	}

	return runs, nil
}

// estimateTotalRuns estimates the total number of runs in a date range
// for high-frequency schedules where calculating each one is too expensive
func estimateTotalRuns(cronExpr string, start, end time.Time) int {
	// Estimate based on common cron patterns
	duration := end.Sub(start)

	// Check for common patterns
	fields := strings.Fields(strings.TrimSpace(cronExpr))
	if len(fields) >= 5 {
		minute := fields[0]
		hour := fields[1]

		// Every minute
		if minute == "*" && hour == "*" {
			return int(duration.Minutes())
		}
		// Every N minutes
		if strings.HasPrefix(minute, "*/") && hour == "*" {
			if n, err := strconv.Atoi(strings.TrimPrefix(minute, "*/")); err == nil && n > 0 {
				return int(duration.Minutes()) / n
			}
		}
		// Every hour
		if minute == "0" && hour == "*" {
			return int(duration.Hours())
		}
		// Every N hours
		if minute == "0" && strings.HasPrefix(hour, "*/") {
			if n, err := strconv.Atoi(strings.TrimPrefix(hour, "*/")); err == nil && n > 0 {
				return int(duration.Hours()) / n
			}
		}
	}

	// Default: assume daily
	return int(duration.Hours() / 24)
}
