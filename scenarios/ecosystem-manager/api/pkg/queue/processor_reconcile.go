package queue

import (
	"log"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/systemlog"
)

// initialInProgressReconcile reconciles stale in-progress tasks left from previous runs
func (qp *Processor) initialInProgressReconcile() {
	// Allow the system to finish startup before running reconciliation
	time.Sleep(InitialReconcileDelay)

	external := qp.getExternalActiveTaskIDs()
	internal := qp.getInternalRunningTaskIDs()
	moved := qp.reconcileInProgressTasks(external, internal)
	if len(moved) > 0 {
		log.Printf("Initial reconciliation recovered %d stale tasks from previous run", len(moved))
		systemlog.Infof("Initial reconciliation recovered %d stale tasks", len(moved))
	}
}

// reconcileInProgressTasks moves orphaned in-progress tasks back to pending
func (qp *Processor) reconcileInProgressTasks(externalActive, internalRunning map[string]struct{}) []string {
	moved := make([]string, 0)
	inProgressTasks, err := qp.storage.GetQueueItems("in-progress")
	if err != nil {
		log.Printf("Error checking in-progress tasks: %v", err)
		return moved
	}

	for _, task := range inProgressTasks {
		// Defensive: Skip tasks with invalid IDs (protects against storage corruption)
		if task.ID == "" {
			log.Printf("Warning: Found task with empty ID during reconciliation, skipping")
			systemlog.Warnf("Found task with empty ID during reconciliation, skipping")
			continue
		}

		// CRITICAL BUG FIX: Detect stale execution tracking
		// If task is tracked as running internally but agent is not actually active,
		// it means finalization failed - the agent completed but we failed to save the final state

		taskIsTracked := qp.IsTaskRunning(task.ID)
		_, agentIsActive := externalActive[task.ID]

		// Case 1: Task is tracked internally AND agent is active externally
		// This is a legitimately running task - skip reconciliation
		if taskIsTracked && agentIsActive {
			continue
		}

		// Case 2: Task is tracked internally but agent is NOT active
		// This means finalization failed - the task finished but we couldn't persist the status
		// Unregister it from execution tracking and fall through to reconciliation
		if taskIsTracked && !agentIsActive {
			log.Printf("⚠️  WARNING: Task %s tracked as running but agent is inactive - finalization failure detected", task.ID)
			systemlog.Warnf("Detected finalization failure for task %s - cleaning up execution registry", task.ID)
			qp.unregisterExecution(task.ID)
			// Fall through to reconciliation below
		}

		// Case 3: Task is NOT tracked but agent IS active externally
		// This is an orphaned agent from a previous crash/restart - skip for now as agent may still be working
		if !taskIsTracked && agentIsActive {
			continue
		}

		// Case 4: Task is NOT tracked and agent is NOT active
		// This is a stale in-progress task that needs reconciliation
		// Fall through to reconciliation below

		agentIdentifier := makeAgentTag(task.ID)
		qp.stopClaudeAgent(agentIdentifier, 0)
		systemlog.Warnf("Detected orphan in-progress task %s; relocating to pending", task.ID)
		if _, _, err := qp.storage.MoveTaskTo(task.ID, "pending"); err != nil {
			log.Printf("Failed to move orphan task %s back to pending: %v", task.ID, err)
			systemlog.Errorf("Failed to move orphan task %s back to pending: %v", task.ID, err)
			continue
		}

		task.Status = "pending"
		task.CurrentPhase = ""
		task.StartedAt = ""
		task.CompletedAt = ""
		task.Results = nil
		task.UpdatedAt = timeutil.NowRFC3339()
		// Use SkipCleanup since MoveTaskTo already cleaned up duplicates
		if err := qp.storage.SaveQueueItemSkipCleanup(task, "pending"); err != nil {
			log.Printf("Failed to persist orphan task %s after move: %v", task.ID, err)
		} else {
			qp.broadcastUpdate("task_status_changed", map[string]interface{}{
				"task_id":    task.ID,
				"old_status": "in-progress",
				"new_status": "pending",
				"task":       task,
			})
		}

		// Clear any cached logs for this run since it will be retried
		qp.ResetTaskLogs(task.ID)
		moved = append(moved, task.ID)
	}

	return moved
}
