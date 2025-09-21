package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

	// Set defaults
	if task.ID == "" {
		timestamp := time.Now().Format("20060102-150405")
		task.ID = fmt.Sprintf("%s-%s-%s", task.Type, task.Operation, timestamp)
	}

	if task.Status == "" {
		task.Status = "pending"
	}

	if task.CreatedAt == "" {
		task.CreatedAt = time.Now().Format(time.RFC3339)
	}

	task.UpdatedAt = time.Now().Format(time.RFC3339)

	// Save to pending queue
	if err := h.storage.SaveQueueItem(task, "pending"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
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

// UpdateTaskHandler updates an existing task
func (h *TaskHandlers) UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	var updatedTask tasks.TaskItem
	if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

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
		updatedTask.ProgressPercent = 0
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
	if newStatus != currentStatus {
		// Set timestamps for status changes
		now := time.Now().Format(time.RFC3339)
		updatedTask.UpdatedAt = now

		if newStatus == "in-progress" && currentTask.StartedAt == "" {
			updatedTask.StartedAt = now
		} else if (newStatus == "completed" || newStatus == "failed") && currentTask.CompletedAt == "" {
			updatedTask.CompletedAt = now
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

	var prompt string
	var fromCache bool

	// First check if we have a cached prompt in /tmp
	promptPath := fmt.Sprintf("/tmp/ecosystem-prompt-%s.txt", taskID)
	if cachedPrompt, err := os.ReadFile(promptPath); err == nil {
		prompt = string(cachedPrompt)
		fromCache = true
		log.Printf("Using cached prompt from %s", promptPath)
	} else {
		// Generate the full assembled prompt
		prompt, err = h.assembler.AssemblePromptForTask(*task)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
			return
		}
		fromCache = false
	}

	// Get operation config for metadata
	operationConfig, _ := h.assembler.SelectPromptAssembly(task.Type, task.Operation)

	response := map[string]interface{}{
		"task_id":          task.ID,
		"operation":        fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt":           prompt,
		"prompt_length":    len(prompt),
		"prompt_cached":    fromCache,
		"operation_config": operationConfig,
		"task_status":      status,
		"task_details":     task,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateTaskStatusHandler updates just the status/progress of a task (simpler than full update)
func (h *TaskHandlers) UpdateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	var update struct {
		Status          string `json:"status"`
		ProgressPercent int    `json:"progress_percentage"`
		CurrentPhase    string `json:"current_phase"`
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
		task.Status = update.Status

		// Handle backwards status transitions (completed/failed -> pending/in-progress)
		isBackwardsTransition := (currentStatus == "completed" || currentStatus == "failed") &&
			(update.Status == "pending" || update.Status == "in-progress")

		if isBackwardsTransition {
			// Clear execution results and timestamps for fresh execution
			log.Printf("Task %s moved backwards from %s to %s - clearing execution data", taskID, currentStatus, update.Status)
			task.Results = nil
			task.StartedAt = ""
			task.CompletedAt = ""
			task.ProgressPercent = 0
			task.CurrentPhase = ""
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

	if update.ProgressPercent > 0 {
		task.ProgressPercent = update.ProgressPercent
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
	Task         *tasks.TaskItem        `json:"task,omitempty"`
	Display      string                 `json:"display,omitempty"`
	Type         string                 `json:"type,omitempty"`
	Operation    string                 `json:"operation,omitempty"`
	Title        string                 `json:"title,omitempty"`
	Category     string                 `json:"category,omitempty"`
	Priority     string                 `json:"priority,omitempty"`
	Notes        string                 `json:"notes,omitempty"`
	Requirements map[string]interface{} `json:"requirements,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
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
	if r.Requirements != nil {
		task.Requirements = r.Requirements
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

	prompt, err := h.assembler.AssemblePromptForTask(tempTask)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}

	promptSize := len(prompt)
	promptSizeKB := float64(promptSize) / 1024.0
	promptSizeMB := promptSizeKB / 1024.0

	response := map[string]interface{}{
		"task_type":      tempTask.Type,
		"operation":      tempTask.Operation,
		"title":          tempTask.Title,
		"sections":       sections,
		"section_count":  len(sections),
		"prompt_size":    promptSize,
		"prompt_size_kb": fmt.Sprintf("%.2f", promptSizeKB),
		"prompt_size_mb": fmt.Sprintf("%.3f", promptSizeMB),
		"timestamp":      time.Now().Format(time.RFC3339),
		"task":           tempTask,
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
