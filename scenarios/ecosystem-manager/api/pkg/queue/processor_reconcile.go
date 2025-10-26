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

		// PERFORMANCE: Compute agent tag once and reuse
		agentTag := makeAgentTag(task.ID)

		// CRITICAL BUG FIX: Detect stale execution tracking
		// If task is tracked as running internally but agent is not actually active,
		// it means finalization failed - the agent completed but we failed to save the final state

		taskIsTracked := qp.IsTaskRunning(task.ID)
		_, agentIsActive := externalActive[task.ID]

		// Case 1: Task is tracked internally AND agent is active externally
		// This is a legitimately running task - BUT check if it timed out!
		if taskIsTracked && agentIsActive {
			// CRITICAL: Check if task has exceeded timeout despite being "active"
			// This catches cases where context timeout failed or cleanup is stuck
			exec, _ := qp.getExecution(task.ID)
			if exec != nil && exec.isTimedOut() {
				log.Printf("⚠️  WARNING: Task %s is timed out but still tracked+active - forcing cleanup", task.ID)
				systemlog.Warnf("Reconciliation detected timed-out task %s still running - forcing cleanup", task.ID)

				// Force removal of zombie agent
				if err := qp.forceRemoveAgentWithRetry(agentTag, exec.pid()); err != nil {
					log.Printf("ERROR: Reconciliation failed to remove timed-out agent %s: %v", agentTag, err)
					systemlog.Errorf("Failed to remove timed-out zombie agent %s: %v", agentTag, err)
					// Keep it tracked - watchdog will handle it
					continue
				}

				// Agent removed, unregister and fall through to reconciliation
				qp.unregisterExecution(task.ID)
				log.Printf("✅ Reconciliation successfully cleaned up timed-out task %s", task.ID)
				// Fall through to move task back to pending or handle appropriately
			} else {
				// Legitimately running task, skip reconciliation
				continue
			}
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
		// This could be:
		// a) Legitimately working orphaned agent from crash/restart
		// b) Zombie agent from failed cleanup (especially MAX_TURNS cases)
		// Be conservative: only clean up if we can PROVE it's a zombie
		if !taskIsTracked && agentIsActive {
			// Try to find PIDs for this agent
			pids := qp.agentProcessPIDs(agentTag)

			// Only intervene if we found PIDs but none are alive
			// If we found NO PIDs, it might be a test scenario or the process
			// hasn't been indexed yet - be conservative and skip
			if len(pids) > 0 {
				hasActiveProcess := false
				for _, pid := range pids {
					if qp.isProcessAlive(pid) {
						hasActiveProcess = true
						break
					}
				}

				// Definitive zombie: found PIDs but all are dead
				if !hasActiveProcess {
					log.Printf("⚠️  WARNING: Task %s has zombie agent with dead processes - cleaning up", task.ID)
					systemlog.Warnf("Detected zombie agent %s with dead processes - forcing cleanup", agentTag)

					// Force remove the zombie agent
					if err := qp.forceRemoveAgentWithRetry(agentTag, 0); err != nil {
						log.Printf("ERROR: Failed to remove zombie agent %s: %v", agentTag, err)
						systemlog.Errorf("Failed to remove zombie agent %s: %v - task will remain stuck", agentTag, err)
					}
					// Fall through to reconciliation below to move task back to pending
				} else {
					// Process is alive - legitimate orphaned work, skip
					log.Printf("Task %s has active agent with live process - skipping reconciliation", task.ID)
					continue
				}
			} else {
				// No PIDs found - could be test or very new process, be conservative
				log.Printf("Task %s has agent in registry but no PIDs found - skipping (might be legitimate)", task.ID)
				continue
			}
		}

		// Case 4: Task is NOT tracked and agent is NOT active
		// This is a stale in-progress task that needs reconciliation
		// Fall through to reconciliation below

		if err := qp.stopClaudeAgent(agentTag, 0); err != nil {
			log.Printf("Warning: failed to stop agent %s during reconciliation: %v", agentTag, err)
		}
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
			qp.broadcastUpdate("task_status_changed", map[string]any{
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
