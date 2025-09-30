package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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
		timestamp := time.Now().Format(time.RFC3339)
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
	response := map[string]interface{}{
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
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

	w.Header().Set("Content-Type", "application/json")

	// Debug: Log what we're about to send
	log.Printf("About to send %d filtered tasks", len(filteredItems))
	systemlog.Debugf("Task list requested: status=%s count=%d", status, len(filteredItems))

	// Encode and check for errors
	if err := json.NewEncoder(w).Encode(filteredItems); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	} else {
		log.Printf("Successfully sent JSON response with %d tasks", len(filteredItems))
	}
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

	if !tasks.Contains(validTypes, task.Type) {
		http.Error(w, fmt.Sprintf("Invalid type: %s. Must be one of: %v", task.Type, validTypes), http.StatusBadRequest)
		return
	}

	if !tasks.Contains(validOperations, task.Operation) {
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

	now := time.Now()

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
		task.CreatedAt = now.Format(time.RFC3339)
	}

	task.UpdatedAt = now.Format(time.RFC3339)

	// Ensure canonical single-target representation is persisted
	if len(task.Targets) == 1 {
		task.Target = task.Targets[0]
	}

	// Save to pending queue
	if err := h.storage.SaveQueueItem(task, "pending"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"task":    task,
	})
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id":       taskID,
		"agent_id":      agentID,
		"process_id":    processID,
		"running":       running,
		"completed":     completed,
		"next_sequence": nextSeq,
		"entries":       entries,
		"timestamp":     time.Now().Unix(),
	})
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode active targets response: %v", err)
	}
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
	if !tasks.Contains(validOperations, updatedTask.Operation) {
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
	if updatedTask.Title == "" {
		updatedTask.Title = currentTask.Title
	}
	if len(updatedTask.Targets) == 0 && len(currentTask.Targets) > 0 {
		updatedTask.Targets = currentTask.Targets
		updatedTask.Target = currentTask.Target
	}
	if updatedTask.Priority == "" {
		updatedTask.Priority = currentTask.Priority
	}
	if updatedTask.Category == "" {
		updatedTask.Category = currentTask.Category
	}
	if updatedTask.Notes == "" {
		updatedTask.Notes = currentTask.Notes
	}
	if updatedTask.EffortEstimate == "" {
		updatedTask.EffortEstimate = currentTask.EffortEstimate
	}
	if updatedTask.CurrentPhase == "" && currentTask.CurrentPhase != "" {
		updatedTask.CurrentPhase = currentTask.CurrentPhase
	}
	if updatedTask.CompletionCount == 0 && currentTask.CompletionCount > 0 {
		updatedTask.CompletionCount = currentTask.CompletionCount
	}
	if updatedTask.LastCompletedAt == "" {
		updatedTask.LastCompletedAt = currentTask.LastCompletedAt
	}

	// Handle backwards status transitions (completed/failed -> pending/in-progress)
	// This happens when users drag tasks back to re-execute them
	newStatus := updatedTask.Status
	isBackwardsTransition := (currentStatus == "completed" || currentStatus == "failed") &&
		(newStatus == "pending" || newStatus == "in-progress")

	// Debug logging
	log.Printf("Task %s status transition: '%s' -> '%s' (backwards: %v)",
		taskID, currentStatus, newStatus, isBackwardsTransition)
	log.Printf("Task %s incoming data - has results: %v, status: '%s'",
		taskID, updatedTask.Results != nil, updatedTask.Status)

	if isBackwardsTransition {
		// Clear execution results and timestamps for fresh execution
		log.Printf("Task %s moved backwards from %s to %s - clearing execution data", taskID, currentStatus, newStatus)
		updatedTask.Results = nil
		updatedTask.StartedAt = ""
		updatedTask.CompletedAt = ""
		updatedTask.CurrentPhase = ""
	} else {
		// Normal status transitions - preserve existing data if not provided
		// Only preserve if the frontend didn't explicitly clear them
		if updatedTask.StartedAt == "" {
			updatedTask.StartedAt = currentTask.StartedAt
		}
		if updatedTask.CompletedAt == "" {
			updatedTask.CompletedAt = currentTask.CompletedAt
		}
		if updatedTask.LastCompletedAt == "" {
			updatedTask.LastCompletedAt = currentTask.LastCompletedAt
		}
		// IMPORTANT: Don't preserve results if it's a backwards transition that wasn't detected
		// This can happen if status strings don't match exactly
		if updatedTask.Results == nil && currentTask.Results != nil {
			// Check if this might be an undetected backwards transition
			if newStatus == "pending" || newStatus == "in-progress" {
				log.Printf("Task %s: Not preserving results when moving to %s", taskID, newStatus)
				// Keep results as nil
			} else {
				updatedTask.Results = currentTask.Results
			}
		}
	}

	// Handle status changes - if status changed, may need to move file
	if newStatus == "archived" && strings.TrimSpace(updatedTask.CurrentPhase) == "" {
		updatedTask.CurrentPhase = "archived"
	}

	if newStatus != currentStatus {
		// Set timestamps for status changes
		now := time.Now().Format(time.RFC3339)
		updatedTask.UpdatedAt = now

		if newStatus == "in-progress" {
			updatedTask.StartedAt = now
		}
		if newStatus == "completed" {
			if updatedTask.CompletionCount <= currentTask.CompletionCount {
				updatedTask.CompletionCount = currentTask.CompletionCount + 1
			}
			if updatedTask.CompletedAt == "" {
				updatedTask.CompletedAt = now
			}
			updatedTask.LastCompletedAt = updatedTask.CompletedAt
		}
		if newStatus == "failed" && updatedTask.CompletedAt == "" {
			updatedTask.CompletedAt = now
		}
		if newStatus == "archived" {
			updatedTask.ProcessorAutoRequeue = false
		}

		// CRITICAL: If task is moved OUT of in-progress, terminate any running process
		if currentStatus == "in-progress" && newStatus != "in-progress" {
			if err := h.processor.TerminateRunningProcess(taskID); err != nil {
				log.Printf("Warning: Failed to terminate process for task %s: %v", taskID, err)
			} else {
				log.Printf("Successfully terminated process for task %s (moved from in-progress to %s)", taskID, newStatus)
				// Update task to reflect cancellation
				updatedTask.Results = map[string]interface{}{
					"success":      false,
					"error":        fmt.Sprintf("Task execution was cancelled (moved to %s)", newStatus),
					"cancelled_at": now,
				}
				updatedTask.CurrentPhase = "cancelled"
			}
		}
	} else {
		// Just update timestamp
		updatedTask.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	// Save updated task
	if newStatus != currentStatus {
		if err := h.storage.MoveTask(taskID, currentStatus, newStatus); err != nil {
			log.Printf("ERROR: Failed to move task %s from %s to %s via MoveTask: %v", taskID, currentStatus, newStatus, err)
			http.Error(w, fmt.Sprintf("Failed to move task: %v", err), http.StatusInternalServerError)
			return
		}
		// Reload updated task after move so we return the latest view
		if reloaded, status, err := h.storage.GetTaskByID(taskID); err == nil {
			updatedTask = *reloaded
			currentStatus = status
		}
	} else {
		if err := h.storage.SaveQueueItem(updatedTask, currentStatus); err != nil {
			http.Error(w, fmt.Sprintf("Error saving updated task: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Broadcast the update via WebSocket
	h.wsManager.BroadcastUpdate("task_updated", updatedTask)

	log.Printf("Task %s updated successfully", taskID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"task":    updatedTask,
	})
}

// DeleteTaskHandler deletes a task
func (h *TaskHandlers) DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Check if task is running and terminate if necessary
	if err := h.processor.TerminateRunningProcess(taskID); err == nil {
		log.Printf("Terminated running process for deleted task %s", taskID)
	}

	// Delete the task file
	status, err := h.storage.DeleteTask(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete task: %v", err), http.StatusInternalServerError)
		return
	}

	// Send WebSocket notification
	h.wsManager.BroadcastUpdate("task_deleted", map[string]interface{}{
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

	response := map[string]interface{}{
		"task_id":          task.ID,
		"operation":        fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt_sections":  sections,
		"operation_config": operationConfig,
		"task_details":     task,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	promptPath := fmt.Sprintf("/tmp/ecosystem-prompt-%s.txt", taskID)
	if cachedPrompt, err := os.ReadFile(promptPath); err == nil {
		prompt = string(cachedPrompt)
		fromCache = true
		log.Printf("Using cached prompt from %s", promptPath)
	}

	assembly.Prompt = prompt

	// Get operation config for metadata
	operationConfig, _ := h.assembler.SelectPromptAssembly(task.Type, task.Operation)

	response := map[string]interface{}{
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	// Update task fields
	if update.Status != "" && update.Status != currentStatus {
		previousCount := task.CompletionCount
		task.Status = update.Status
		now := time.Now().Format(time.RFC3339)

		// Handle backwards status transitions (completed/failed -> pending/in-progress)
		isBackwardsTransition := (currentStatus == "completed" || currentStatus == "failed") &&
			(update.Status == "pending" || update.Status == "in-progress")

		if isBackwardsTransition {
			// Clear execution results and timestamps for fresh execution
			log.Printf("Task %s moved backwards from %s to %s - clearing execution data", taskID, currentStatus, update.Status)
			task.Results = nil
			task.StartedAt = ""
			task.CompletedAt = ""
			task.CurrentPhase = ""
		}

		if update.Status == "in-progress" {
			task.StartedAt = now
		}
		if update.Status == "completed" {
			if task.CompletionCount <= previousCount {
				task.CompletionCount = previousCount + 1
			}
			task.CompletedAt = now
			task.LastCompletedAt = task.CompletedAt
		}
		if update.Status == "failed" {
			task.CompletedAt = now
		}
		if update.Status == "archived" && strings.TrimSpace(task.CurrentPhase) == "" {
			task.CurrentPhase = "archived"
		}
		if update.Status == "pending" {
			task.ConsecutiveCompletionClaims = 0
			task.ConsecutiveFailures = 0
			task.ProcessorAutoRequeue = true
		}
		if update.Status == "completed-finalized" {
			task.ProcessorAutoRequeue = false
		}
		if update.Status == "failed-blocked" {
			task.ProcessorAutoRequeue = false
		}
		if update.Status == "archived" {
			task.ProcessorAutoRequeue = false
		}

		// CRITICAL: If task is moved OUT of in-progress, terminate any running process
		if currentStatus == "in-progress" && update.Status != "in-progress" {
			if err := h.processor.TerminateRunningProcess(taskID); err != nil {
				log.Printf("Warning: Failed to terminate process for task %s: %v", taskID, err)
			} else {
				log.Printf("Successfully terminated process for task %s (moved from in-progress to %s)", taskID, update.Status)
				// Update task to reflect cancellation
				now := time.Now().Format(time.RFC3339)
				task.Results = map[string]interface{}{
					"success":      false,
					"error":        fmt.Sprintf("Task execution was cancelled (moved to %s)", update.Status),
					"cancelled_at": now,
				}
				task.CurrentPhase = "cancelled"
			}
		}

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

	task.UpdatedAt = time.Now().Format(time.RFC3339)

	// Save updated task
	if err := h.storage.SaveQueueItem(*task, task.Status); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast the update via WebSocket
	h.wsManager.BroadcastUpdate("task_status_updated", *task)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(*task)
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
		task.CreatedAt = time.Now().Format(time.RFC3339)
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

	response := map[string]interface{}{
		"task_type":         tempTask.Type,
		"operation":         tempTask.Operation,
		"title":             tempTask.Title,
		"sections":          sections,
		"section_count":     len(sections),
		"sections_detailed": assembly.Sections,
		"prompt_size":       promptSize,
		"prompt_size_kb":    fmt.Sprintf("%.2f", promptSizeKB),
		"prompt_size_mb":    fmt.Sprintf("%.3f", promptSizeMB),
		"timestamp":         time.Now().Format(time.RFC3339),
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
