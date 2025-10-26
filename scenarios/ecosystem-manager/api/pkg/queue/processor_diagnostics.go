package queue

import (
	"fmt"
	"log"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/slices"
	"github.com/ecosystem-manager/api/pkg/settings"
)

// ResumeDiagnostics summarizes potential blockers before resuming the queue processor.
type ResumeDiagnostics struct {
	NeedsConfirmation        bool     `json:"needs_confirmation"`
	RateLimitPaused          bool     `json:"rate_limit_paused"`
	RateLimitResumeAt        string   `json:"rate_limit_resume_at,omitempty"`
	ActiveAgentTaskIDs       []string `json:"active_agent_task_ids"`
	ActiveAgentTags          []string `json:"active_agent_tags"`
	InternalExecutingTaskIDs []string `json:"internal_executing_task_ids"`
	OrphanInProgressTaskIDs  []string `json:"orphan_in_progress_task_ids"`
	Notes                    []string `json:"notes,omitempty"`
}

// ResumeResetSummary records the recovery work performed before resuming processing.
type ResumeResetSummary struct {
	RateLimitCleared    bool     `json:"rate_limit_cleared"`
	AgentsStopped       []string `json:"agents_stopped"`
	ProcessesTerminated []string `json:"processes_terminated"`
	TasksMovedToPending []string `json:"tasks_moved_to_pending"`
	ExecutionsCleared   int      `json:"executions_cleared"`
	Notes               []string `json:"notes,omitempty"`
	ActionsTaken        int      `json:"actions_taken"`
}

// GetResumeDiagnostics reports blockers that will be cleared when resuming the processor
func (qp *Processor) GetResumeDiagnostics() ResumeDiagnostics {
	diag := ResumeDiagnostics{}
	if qp == nil {
		return diag
	}

	rateLimitPaused, pauseUntil := qp.IsRateLimitPaused()
	if rateLimitPaused {
		diag.RateLimitPaused = true
		diag.RateLimitResumeAt = pauseUntil.Format(time.RFC3339)
	}

	internalRunning := qp.getInternalRunningTaskIDs()
	diag.InternalExecutingTaskIDs = slices.SortedKeys(internalRunning)

	externalActive := qp.getExternalActiveTaskIDs()
	diag.ActiveAgentTaskIDs = slices.SortedKeys(externalActive)
	if len(diag.ActiveAgentTaskIDs) > 0 {
		tags := make([]string, 0, len(diag.ActiveAgentTaskIDs))
		for _, id := range diag.ActiveAgentTaskIDs {
			tags = append(tags, makeAgentTag(id))
		}
		diag.ActiveAgentTags = tags
	}

	if qp.storage != nil {
		inProgressTasks, err := qp.storage.GetQueueItems("in-progress")
		if err != nil {
			diag.Notes = append(diag.Notes, fmt.Sprintf("failed to inspect in-progress queue: %v", err))
		} else {
			for _, task := range inProgressTasks {
				if _, running := internalRunning[task.ID]; running {
					continue
				}
				if _, active := externalActive[task.ID]; active {
					continue
				}
				diag.OrphanInProgressTaskIDs = append(diag.OrphanInProgressTaskIDs, task.ID)
			}
		}
	} else {
		diag.Notes = append(diag.Notes, "task storage unavailable for diagnostics")
	}

	diag.OrphanInProgressTaskIDs = slices.DedupeAndSort(diag.OrphanInProgressTaskIDs)
	diag.NeedsConfirmation = diag.RateLimitPaused || len(diag.ActiveAgentTaskIDs) > 0 || len(diag.InternalExecutingTaskIDs) > 0 || len(diag.OrphanInProgressTaskIDs) > 0

	return diag
}

// ResetForResume performs recovery work (terminating processes, clearing rate
// limits, and reconciling queue state) so that resuming processing starts from a
// clean slate.
func (qp *Processor) ResetForResume() ResumeResetSummary {
	summary := ResumeResetSummary{}
	if qp == nil {
		return summary
	}

	rateLimitPaused, pauseUntil := qp.IsRateLimitPaused()
	if rateLimitPaused {
		qp.ResetRateLimitPause()
		summary.RateLimitCleared = true
		summary.ActionsTaken++
		summary.Notes = append(summary.Notes, fmt.Sprintf("rate limit pause cleared (was until %s)", pauseUntil.Format(time.RFC3339)))
	}

	internalRunning := qp.getInternalRunningTaskIDs()
	internalIDs := slices.SortedKeys(internalRunning)
	if len(internalIDs) > 0 {
		// Terminate all internal executions
		for _, id := range internalIDs {
			if err := qp.TerminateRunningProcess(id); err != nil {
				log.Printf("Warning: failed to terminate task %s during reset: %v", id, err)
			}
		}
		summary.ProcessesTerminated = append(summary.ProcessesTerminated, internalIDs...)
		summary.ExecutionsCleared += len(internalIDs)
		summary.ActionsTaken++
	}

	externalActive := qp.getExternalActiveTaskIDs()
	if len(externalActive) > 0 {
		agentTags := make([]string, 0, len(externalActive))
		for _, taskID := range slices.SortedKeys(externalActive) {
			agentTag := makeAgentTag(taskID)
			agentTags = append(agentTags, agentTag)
			if err := qp.stopClaudeAgent(agentTag, 0); err != nil {
				log.Printf("Warning: failed to stop agent %s during resume: %v", agentTag, err)
			}
		}
		if err := qp.cleanupClaudeAgentRegistry(); err != nil {
			log.Printf("Warning: cleanup failed during resume: %v", err)
		}
		summary.AgentsStopped = append(summary.AgentsStopped, agentTags...)
		summary.ActionsTaken++
	}

	// After terminating agents/processes, reconcile any in-progress tasks back to pending
	externalAfter := qp.getExternalActiveTaskIDs()
	moved := qp.reconcileInProgressTasks(externalAfter, make(map[string]struct{}))
	if len(moved) > 0 {
		summary.TasksMovedToPending = slices.DedupeAndSort(moved)
		summary.ActionsTaken++
	}

	summary.ProcessesTerminated = slices.DedupeAndSort(summary.ProcessesTerminated)
	summary.AgentsStopped = slices.DedupeAndSort(summary.AgentsStopped)

	return summary
}

// ResumeWithReset resumes queue processing after performing safety recovery steps.
func (qp *Processor) ResumeWithReset() ResumeResetSummary {
	summary := qp.ResetForResume()

	qp.mu.Lock()
	defer qp.mu.Unlock()

	if !qp.isRunning {
		qp.isRunning = true
		go qp.processLoop()
		log.Println("Queue processor started from Resume()")
	}

	qp.isPaused = false
	log.Println("Queue processor resumed from maintenance")

	return summary
}

// GetQueueStatus returns current queue processor status and metrics
func (qp *Processor) GetQueueStatus() map[string]any {
	// Get current maintenance state
	qp.mu.Lock()
	isPaused := qp.isPaused
	isRunning := qp.isRunning
	qp.mu.Unlock()

	// Check rate limit pause status
	rateLimitPaused, pauseUntil := qp.IsRateLimitPaused()
	var rateLimitInfo map[string]any
	if rateLimitPaused {
		remaining := pauseUntil.Sub(time.Now())
		rateLimitInfo = map[string]any{
			"paused":         true,
			"pause_until":    pauseUntil.Format(time.RFC3339),
			"remaining_secs": int(remaining.Seconds()),
		}
	} else {
		rateLimitInfo = map[string]any{
			"paused": false,
		}
	}

	// Count tasks by status
	inProgressTasks, _ := qp.storage.GetQueueItems("in-progress")
	pendingTasks, _ := qp.storage.GetQueueItems("pending")
	completedTasks, _ := qp.storage.GetQueueItems("completed")
	failedTasks, _ := qp.storage.GetQueueItems("failed")
	completedFinalizedTasks, _ := qp.storage.GetQueueItems("completed-finalized")
	failedBlockedTasks, _ := qp.storage.GetQueueItems("failed-blocked")
	archivedTasks, _ := qp.storage.GetQueueItems("archived")
	reviewTasks, _ := qp.storage.GetQueueItems("review")

	internalRunning := qp.getInternalRunningTaskIDs()
	externalActive := qp.getExternalActiveTaskIDs()

	executingCount := len(internalRunning)
	for taskID := range externalActive {
		if _, tracked := internalRunning[taskID]; !tracked {
			executingCount++
		}
	}

	// Count ready-to-execute tasks in in-progress
	readyInProgress := 0
	for _, task := range inProgressTasks {
		if _, tracked := internalRunning[task.ID]; tracked {
			continue
		}
		if _, active := externalActive[task.ID]; active {
			continue
		}
		readyInProgress++
	}

	// Get maxConcurrent and refresh interval from settings
	currentSettings := settings.GetSettings()
	maxConcurrent := currentSettings.Slots
	availableSlots := maxConcurrent - executingCount
	if availableSlots < 0 {
		availableSlots = 0
	}

	// Check if processor should be active (both internal state and settings)
	settingsActive := currentSettings.Active
	processorActive := !isPaused && isRunning && !rateLimitPaused && settingsActive

	lastProcessed := qp.getLastProcessed()
	var lastProcessedAt any
	if !lastProcessed.IsZero() {
		lastProcessedAt = lastProcessed.Format(time.RFC3339)
	}

	return map[string]any{
		"processor_active":          processorActive,
		"settings_active":           settingsActive,
		"maintenance_state":         map[bool]string{true: "inactive", false: "active"}[isPaused],
		"rate_limit_info":           rateLimitInfo,
		"max_concurrent":            maxConcurrent,
		"executing_count":           executingCount,
		"running_count":             executingCount,
		"available_slots":           availableSlots,
		"pending_count":             len(pendingTasks),
		"in_progress_count":         len(inProgressTasks),
		"completed_count":           len(completedTasks),
		"failed_count":              len(failedTasks),
		"completed_finalized_count": len(completedFinalizedTasks),
		"failed_blocked_count":      len(failedBlockedTasks),
		"archived_count":            len(archivedTasks),
		"review_count":              len(reviewTasks),
		"ready_in_progress":         readyInProgress,
		"refresh_interval":          currentSettings.RefreshInterval, // from settings
		"processor_running":         processorActive,
		"timestamp":                 time.Now().Unix(),
		"last_processed_at":         lastProcessedAt,
	}
}

// setLastProcessed updates the timestamp of the last queue processing cycle
func (qp *Processor) setLastProcessed(t time.Time) {
	qp.lastProcessedMu.Lock()
	qp.lastProcessedAt = t
	qp.lastProcessedMu.Unlock()
}

// getLastProcessed retrieves the timestamp of the last queue processing cycle
func (qp *Processor) getLastProcessed() time.Time {
	qp.lastProcessedMu.RLock()
	defer qp.lastProcessedMu.RUnlock()
	return qp.lastProcessedAt
}
