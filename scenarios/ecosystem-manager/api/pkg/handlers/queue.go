package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

// QueueHandlers contains handlers for queue-related endpoints
type QueueHandlers struct {
	processor *queue.Processor
	wsManager *websocket.Manager
	storage   *tasks.Storage
	coord     *tasks.Coordinator
}

// NewQueueHandlers creates a new queue handlers instance
func NewQueueHandlers(processor *queue.Processor, wsManager *websocket.Manager, storage *tasks.Storage, coord *tasks.Coordinator) *QueueHandlers {
	return &QueueHandlers{
		processor: processor,
		wsManager: wsManager,
		storage:   storage,
		coord:     coord,
	}
}

// applyTransitionEffects executes lifecycle side effects when coordinator is unavailable in handlers.
func (h *QueueHandlers) applyTransitionEffects(outcome *tasks.TransitionOutcome, task *tasks.TaskItem, taskID string) {
	if outcome == nil {
		return
	}
	if h.wsManager != nil && task != nil {
		h.wsManager.BroadcastUpdate("task_status_changed", map[string]any{
			"task_id":    taskID,
			"old_status": outcome.From,
			"new_status": task.Status,
			"task":       task,
		})
	}
	if h.processor == nil {
		return
	}
	if outcome.Effects.TerminateProcess {
		if err := h.processor.TerminateRunningProcess(taskID); err != nil {
			systemlog.Warnf("Failed to terminate process for task %s after lifecycle transition: %v", taskID, err)
		}
	}
	if outcome.Effects.StartIfSlotAvailable {
		if err := h.processor.StartTaskIfSlotAvailable(taskID); err != nil {
			systemlog.Warnf("Failed to opportunistically restart task %s after transition: %v", taskID, err)
		}
	}
	if outcome.Effects.ForceStart {
		if err := h.processor.ForceStartTask(taskID, true); err != nil {
			systemlog.Warnf("Failed to force-start task %s after transition: %v", taskID, err)
		}
	}
	if outcome.Effects.WakeProcessorAfterSave {
		h.processor.Wake()
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

	task, _, err := h.storage.GetTaskByID(request.TaskID)
	if err != nil || task == nil {
		writeError(w, "Task not found", http.StatusNotFound)
		return
	}

	var outcome *tasks.TransitionOutcome
	if h.coord != nil {
		var err error
		task, outcome, err = h.coord.ApplyTransition(tasks.TransitionRequest{
			TaskID:   request.TaskID,
			ToStatus: tasks.StatusPending,
			TransitionContext: tasks.TransitionContext{
				Intent: tasks.IntentManual,
			},
		}, tasks.ApplyOptions{
			BroadcastEvent: "task_status_changed",
			ForceResave:    true,
		})
		if err != nil {
			writeError(w, fmt.Sprintf("Failed to move task to pending: %v", err), http.StatusConflict)
			return
		}
	} else {
		// Fallback to direct lifecycle if coordinator unavailable (should not happen in normal flow).
		lc := tasks.Lifecycle{Store: h.storage}
		var err error
		outcome, err = lc.ApplyTransition(tasks.TransitionRequest{
			TaskID:   request.TaskID,
			ToStatus: tasks.StatusPending,
			TransitionContext: tasks.TransitionContext{
				Intent: tasks.IntentReconcile,
			},
		})
		if err != nil {
			writeError(w, fmt.Sprintf("Failed to move task to pending: %v", err), http.StatusConflict)
			return
		}
		task = outcome.Task
		h.applyTransitionEffects(outcome, task, request.TaskID)
	}

	if h.processor != nil {
		h.processor.ResetTaskLogs(request.TaskID)
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

	// Keep settings in sync with the processor lifecycle so the play/pause button reflects reality.
	currentSettings := settings.GetSettings()
	if currentSettings.Active {
		currentSettings.Active = false
		settings.UpdateSettings(currentSettings)
		h.wsManager.BroadcastUpdate("settings_updated", currentSettings)
	}

	response := map[string]any{
		"success": true,
		"message": "Queue processor stopped",
		"status":  h.processor.GetQueueStatus(),
	}

	writeJSON(w, response, http.StatusOK)

	log.Println("Queue processor stopped via API")
	systemlog.Info("Queue processor stopped via API request")
}

// StartQueueProcessorHandler starts the queue processor
func (h *QueueHandlers) StartQueueProcessorHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure settings are marked active so the processor actually runs
	currentSettings := settings.GetSettings()
	if !currentSettings.Active {
		currentSettings.Active = true
		settings.UpdateSettings(currentSettings)
		h.wsManager.BroadcastUpdate("settings_updated", currentSettings)
	}

	status := h.processor.GetQueueStatus()
	processorActive, _ := status["processor_active"].(bool)

	// Resume from maintenance mode and clear stale state before starting when inactive; otherwise just ensure the loop is running.
	var resumeSummary queue.ResumeResetSummary
	if processorActive {
		h.processor.Start()
	} else {
		running := h.processor.GetRunningProcessesInfo()
		if len(running) > 0 {
			h.processor.ResumeWithoutReset()
		} else {
			resumeSummary = h.processor.ResumeWithReset()
		}
	}

	status = h.processor.GetQueueStatus()

	response := map[string]any{
		"success": true,
		"message": "Queue processor started",
		"status":  status,
	}
	if resumeSummary.ActionsTaken > 0 {
		response["resume_reset_summary"] = resumeSummary
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
