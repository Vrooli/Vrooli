package queue

import (
	"log"
	"time"

	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// initialInProgressReconcile reconciles stale in-progress tasks left from previous runs
func (qp *Processor) initialInProgressReconcile() {
	// Allow the system to finish startup before running reconciliation
	select {
	case <-time.After(scaleDuration(InitialReconcileDelay)):
	case <-qp.ctx.Done():
		return
	}

	external := qp.getExternalActiveTaskIDs()
	internal := qp.getInternalRunningTaskIDs()
	moved := qp.reconcileInProgressTasks(external, internal)
	if len(moved) > 0 {
		log.Printf("Initial reconciliation recovered %d stale tasks from previous run", len(moved))
		systemlog.Infof("Initial reconciliation recovered %d stale tasks", len(moved))
	}
}

// reconcileInProgressTasks moves orphaned in-progress tasks back to pending.
// With agent-manager integration, we rely on internal tracking only.
// External agent state is managed by agent-manager.
func (qp *Processor) reconcileInProgressTasks(externalActive, internalRunning map[string]struct{}) []string {
	moved := make([]string, 0)
	inProgressTasks, err := qp.storage.GetQueueItems("in-progress")
	if err != nil {
		log.Printf("Error checking in-progress tasks: %v", err)
		return moved
	}

	lc := tasks.Lifecycle{Store: qp.storage}

	for _, task := range inProgressTasks {
		// Defensive: Skip tasks with invalid IDs
		if task.ID == "" {
			log.Printf("Warning: Found task with empty ID during reconciliation, skipping")
			systemlog.Warnf("Found task with empty ID during reconciliation, skipping")
			continue
		}

		taskIsTracked := qp.IsTaskRunning(task.ID)

		// Case 1: Task is tracked internally - check for timeout
		if taskIsTracked {
			exec, _ := qp.getExecution(task.ID)
			if exec != nil {
				// Check if timed out
				if exec.isTimedOut() {
					log.Printf("Reconciliation: Task %s timed out - stopping via agent-manager", task.ID)
					systemlog.Warnf("Reconciliation detected timed-out task %s", task.ID)

					// Stop via agent-manager
					if err := qp.stopRunViaAgentManager(task.ID); err != nil {
						log.Printf("Warning: failed to stop timed-out task %s: %v", task.ID, err)
					}
					qp.unregisterExecution(task.ID)
					// Fall through to move to pending
				} else if time.Since(exec.started) < 2*time.Minute {
					// Grace period for recently started tasks
					log.Printf("Task %s started %s ago; deferring reconciliation", task.ID, time.Since(exec.started).Round(time.Second))
					continue
				} else {
					// Actively running, skip
					continue
				}
			} else {
				// Tracked but no exec record - cleanup stale tracking
				qp.unregisterExecution(task.ID)
			}
		}

		// Case 2: Task is NOT tracked - stale in-progress task
		// Move back to pending
		systemlog.Warnf("Detected orphan in-progress task %s; relocating to pending", task.ID)

		outcome, err := lc.ApplyTransition(tasks.TransitionRequest{
			TaskID:   task.ID,
			ToStatus: tasks.StatusPending,
			TransitionContext: tasks.TransitionContext{
				Intent: tasks.IntentReconcile, // reconcile should not be blocked by auto-requeue locks
				Now:    time.Now,
			},
		})
		if err != nil {
			log.Printf("Failed to move orphan task %s back to pending: %v", task.ID, err)
			systemlog.Errorf("Failed to move orphan task %s back to pending: %v", task.ID, err)
			continue
		}

		task = *outcome.Task
		qp.broadcastUpdate("task_status_changed", map[string]any{
			"task_id":    task.ID,
			"old_status": outcome.From,
			"new_status": tasks.StatusPending,
			"task":       task,
		})

		// Clear any cached logs for this run since it will be retried
		qp.ResetTaskLogs(task.ID)
		moved = append(moved, task.ID)
	}

	return moved
}
