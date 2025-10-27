package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

// QueueHandlers contains handlers for queue-related endpoints
type QueueHandlers struct {
	processor *queue.Processor
	wsManager *websocket.Manager
	storage   *tasks.Storage
}

// NewQueueHandlers creates a new queue handlers instance
func NewQueueHandlers(processor *queue.Processor, wsManager *websocket.Manager, storage *tasks.Storage) *QueueHandlers {
	return &QueueHandlers{
		processor: processor,
		wsManager: wsManager,
		storage:   storage,
	}
}

// GetQueueStatusHandler returns the current queue status
func (h *QueueHandlers) GetQueueStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := h.processor.GetQueueStatus()

	writeJSON(w, status, http.StatusOK)
}

// GetResumeDiagnosticsHandler reports blockers that will be cleared when resuming the processor.
func (h *QueueHandlers) GetResumeDiagnosticsHandler(w http.ResponseWriter, r *http.Request) {
	if h.processor == nil {
		writeError(w, "queue processor not available", http.StatusServiceUnavailable)
		return
	}

	diagnostics := h.processor.GetResumeDiagnostics()

	response := ResumeDiagnosticsResponse{
		Success:     true,
		Diagnostics: diagnostics,
		GeneratedAt: timeutil.NowRFC3339(),
	}

	writeJSON(w, response, http.StatusOK)
}

// TriggerQueueProcessingHandler forces immediate queue processing
func (h *QueueHandlers) TriggerQueueProcessingHandler(w http.ResponseWriter, r *http.Request) {
	if h.processor == nil {
		writeJSON(w, map[string]string{
			"error":   "queue processor not available",
			"message": "Queue processor is not initialized",
		}, http.StatusServiceUnavailable)
		return
	}

	// Check maintenance state from queue status
	status := h.processor.GetQueueStatus()
	processorActive, ok := status["processor_active"].(bool)

	if !ok || !processorActive {
		writeJSON(w, map[string]any{
			"error":             "processor inactive",
			"message":           "Queue processor is paused or not active",
			"maintenance_state": status["maintenance_state"],
			"processor_active":  processorActive,
		}, http.StatusServiceUnavailable)
		return
	}

	// Trigger queue processing in a goroutine to avoid blocking
	go h.processor.ProcessQueue()

	// Also update the status after triggering
	refreshedStatus := h.processor.GetQueueStatus()

	writeJSON(w, map[string]any{
		"success":   true,
		"message":   "Queue processing triggered successfully",
		"timestamp": time.Now().Unix(),
		"status":    refreshedStatus,
	}, http.StatusOK)

	log.Println("Queue processing manually triggered via API")
}

// SetMaintenanceStateHandler sets the queue processor maintenance state
func (h *QueueHandlers) SetMaintenanceStateHandler(w http.ResponseWriter, r *http.Request) {
	type maintenanceRequest struct {
		State string `json:"state"` // "active" or "inactive"
	}

	request, ok := decodeJSONBody[maintenanceRequest](w, r)
	if !ok {
		return
	}

	if request.State != "active" && request.State != "inactive" {
		writeError(w, "State must be 'active' or 'inactive'", http.StatusBadRequest)
		return
	}

	// Set maintenance state on processor
	var resumeSummary *queue.ResumeResetSummary
	if request.State == "active" {
		summary := h.processor.ResumeWithReset()
		resumeSummary = &summary
	} else {
		h.processor.Pause()
	}

	// Get updated status
	status := h.processor.GetQueueStatus()

	response := map[string]any{
		"success": true,
		"message": "Maintenance state updated",
		"state":   request.State,
		"status":  status,
	}
	if resumeSummary != nil {
		response["resume_reset_summary"] = resumeSummary
	}

	writeJSON(w, response, http.StatusOK)

	log.Printf("Queue maintenance state set to: %s", request.State)
}

// GetRunningProcessesHandler returns information about currently running processes
func (h *QueueHandlers) GetRunningProcessesHandler(w http.ResponseWriter, r *http.Request) {
	processes := h.processor.GetRunningProcessesInfo()

	response := ProcessesListResponse{
		Processes: processes,
		Count:     len(processes),
		Timestamp: time.Now().Unix(),
	}

	writeJSON(w, response, http.StatusOK)
}

// TerminateProcessHandler terminates a specific running process
func (h *QueueHandlers) TerminateProcessHandler(w http.ResponseWriter, r *http.Request) {
	type terminateRequest struct {
		TaskID string `json:"task_id"`
	}

	request, ok := decodeJSONBody[terminateRequest](w, r)
	if !ok {
		return
	}

	if request.TaskID == "" {
		writeError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	err := h.processor.TerminateRunningProcess(request.TaskID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Find the task and move it back to pending with cleared state
	task, currentStatus, err := h.storage.GetTaskByID(request.TaskID)
	if err == nil && task != nil && currentStatus == "in-progress" {
		// Clear execution state and move to pending
		task.Status = "pending"
		task.CurrentPhase = ""
		task.StartedAt = ""
		task.Results = nil

		// Move task from in-progress to pending
		if err := h.storage.MoveTask(request.TaskID, "in-progress", "pending"); err != nil {
			log.Printf("Warning: Failed to move cancelled task %s to pending: %v", request.TaskID, err)
		} else {
			// Save the updated task with cleared state
			if err := h.storage.SaveQueueItem(*task, "pending"); err != nil {
				log.Printf("Warning: Failed to update cancelled task %s state: %v", request.TaskID, err)
			}
			// Reset any cached logs so future runs start clean
			h.processor.ResetTaskLogs(request.TaskID)

			// Broadcast task status change for UI update
			h.wsManager.BroadcastUpdate("task_status_changed", map[string]any{
				"task_id":    request.TaskID,
				"old_status": "in-progress",
				"new_status": "pending",
				"task":       task,
			})

			log.Printf("Cancelled task %s moved back to pending", request.TaskID)
		}
	}

	// Broadcast termination event
	h.wsManager.BroadcastUpdate("process_terminated", map[string]any{
		"task_id":   request.TaskID,
		"timestamp": time.Now().Unix(),
	})

	response := ProcessTerminateResponse{
		Success: true,
		Message: "Process terminated successfully",
		TaskID:  request.TaskID,
	}

	writeJSON(w, response, http.StatusOK)

	log.Printf("Process terminated for task: %s", request.TaskID)
}

// StopQueueProcessorHandler stops the queue processor
func (h *QueueHandlers) StopQueueProcessorHandler(w http.ResponseWriter, r *http.Request) {
	h.processor.Stop()

	response := SimpleSuccessResponse{
		Success: true,
		Message: "Queue processor stopped",
	}

	writeJSON(w, response, http.StatusOK)

	log.Println("Queue processor stopped via API")
	systemlog.Info("Queue processor stopped via API request")
}

// StartQueueProcessorHandler starts the queue processor
func (h *QueueHandlers) StartQueueProcessorHandler(w http.ResponseWriter, r *http.Request) {
	h.processor.Start()

	response := SimpleSuccessResponse{
		Success: true,
		Message: "Queue processor started",
	}

	writeJSON(w, response, http.StatusOK)

	log.Println("Queue processor started via API")
	systemlog.Info("Queue processor started via API request")
}

// ResetRateLimitHandler resets the rate limit pause manually
func (h *QueueHandlers) ResetRateLimitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reset the rate limit pause in the processor
	h.processor.ResetRateLimitPause()

	// Broadcast the reset event
	h.wsManager.BroadcastUpdate("rate_limit_resume", map[string]any{
		"paused":    false,
		"manual":    true,
		"timestamp": time.Now().Unix(),
	})

	// Get updated status
	status := h.processor.GetQueueStatus()

	writeJSON(w, map[string]any{
		"success": true,
		"message": "Rate limit pause has been reset",
		"status":  status,
	}, http.StatusOK)

	log.Println("Rate limit pause manually reset via API")
}
