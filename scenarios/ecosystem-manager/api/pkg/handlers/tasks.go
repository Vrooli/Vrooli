package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/slices"
	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
	"github.com/gorilla/mux"
)

// TaskHandlers contains handlers for task-related endpoints
type TaskHandlers struct {
	storage   *tasks.Storage
	assembler *prompts.Assembler
	processor *queue.Processor
	wsManager *websocket.Manager
}

func (h *TaskHandlers) handleMultiTargetCreate(w http.ResponseWriter, baseTask tasks.TaskItem) {
	created := make([]tasks.TaskItem, 0, len(baseTask.Targets))
	skipped := make([]map[string]string, 0)
	errors := make([]map[string]string, 0)
	baseTitle := baseTask.Title

	for _, target := range baseTask.Targets {
		existing, status, lookupErr := h.storage.FindActiveTargetTask(baseTask.Type, baseTask.Operation, target)
		if lookupErr != nil {
			errors = append(errors, map[string]string{
				"target": target,
				"error":  lookupErr.Error(),
			})
			continue
		}

		if existing != nil {
			skipped = append(skipped, map[string]string{
				"target": target,
				"reason": fmt.Sprintf("existing %s task %s in %s", baseTask.Operation, existing.ID, status),
			})
			continue
		}

		newTask := baseTask
		newTask.ID = generateTaskID(baseTask.Type, baseTask.Operation, target)
		newTask.Target = target
		newTask.Targets = []string{target}
		newTask.Title = deriveTaskTitle(baseTitle, baseTask.Operation, baseTask.Type, target)
		newTask.Status = "pending"
		newTask.Results = nil
		timestamp := timeutil.NowRFC3339()
		newTask.CreatedAt = timestamp
		newTask.UpdatedAt = timestamp

		if err := h.storage.SaveQueueItem(newTask, "pending"); err != nil {
			errors = append(errors, map[string]string{
				"target": target,
				"error":  err.Error(),
			})
			continue
		}

		created = append(created, newTask)
	}

	success := len(errors) == 0 && len(created) > 0
	response := map[string]any{
		"success": success,
		"created": created,
		"skipped": skipped,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := http.StatusConflict
	if len(created) > 0 {
		statusCode = http.StatusCreated
	} else if len(errors) > 0 {
		statusCode = http.StatusInternalServerError
	}

	writeJSON(w, response, statusCode)
}

var targetSlugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func generateTaskID(taskType, operation, target string) string {
	baseTimestamp := time.Now().Format("20060102-150405")
	if strings.TrimSpace(target) == "" {
		return fmt.Sprintf("%s-%s-%s", taskType, operation, baseTimestamp)
	}

	slug := sanitizeTargetSlug(target)
	return fmt.Sprintf("%s-%s-%s-%s", taskType, operation, slug, baseTimestamp)
}

func sanitizeTargetSlug(target string) string {
	slug := strings.ToLower(strings.TrimSpace(target))
	slug = targetSlugSanitizer.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "target"
	}
	if len(slug) > 48 {
		slug = slug[:48]
	}
	return slug
}

func deriveTaskTitle(baseTitle, operation, taskType, target string) string {
	trimmedBase := strings.TrimSpace(baseTitle)
	trimmedTarget := strings.TrimSpace(target)

	if trimmedTarget == "" {
		if trimmedBase != "" {
			return trimmedBase
		}
		return fmt.Sprintf("%s %s", operationDisplayName(operation), taskType)
	}

	if trimmedBase == "" {
		return fmt.Sprintf("%s %s %s", operationDisplayName(operation), taskType, trimmedTarget)
	}

	lowerBase := strings.ToLower(trimmedBase)
	lowerTarget := strings.ToLower(trimmedTarget)
	if strings.Contains(trimmedBase, "{{target}}") {
		return strings.ReplaceAll(trimmedBase, "{{target}}", trimmedTarget)
	}
	if strings.Contains(lowerBase, lowerTarget) {
		return trimmedBase
	}

	return fmt.Sprintf("%s (%s)", trimmedBase, trimmedTarget)
}

func operationDisplayName(operation string) string {
	switch operation {
	case "generator":
		return "Create"
	case "improver":
		return "Enhance"
	default:
		if operation == "" {
			return "Task"
		}
		return strings.ToUpper(operation[:1]) + operation[1:]
	}
}

// NewTaskHandlers creates a new task handlers instance
func NewTaskHandlers(storage *tasks.Storage, assembler *prompts.Assembler, processor *queue.Processor, wsManager *websocket.Manager) *TaskHandlers {
	return &TaskHandlers{
		storage:   storage,
		assembler: assembler,
		processor: processor,
		wsManager: wsManager,
	}
}

// preserveUnsetFields copies non-zero values from current to updated for fields that are unset.
// This helper consolidates field preservation logic used when updating tasks.
func preserveUnsetFields(updated, current *tasks.TaskItem) {
	if updated.Title == "" {
		updated.Title = current.Title
	}
	if updated.Priority == "" {
		updated.Priority = current.Priority
	}
	if updated.Category == "" {
		updated.Category = current.Category
	}
	if updated.Notes == "" {
		updated.Notes = current.Notes
	}
	if updated.EffortEstimate == "" {
		updated.EffortEstimate = current.EffortEstimate
	}
	if updated.CurrentPhase == "" && current.CurrentPhase != "" {
		updated.CurrentPhase = current.CurrentPhase
	}
	if updated.CompletionCount == 0 && current.CompletionCount > 0 {
		updated.CompletionCount = current.CompletionCount
	}
	if updated.LastCompletedAt == "" {
		updated.LastCompletedAt = current.LastCompletedAt
	}
	if len(updated.Targets) == 0 && len(current.Targets) > 0 {
		updated.Targets = current.Targets
		updated.Target = current.Target
	}
}

// isBackwardsTransition checks if a status change represents a backwards transition
// (moving from a terminal state back to an active state for re-execution)
func isBackwardsTransition(fromStatus, toStatus string) bool {
	terminalStatuses := map[string]bool{"completed": true, "failed": true}
	activeStatuses := map[string]bool{"pending": true, "in-progress": true}
	return terminalStatuses[fromStatus] && activeStatuses[toStatus]
}

// clearTaskExecutionData resets task fields for fresh execution after a backwards transition
func clearTaskExecutionData(task *tasks.TaskItem, taskID, fromStatus, toStatus string) {
	systemlog.Infof("Task %s moved backwards from %s to %s - clearing execution data", taskID, fromStatus, toStatus)
	task.Results = nil
	task.StartedAt = ""
	task.CompletedAt = ""
	task.CurrentPhase = ""
}

// applyStatusTransitionLogic handles status change logic including timestamps, completion counts,
// backwards transitions, and process termination. Returns the updated timestamp string.
func (h *TaskHandlers) applyStatusTransitionLogic(task *tasks.TaskItem, taskID, currentStatus, newStatus string) (string, error) {
	now := timeutil.NowRFC3339()

	// Handle backwards status transitions (completed/failed -> pending/in-progress)
	backwards := isBackwardsTransition(currentStatus, newStatus)
	if backwards {
		clearTaskExecutionData(task, taskID, currentStatus, newStatus)
	} else {
		// Normal transitions - preserve existing data if not explicitly cleared
		// (Frontend can clear by sending empty values in the update)
	}

	// Set status-specific timestamps and state
	if newStatus == "in-progress" {
		task.StartedAt = now
	}

	if newStatus == "completed" {
		if task.CompletionCount <= 0 || backwards {
			task.CompletionCount = 1
		} else {
			task.CompletionCount++
		}
		if task.CompletedAt == "" {
			task.CompletedAt = now
		}
		task.LastCompletedAt = task.CompletedAt
	}

	if newStatus == "failed" && task.CompletedAt == "" {
		task.CompletedAt = now
	}

	if newStatus == "archived" {
		if strings.TrimSpace(task.CurrentPhase) == "" {
			task.CurrentPhase = "archived"
		}
		task.ProcessorAutoRequeue = false
	}

	if newStatus == "pending" {
		task.ConsecutiveCompletionClaims = 0
		task.ConsecutiveFailures = 0
		task.ProcessorAutoRequeue = true
	}

	if newStatus == "completed-finalized" {
		task.ProcessorAutoRequeue = false
	}

	if newStatus == "failed-blocked" {
		task.ProcessorAutoRequeue = false
	}

	// CRITICAL: If task is moved OUT of in-progress, terminate any running process
	if currentStatus == "in-progress" && newStatus != "in-progress" {
		if err := h.processor.TerminateRunningProcess(taskID); err != nil {
			systemlog.Warnf("Failed to terminate process for task %s: %v", taskID, err)
		} else {
			systemlog.Infof("Successfully terminated process for task %s (moved from in-progress to %s)", taskID, newStatus)
			// Update task to reflect cancellation
			cancelTime := timeutil.NowRFC3339()
			task.Results = map[string]any{
				"success":      false,
				"error":        fmt.Sprintf("Task execution was cancelled (moved to %s)", newStatus),
				"cancelled_at": cancelTime,
			}
			task.CurrentPhase = "cancelled"
		}
	}

	return now, nil
}

// GetTasksHandler retrieves tasks with optional filtering
func (h *TaskHandlers) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	taskType := r.URL.Query().Get("type")       // filter by resource/scenario
	operation := r.URL.Query().Get("operation") // filter by generator/improver
	category := r.URL.Query().Get("category")   // filter by category

	items, err := h.storage.GetQueueItems(status)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tasks: %v", err), http.StatusInternalServerError)
		return
	}

	// Apply filters
	filteredItems := []tasks.TaskItem{} // Initialize as empty slice, not nil
	for _, item := range items {
		if taskType != "" && item.Type != taskType {
			continue
		}
		if operation != "" && item.Operation != operation {
			continue
		}
		if category != "" && item.Category != category {
			continue
		}
		filteredItems = append(filteredItems, item)
	}

	systemlog.Debugf("Task list requested: status=%s count=%d", status, len(filteredItems))

	// Wrap response in object for consistency with other endpoints
	response := map[string]any{
		"tasks": filteredItems,
		"count": len(filteredItems),
	}

	writeJSON(w, response, http.StatusOK)
}

// CreateTaskHandler creates a new task
func (h *TaskHandlers) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task tasks.TaskItem
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Normalize target inputs before validation
	task.Targets, task.Target = tasks.NormalizeTargets(task.Target, task.Targets)

	// Validate task type and operation
	validTypes := []string{"resource", "scenario"}
	validOperations := []string{"generator", "improver"}

	if !slices.Contains(validTypes, task.Type) {
		http.Error(w, fmt.Sprintf("Invalid type: %s. Must be one of: %v", task.Type, validTypes), http.StatusBadRequest)
		return
	}

	if !slices.Contains(validOperations, task.Operation) {
		http.Error(w, fmt.Sprintf("Invalid operation: %s. Must be one of: %v", task.Operation, validOperations), http.StatusBadRequest)
		return
	}

	// Validate that we have configuration for this operation
	_, err := h.assembler.SelectPromptAssembly(task.Type, task.Operation)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unsupported operation combination: %v", err), http.StatusBadRequest)
		return
	}

	// Target validation for improver operations
	if task.Operation == "improver" && len(task.Targets) == 0 {
		http.Error(w, "Improver tasks require at least one target", http.StatusBadRequest)
		return
	}

	// Generators shouldn't carry target metadata
	if task.Operation == "generator" {
		task.Target = ""
		task.Targets = nil
	}

	// Handle multi-target creation as a batch operation
	if len(task.Targets) > 1 {
		h.handleMultiTargetCreate(w, task)
		return
	}

	// Guard against duplicate improver tasks for the same target
	if len(task.Targets) == 1 {
		existing, status, lookupErr := h.storage.FindActiveTargetTask(task.Type, task.Operation, task.Targets[0])
		if lookupErr != nil {
			http.Error(w, fmt.Sprintf("Failed to verify existing tasks: %v", lookupErr), http.StatusInternalServerError)
			return
		}

		if existing != nil {
			http.Error(w, fmt.Sprintf("An active %s task (%s) already exists for %s (%s status)", task.Operation, existing.ID, task.Targets[0], status), http.StatusConflict)
			return
		}
	}

	// Set defaults
	if task.ID == "" {
		task.ID = generateTaskID(task.Type, task.Operation, task.Target)
	}

	if task.Status == "" {
		task.Status = "pending"
	}

	if strings.TrimSpace(task.Title) == "" {
		task.Title = deriveTaskTitle("", task.Operation, task.Type, task.Target)
	}

	if task.CreatedAt == "" {
		task.CreatedAt = timeutil.NowRFC3339()
	}

	task.UpdatedAt = timeutil.NowRFC3339()

	// Ensure canonical single-target representation is persisted
	if len(task.Targets) == 1 {
		task.Target = task.Targets[0]
	}

	// Save to pending queue
	if err := h.storage.SaveQueueItem(task, "pending"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"success": true,
		"task":    task,
	}, http.StatusCreated)
}

// GetTaskHandler retrieves a specific task by ID
func (h *TaskHandlers) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, _, err := h.storage.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	writeJSON(w, task, http.StatusOK)
}

// GetTaskLogsHandler streams back collected execution logs for a task
func (h *TaskHandlers) GetTaskLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	afterParam := r.URL.Query().Get("after")
	var afterSeq int64
	if afterParam != "" {
		seq, err := strconv.ParseInt(afterParam, 10, 64)
		if err != nil {
			http.Error(w, "after must be an integer", http.StatusBadRequest)
			return
		}
		afterSeq = seq
	}

	entries, nextSeq, running, agentID, completed, processID := h.processor.GetTaskLogs(taskID, afterSeq)

	writeJSON(w, map[string]any{
		"task_id":       taskID,
		"agent_id":      agentID,
		"process_id":    processID,
		"running":       running,
		"completed":     completed,
		"next_sequence": nextSeq,
		"entries":       entries,
		"timestamp":     time.Now().Unix(),
	}, http.StatusOK)
}

// GetActiveTargetsHandler returns active targets for the specified type and operation across relevant queues.
func (h *TaskHandlers) GetActiveTargetsHandler(w http.ResponseWriter, r *http.Request) {
	taskType := strings.TrimSpace(r.URL.Query().Get("type"))
	operation := strings.TrimSpace(r.URL.Query().Get("operation"))

	if taskType == "" || operation == "" {
		http.Error(w, "type and operation query parameters are required", http.StatusBadRequest)
		return
	}

	reqType := strings.ToLower(taskType)
	reqOperation := strings.ToLower(operation)

	statuses := []string{"pending", "in-progress", "review"}
	response := make([]map[string]string, 0)
	seen := make(map[string]struct{})

	for _, status := range statuses {
		items, err := h.storage.GetQueueItems(status)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load %s tasks: %v", status, err), http.StatusInternalServerError)
			return
		}

		for i := range items {
			item := items[i]
			itemType := strings.ToLower(strings.TrimSpace(item.Type))
			itemOperation := strings.ToLower(strings.TrimSpace(item.Operation))

			if itemType != reqType || itemOperation != reqOperation {
				continue
			}

			targets := h.collectTaskTargets(&item)
			for _, target := range targets {
				normalized := strings.ToLower(strings.TrimSpace(target))
				if normalized == "" {
					continue
				}

				if _, exists := seen[normalized]; exists {
					continue
				}
				seen[normalized] = struct{}{}

				response = append(response, map[string]string{
					"target":  target,
					"task_id": item.ID,
					"status":  status,
					"title":   item.Title,
				})
			}
		}
	}

	writeJSON(w, response, http.StatusOK)
}

func (h *TaskHandlers) collectTaskTargets(item *tasks.TaskItem) []string {
	normalizedTargets, _ := tasks.NormalizeTargets(item.Target, item.Targets)
	if len(normalizedTargets) > 0 {
		return normalizedTargets
	}

	var derived []string

	if strings.EqualFold(item.Type, "resource") {
		derived = append(derived, item.RelatedResources...)
	} else if strings.EqualFold(item.Type, "scenario") {
		derived = append(derived, item.RelatedScenarios...)
	}

	for _, candidate := range derived {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			normalizedTargets = append(normalizedTargets, trimmed)
		}
	}

	if len(normalizedTargets) > 0 {
		return normalizedTargets
	}

	if inferred := inferTargetFromTitle(item.Title, item.Type); inferred != "" {
		return []string{inferred}
	}

	if inferred := inferTargetFromID(item.ID, item.Type, item.Operation); inferred != "" {
		return []string{inferred}
	}

	return normalizedTargets
}

func inferTargetFromTitle(title string, taskType string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return ""
	}

	typeToken := strings.ToLower(strings.TrimSpace(taskType))
	var typePattern string
	if typeToken == "" {
		typePattern = "(resource|scenario)"
	} else {
		typePattern = regexp.QuoteMeta(typeToken)
	}

	tarTargetPattern := `([A-Za-z0-9][A-Za-z0-9\-_.\s/]+?)`
	patterns := []string{
		fmt.Sprintf(`(?i)(enhance|improve|upgrade|fix|polish)\s+%s\s+%s`, typePattern, tarTargetPattern),
		fmt.Sprintf(`(?i)(enhance|improve|upgrade|fix|polish)\s+%s\s+%s`, tarTargetPattern, typePattern),
	}

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}

		match := re.FindStringSubmatch(trimmed)
		if len(match) >= 3 {
			target := strings.TrimSpace(match[len(match)-1])
			target = strings.Trim(target, "-_")
			target = strings.ReplaceAll(target, "  ", " ")
			if target != "" {
				return target
			}
		}
	}

	return ""
}

func inferTargetFromID(id, taskType, operation string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ""
	}

	prefix := strings.ToLower(strings.TrimSpace(taskType))
	op := strings.ToLower(strings.TrimSpace(operation))
	compoundPrefix := fmt.Sprintf("%s-%s-", prefix, op)

	lowerID := strings.ToLower(trimmed)
	if strings.HasPrefix(lowerID, compoundPrefix) {
		trimmed = trimmed[len(compoundPrefix):]
	}

	reNumeric := regexp.MustCompile(`-[0-9]{4,}$`)
	for {
		if loc := reNumeric.FindStringIndex(trimmed); loc != nil && loc[0] > 0 {
			trimmed = trimmed[:loc[0]]
		} else {
			break
		}
	}

	trimmed = strings.Trim(trimmed, "-_")
	trimmed = strings.TrimSpace(trimmed)
	trimmed = strings.ReplaceAll(trimmed, "-", " ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")

	if trimmed == "" {
		return ""
	}

	return trimmed
}

// UpdateTaskHandler updates an existing task
func (h *TaskHandlers) UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	var updatedTask tasks.TaskItem
	if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	updatedTask.Targets, updatedTask.Target = tasks.NormalizeTargets(updatedTask.Target, updatedTask.Targets)

	// Find current task location
	currentTask, currentStatus, err := h.storage.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Preserve certain fields that shouldn't be changed via general update
	updatedTask.ID = currentTask.ID
	updatedTask.Type = currentTask.Type
	updatedTask.CreatedBy = currentTask.CreatedBy
	updatedTask.CreatedAt = currentTask.CreatedAt

	// Allow operation to be updated but preserve if not provided
	if updatedTask.Operation == "" {
		updatedTask.Operation = currentTask.Operation
	}

	if updatedTask.Operation == "generator" {
		updatedTask.Target = ""
		updatedTask.Targets = nil
	}

	// Validate operation if it was changed
	validOperations := []string{"generator", "improver"}
	if !slices.Contains(validOperations, updatedTask.Operation) {
		http.Error(w, fmt.Sprintf("Invalid operation: %s. Must be one of: %v", updatedTask.Operation, validOperations), http.StatusBadRequest)
		return
	}

	// Validate that we have configuration for the new operation combination
	if updatedTask.Operation != currentTask.Operation {
		_, err := h.assembler.SelectPromptAssembly(updatedTask.Type, updatedTask.Operation)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unsupported operation combination after update: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Preserve all other fields if they weren't provided in the update
	preserveUnsetFields(&updatedTask, currentTask)

	// Validate status if provided
	newStatus := updatedTask.Status
	if newStatus != "" && !tasks.IsValidStatus(newStatus) {
		http.Error(w, fmt.Sprintf("Invalid status: %s. Must be one of: %v", newStatus, tasks.GetValidStatuses()), http.StatusBadRequest)
		return
	}

	// Handle status transitions using consolidated logic
	backwards := isBackwardsTransition(currentStatus, newStatus)

	// Debug logging
	systemlog.Debugf("Task %s status transition: '%s' -> '%s' (backwards: %v)",
		taskID, currentStatus, newStatus, backwards)
	systemlog.Debugf("Task %s incoming data - has results: %v, status: '%s'",
		taskID, updatedTask.Results != nil, updatedTask.Status)

	// Preserve existing data if not explicitly provided by frontend
	if !backwards {
		if updatedTask.StartedAt == "" {
			updatedTask.StartedAt = currentTask.StartedAt
		}
		if updatedTask.CompletedAt == "" {
			updatedTask.CompletedAt = currentTask.CompletedAt
		}
		if updatedTask.LastCompletedAt == "" {
			updatedTask.LastCompletedAt = currentTask.LastCompletedAt
		}
		// IMPORTANT: Don't preserve results if moving to active state
		if updatedTask.Results == nil && currentTask.Results != nil {
			if newStatus == "pending" || newStatus == "in-progress" {
				systemlog.Debugf("Task %s: Not preserving results when moving to %s", taskID, newStatus)
			} else {
				updatedTask.Results = currentTask.Results
			}
		}
	}

	// Apply status transition logic
	if newStatus != currentStatus {
		updatedTask.CompletionCount = currentTask.CompletionCount // Preserve for increment logic
		now, err := h.applyStatusTransitionLogic(&updatedTask, taskID, currentStatus, newStatus)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to apply status transition: %v", err), http.StatusInternalServerError)
			return
		}
		updatedTask.UpdatedAt = now
	} else {
		// Just update timestamp
		updatedTask.UpdatedAt = timeutil.NowRFC3339()
	}

	// Save updated task
	if newStatus != currentStatus {
		// Move the task file to the new status directory
		if err := h.storage.MoveTask(taskID, currentStatus, newStatus); err != nil {
			systemlog.Errorf("Failed to move task %s from %s to %s via MoveTask: %v", taskID, currentStatus, newStatus, err)
			http.Error(w, fmt.Sprintf("Failed to move task: %v", err), http.StatusInternalServerError)
			return
		}
		// CRITICAL: Save the updated task with all field changes to the new location
		// Do NOT reload from disk as that would discard all the field updates we just applied
		if err := h.storage.SaveQueueItem(updatedTask, newStatus); err != nil {
			systemlog.Errorf("Failed to save updated task %s after move to %s: %v", taskID, newStatus, err)
			http.Error(w, fmt.Sprintf("Error saving updated task: %v", err), http.StatusInternalServerError)
			return
		}
		currentStatus = newStatus
	} else {
		if err := h.storage.SaveQueueItem(updatedTask, currentStatus); err != nil {
			http.Error(w, fmt.Sprintf("Error saving updated task: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Broadcast the update via WebSocket
	h.wsManager.BroadcastUpdate("task_updated", updatedTask)

	systemlog.Infof("Task %s updated successfully", taskID)

	writeJSON(w, map[string]any{
		"success": true,
		"task":    updatedTask,
	}, http.StatusOK)
}

// DeleteTaskHandler deletes a task
func (h *TaskHandlers) DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Check if task is running and terminate if necessary
	if err := h.processor.TerminateRunningProcess(taskID); err == nil {
		systemlog.Infof("Terminated running process for deleted task %s", taskID)
	}

	// Delete the task file
	status, err := h.storage.DeleteTask(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete task: %v", err), http.StatusInternalServerError)
		return
	}

	// Send WebSocket notification
	h.wsManager.BroadcastUpdate("task_deleted", map[string]any{
		"id":     taskID,
		"status": status,
	})

	w.WriteHeader(http.StatusNoContent) // 204 No Content for successful deletion
}

// GetTaskPromptHandler retrieves prompt details for a task
func (h *TaskHandlers) GetTaskPromptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Find task
	task, _, err := h.storage.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Generate prompt sections
	sections, err := h.assembler.GeneratePromptSections(*task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate prompt: %v", err), http.StatusInternalServerError)
		return
	}

	operationConfig, _ := h.assembler.SelectPromptAssembly(task.Type, task.Operation)

	response := map[string]any{
		"task_id":          task.ID,
		"operation":        fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt_sections":  sections,
		"operation_config": operationConfig,
		"task_details":     task,
	}

	writeJSON(w, response, http.StatusOK)
}

// GetAssembledPromptHandler returns the fully assembled prompt for a task
func (h *TaskHandlers) GetAssembledPromptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Find task
	task, status, err := h.storage.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	fromCache := false
	assembly, err := h.assembler.AssemblePromptForTask(*task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}
	prompt := assembly.Prompt

	// Check for cached prompt content (legacy behavior)
	promptPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s%s.txt", queue.PromptFilePrefix, taskID))
	if cachedPrompt, err := os.ReadFile(promptPath); err == nil {
		prompt = string(cachedPrompt)
		fromCache = true
		systemlog.Debugf("Using cached prompt from %s", promptPath)
	}

	assembly.Prompt = prompt

	// Get operation config for metadata
	operationConfig, _ := h.assembler.SelectPromptAssembly(task.Type, task.Operation)

	response := map[string]any{
		"task_id":           task.ID,
		"operation":         fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt":            prompt,
		"prompt_length":     len(prompt),
		"prompt_cached":     fromCache,
		"sections_detailed": assembly.Sections,
		"operation_config":  operationConfig,
		"task_status":       status,
		"task_details":      task,
	}

	writeJSON(w, response, http.StatusOK)
}

// UpdateTaskStatusHandler updates just the status/progress of a task (simpler than full update)
func (h *TaskHandlers) UpdateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	var update struct {
		Status       string `json:"status"`
		CurrentPhase string `json:"current_phase"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Find current task location
	task, currentStatus, err := h.storage.GetTaskByID(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Validate status if provided
	if update.Status != "" && !tasks.IsValidStatus(update.Status) {
		http.Error(w, fmt.Sprintf("Invalid status: %s. Must be one of: %v", update.Status, tasks.GetValidStatuses()), http.StatusBadRequest)
		return
	}

	// Update task fields
	if update.Status != "" && update.Status != currentStatus {
		task.Status = update.Status

		// Apply consolidated status transition logic
		now, err := h.applyStatusTransitionLogic(task, taskID, currentStatus, update.Status)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to apply status transition: %v", err), http.StatusInternalServerError)
			return
		}
		task.UpdatedAt = now

		// Move task to new status if needed
		if update.Status != currentStatus {
			if err := h.storage.MoveTask(taskID, currentStatus, update.Status); err != nil {
				http.Error(w, fmt.Sprintf("Failed to move task: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	if update.CurrentPhase != "" {
		task.CurrentPhase = update.CurrentPhase
	}

	task.UpdatedAt = timeutil.NowRFC3339()

	// Save updated task
	if err := h.storage.SaveQueueItem(*task, task.Status); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast the update via WebSocket
	h.wsManager.BroadcastUpdate("task_status_updated", *task)

	writeJSON(w, *task, http.StatusOK)
}

// promptPreviewRequest captures optional data for assembling a preview task
type promptPreviewRequest struct {
	Task      *tasks.TaskItem `json:"task,omitempty"`
	Display   string          `json:"display,omitempty"`
	Type      string          `json:"type,omitempty"`
	Operation string          `json:"operation,omitempty"`
	Title     string          `json:"title,omitempty"`
	Category  string          `json:"category,omitempty"`
	Priority  string          `json:"priority,omitempty"`
	Notes     string          `json:"notes,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
}

func (r promptPreviewRequest) buildTask(defaultID string) tasks.TaskItem {
	var task tasks.TaskItem

	if r.Task != nil {
		task = *r.Task
	}

	if r.Type != "" {
		task.Type = r.Type
	}
	if r.Operation != "" {
		task.Operation = r.Operation
	}
	if r.Title != "" {
		task.Title = r.Title
	}
	if r.Category != "" {
		task.Category = r.Category
	}
	if r.Priority != "" {
		task.Priority = r.Priority
	}
	if r.Notes != "" {
		task.Notes = r.Notes
	}
	if len(r.Tags) > 0 {
		task.Tags = r.Tags
	}

	if task.ID == "" {
		task.ID = defaultID
	}
	if task.Type == "" {
		task.Type = "resource"
	}
	if task.Operation == "" {
		task.Operation = "generator"
	}
	if task.Title == "" {
		task.Title = "Prompt Viewer Test"
	}
	if task.Category == "" {
		task.Category = "test"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.Notes == "" {
		task.Notes = "Temporary task for prompt viewing"
	}
	if task.CreatedAt == "" {
		task.CreatedAt = timeutil.NowRFC3339()
	}
	if task.Status == "" {
		task.Status = "pending"
	}

	return task
}

// PromptViewerHandler creates a temporary task to view assembled prompts
func (h *TaskHandlers) PromptViewerHandler(w http.ResponseWriter, r *http.Request) {
	defaultID := fmt.Sprintf("temp-prompt-viewer-%d", time.Now().UnixNano())

	var req promptPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	display := strings.ToLower(strings.TrimSpace(req.Display))
	if display == "" {
		display = "preview"
	}

	tempTask := req.buildTask(defaultID)

	if _, err := h.assembler.SelectPromptAssembly(tempTask.Type, tempTask.Operation); err != nil {
		http.Error(w, fmt.Sprintf("Unsupported operation combination %s/%s: %v", tempTask.Type, tempTask.Operation, err), http.StatusBadRequest)
		return
	}

	sections, err := h.assembler.GeneratePromptSections(tempTask)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get prompt sections: %v", err), http.StatusInternalServerError)
		return
	}

	assembly, err := h.assembler.AssemblePromptForTask(tempTask)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}
	prompt := assembly.Prompt

	promptSize := len(prompt)
	promptSizeKB := float64(promptSize) / 1024.0
	promptSizeMB := promptSizeKB / 1024.0

	response := map[string]any{
		"task_type":         tempTask.Type,
		"operation":         tempTask.Operation,
		"title":             tempTask.Title,
		"sections":          sections,
		"section_count":     len(sections),
		"sections_detailed": assembly.Sections,
		"prompt_size":       promptSize,
		"prompt_size_kb":    fmt.Sprintf("%.2f", promptSizeKB),
		"prompt_size_mb":    fmt.Sprintf("%.3f", promptSizeMB),
		"timestamp":         timeutil.NowRFC3339(),
		"task":              tempTask,
	}

	switch display {
	case "full", "all":
		response["prompt"] = prompt
		response["display"] = "full"
	case "first", "preview":
		response["prompt"] = prompt
		response["display"] = "preview"
		response["truncated"] = false
	case "size", "stats":
		response["display"] = "size"
	default:
		response["display"] = "summary"
		response["available_displays"] = []string{"full", "preview", "size"}
	}

	writeJSON(w, response, http.StatusOK)
}
