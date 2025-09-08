package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type TaskStatusChange struct {
	TaskID     uuid.UUID `json:"task_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	ChangedBy  string    `json:"changed_by,omitempty"`
	ChangedAt  time.Time `json:"changed_at"`
}

type TaskMonitoringResult struct {
	Success         bool               `json:"success"`
	TaskID         uuid.UUID          `json:"task_id"`
	StatusChanged  bool               `json:"status_changed"`
	PreviousStatus string             `json:"previous_status"`
	NewStatus      string             `json:"new_status"`
	StatusHistory  []TaskStatusChange `json:"status_history"`
	NextActions    []string           `json:"next_actions"`
	Error          string             `json:"error,omitempty"`
}

func (s *TaskPlannerService) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON payload", http.StatusBadRequest, err)
		return
	}

	if req.TaskID == "" || req.ToStatus == "" {
		HTTPError(w, "task_id and to_status are required", http.StatusBadRequest, nil)
		return
	}

	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		HTTPError(w, "Invalid task ID format", http.StatusBadRequest, err)
		return
	}

	// Validate status transition
	validStatuses := map[string]bool{
		"backlog":     true,
		"in_progress": true,
		"blocked":     true,
		"review":      true,
		"completed":   true,
		"cancelled":   true,
	}

	if !validStatuses[req.ToStatus] {
		HTTPError(w, "Invalid status. Valid statuses: backlog, in_progress, blocked, review, completed, cancelled", http.StatusBadRequest, nil)
		return
	}

	// Get current task
	task, err := s.getTaskByID(taskID)
	if err != nil {
		HTTPError(w, "Task not found", http.StatusNotFound, err)
		return
	}

	if task.Status == req.ToStatus {
		HTTPError(w, "Task is already in the requested status", http.StatusBadRequest, nil)
		return
	}

	// Validate status transition rules
	if err := s.validateStatusTransition(task.Status, req.ToStatus); err != nil {
		HTTPError(w, err.Error(), http.StatusBadRequest, err)
		return
	}

	// Perform the status update
	ctx := context.Background()
	result, err := s.performStatusUpdate(ctx, task, req)
	if err != nil {
		HTTPError(w, "Failed to update task status", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *TaskPlannerService) GetTaskStatusHistory(w http.ResponseWriter, r *http.Request) {
	taskIDStr := r.URL.Query().Get("task_id")
	if taskIDStr == "" {
		HTTPError(w, "task_id parameter is required", http.StatusBadRequest, nil)
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		HTTPError(w, "Invalid task ID format", http.StatusBadRequest, err)
		return
	}

	history, err := s.getTaskStatusHistory(taskID)
	if err != nil {
		HTTPError(w, "Failed to get task status history", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id": taskID,
		"history": history,
	})
}

func (s *TaskPlannerService) performStatusUpdate(ctx context.Context, task *Task, req TaskRequest) (*TaskMonitoringResult, error) {
	previousStatus := task.Status

	// Create status change record
	statusChange := TaskStatusChange{
		TaskID:     task.ID,
		FromStatus: previousStatus,
		ToStatus:   req.ToStatus,
		Reason:     req.Reason,
		Notes:      req.Notes,
		ChangedBy:  "api", // In real implementation, get from auth context
		ChangedAt:  time.Now(),
	}

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update task status
	query := `UPDATE tasks SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err = tx.Exec(query, req.ToStatus, task.ID)
	if err != nil {
		return nil, err
	}

	// Record status change in history
	err = s.recordStatusChange(tx, statusChange)
	if err != nil {
		return nil, err
	}

	// Handle status-specific logic
	err = s.handleStatusSpecificActions(ctx, tx, task, req.ToStatus, req)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Get updated status history
	history, err := s.getTaskStatusHistory(task.ID)
	if err != nil {
		s.logger.Warn("Failed to get status history", err)
		history = []TaskStatusChange{statusChange}
	}

	// Determine next actions
	nextActions := s.getNextActions(req.ToStatus, task)

	return &TaskMonitoringResult{
		Success:        true,
		TaskID:        task.ID,
		StatusChanged: true,
		PreviousStatus: previousStatus,
		NewStatus:     req.ToStatus,
		StatusHistory: history,
		NextActions:   nextActions,
	}, nil
}

func (s *TaskPlannerService) validateStatusTransition(fromStatus, toStatus string) error {
	// Define valid status transitions
	validTransitions := map[string][]string{
		"backlog":     {"in_progress", "cancelled"},
		"in_progress": {"blocked", "review", "completed", "backlog", "cancelled"},
		"blocked":     {"in_progress", "backlog", "cancelled"},
		"review":      {"in_progress", "completed", "backlog"},
		"completed":   {"review", "backlog"}, // Allow reopening
		"cancelled":   {"backlog"}, // Allow reactivation
	}

	allowedTransitions, exists := validTransitions[fromStatus]
	if !exists {
		return fmt.Errorf("invalid current status: %s", fromStatus)
	}

	for _, allowed := range allowedTransitions {
		if allowed == toStatus {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", fromStatus, toStatus)
}

func (s *TaskPlannerService) recordStatusChange(tx *sql.Tx, change TaskStatusChange) error {
	query := `
		INSERT INTO task_status_history (task_id, from_status, to_status, reason, notes, changed_by, changed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err := tx.Exec(query, change.TaskID, change.FromStatus, change.ToStatus,
		change.Reason, change.Notes, change.ChangedBy, change.ChangedAt)
	return err
}

func (s *TaskPlannerService) handleStatusSpecificActions(ctx context.Context, tx *sql.Tx, task *Task, newStatus string, req TaskRequest) error {
	switch newStatus {
	case "in_progress":
		// Set started_at timestamp
		query := `UPDATE tasks SET started_at = CURRENT_TIMESTAMP WHERE id = $1 AND started_at IS NULL`
		tx.Exec(query, task.ID)
		
		// Trigger research if not already done
		go s.ensureTaskResearched(ctx, task.ID)

	case "completed":
		// Set completed_at timestamp
		query := `UPDATE tasks SET completed_at = CURRENT_TIMESTAMP WHERE id = $1`
		tx.Exec(query, task.ID)
		
		// Update app statistics
		updateAppQuery := `UPDATE apps SET completed_tasks = completed_tasks + 1 WHERE id = $1`
		tx.Exec(updateAppQuery, task.AppID)
		
		// Auto-suggest related tasks
		go s.suggestRelatedTasks(ctx, task)

	case "blocked":
		// Ensure reason is provided for blocked status
		if req.Reason == "" {
			return fmt.Errorf("reason is required when marking task as blocked")
		}
		
		// Trigger investigation workflow
		go s.investigateBlockedTask(ctx, task.ID, req.Reason)

	case "cancelled":
		// Set cancelled_at timestamp
		query := `UPDATE tasks SET cancelled_at = CURRENT_TIMESTAMP WHERE id = $1`
		tx.Exec(query, task.ID)

	case "review":
		// Set review_requested_at timestamp
		query := `UPDATE tasks SET review_requested_at = CURRENT_TIMESTAMP WHERE id = $1`
		tx.Exec(query, task.ID)
	}

	return nil
}

func (s *TaskPlannerService) getTaskStatusHistory(taskID uuid.UUID) ([]TaskStatusChange, error) {
	query := `
		SELECT task_id, from_status, to_status, reason, notes, changed_by, changed_at
		FROM task_status_history
		WHERE task_id = $1
		ORDER BY changed_at DESC
		LIMIT 20`

	rows, err := s.db.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []TaskStatusChange
	for rows.Next() {
		var change TaskStatusChange
		err := rows.Scan(&change.TaskID, &change.FromStatus, &change.ToStatus,
			&change.Reason, &change.Notes, &change.ChangedBy, &change.ChangedAt)
		if err != nil {
			s.logger.Error("Failed to scan status change", err)
			continue
		}
		history = append(history, change)
	}

	return history, nil
}

func (s *TaskPlannerService) getNextActions(status string, task *Task) []string {
	switch status {
	case "backlog":
		return []string{
			"Research task requirements and dependencies",
			"Estimate implementation effort",
			"Begin implementation when ready",
		}
	case "in_progress":
		return []string{
			"Update progress regularly",
			"Document implementation notes",
			"Move to review when ready for validation",
		}
	case "blocked":
		return []string{
			"Identify and resolve blocking issues",
			"Update task with resolution steps",
			"Move back to in_progress when unblocked",
		}
	case "review":
		return []string{
			"Review implementation against requirements",
			"Test functionality thoroughly",
			"Complete task or return to development",
		}
	case "completed":
		return []string{
			"Document lessons learned",
			"Update related documentation",
			"Consider follow-up tasks",
		}
	case "cancelled":
		return []string{
			"Document cancellation reason",
			"Archive related resources",
			"Consider alternative approaches if needed",
		}
	default:
		return []string{}
	}
}

// Background functions
func (s *TaskPlannerService) ensureTaskResearched(ctx context.Context, taskID uuid.UUID) {
	// Check if task has research metadata
	var hasResearch bool
	query := `SELECT metadata ? 'research_completed_at' FROM tasks WHERE id = $1`
	err := s.db.QueryRow(query, taskID).Scan(&hasResearch)
	if err != nil || hasResearch {
		return // Already researched or error
	}

	// Trigger research
	task, err := s.getTaskByID(taskID)
	if err != nil {
		return
	}

	_, err = s.performTaskResearch(ctx, task)
	if err != nil {
		s.logger.Error("Failed to auto-research task", err)
	}
}

func (s *TaskPlannerService) suggestRelatedTasks(ctx context.Context, completedTask *Task) {
	// This would implement task suggestion logic based on the completed task
	// For now, just log that we should suggest related tasks
	s.logger.Info(fmt.Sprintf("Should suggest related tasks for completed task: %s", completedTask.Title))
	
	// In a full implementation, this might:
	// 1. Analyze the completed task's patterns
	// 2. Look for similar incomplete tasks
	// 3. Generate suggestions for follow-up work
	// 4. Update recommendations in the database
}

func (s *TaskPlannerService) investigateBlockedTask(ctx context.Context, taskID uuid.UUID, reason string) {
	// This would implement logic to investigate and potentially resolve blocked tasks
	s.logger.Info(fmt.Sprintf("Investigating blocked task %s: %s", taskID, reason))
	
	// In a full implementation, this might:
	// 1. Analyze the blocking reason
	// 2. Check for similar previously resolved blocks
	// 3. Generate suggested resolution steps
	// 4. Notify relevant team members
}