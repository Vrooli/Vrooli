package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
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
	
	taskType := r.URL.Query().Get("type")     // filter by resource/scenario
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
	updatedTask.Operation = currentTask.Operation
	updatedTask.CreatedBy = currentTask.CreatedBy
	updatedTask.CreatedAt = currentTask.CreatedAt
	updatedTask.StartedAt = currentTask.StartedAt
	updatedTask.CompletedAt = currentTask.CompletedAt
	updatedTask.Results = currentTask.Results
	
	// Handle status changes - if status changed, may need to move file
	newStatus := updatedTask.Status
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
					"success": false,
					"error":   fmt.Sprintf("Task execution was cancelled (moved to %s)", newStatus),
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
		// Move to new status
		if err := h.storage.SaveQueueItem(updatedTask, newStatus); err != nil {
			http.Error(w, fmt.Sprintf("Error saving updated task: %v", err), http.StatusInternalServerError)
			return
		}
		// Remove from old status
		h.storage.DeleteTask(taskID) // This will find and delete from the old location
	} else {
		// Update in place
		if err := h.storage.SaveQueueItem(updatedTask, currentStatus); err != nil {
			http.Error(w, fmt.Sprintf("Error saving updated task: %v", err), http.StatusInternalServerError)
			return
		}
	}
	
	// Broadcast the update via WebSocket
	h.wsManager.BroadcastUpdate("task_updated", updatedTask)
	
	log.Printf("Task %s updated successfully", taskID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTask)
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
		"id": taskID,
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
		"task_id":           task.ID,
		"operation":         fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt_sections":   sections,
		"operation_config":  operationConfig,
		"task_details":      task,
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
	
	// Generate the full assembled prompt
	prompt, err := h.assembler.AssemblePromptForTask(*task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Get operation config for metadata
	operationConfig, _ := h.assembler.SelectPromptAssembly(task.Type, task.Operation)
	
	response := map[string]interface{}{
		"task_id":          task.ID,
		"operation":        fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt":           prompt,
		"prompt_length":    len(prompt),
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
		
		// CRITICAL: If task is moved OUT of in-progress, terminate any running process
		if currentStatus == "in-progress" && update.Status != "in-progress" {
			if err := h.processor.TerminateRunningProcess(taskID); err != nil {
				log.Printf("Warning: Failed to terminate process for task %s: %v", taskID, err)
			} else {
				log.Printf("Successfully terminated process for task %s (moved from in-progress to %s)", taskID, update.Status)
				// Update task to reflect cancellation
				now := time.Now().Format(time.RFC3339)
				task.Results = map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("Task execution was cancelled (moved to %s)", update.Status),
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