package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ecosystem-manager/api/pkg/agentmanager"
	"github.com/ecosystem-manager/api/pkg/systemlog"
)

// TimeoutWatchdog monitors executions and terminates tasks that exceed their timeout.
// It provides defense-in-depth backup enforcement if context.WithTimeout fails
// or if cleanup/finalization failures leave tasks stuck in the executions map.
type TimeoutWatchdog struct {
	registry *ExecutionRegistry
	agentSvc *agentmanager.AgentService
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewTimeoutWatchdog creates a new timeout watchdog.
func NewTimeoutWatchdog(registry *ExecutionRegistry, agentSvc *agentmanager.AgentService) *TimeoutWatchdog {
	ctx, cancel := context.WithCancel(context.Background())
	return &TimeoutWatchdog{
		registry: registry,
		agentSvc: agentSvc,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the watchdog loop in a background goroutine.
func (tw *TimeoutWatchdog) Start() {
	go tw.watchLoop()
}

// Stop terminates the watchdog loop.
func (tw *TimeoutWatchdog) Stop() {
	tw.cancel()
}

// watchLoop periodically checks for timed-out executions.
func (tw *TimeoutWatchdog) watchLoop() {
	ticker := time.NewTicker(scaleDuration(TimeoutWatchdogInterval))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tw.enforceTimeouts()
		case <-tw.ctx.Done():
			return
		}
	}
}

// enforceTimeouts checks all tracked executions and forcibly terminates timed-out tasks.
func (tw *TimeoutWatchdog) enforceTimeouts() {
	timedOutTasks := tw.registry.GetTimedOutExecutions()

	if len(timedOutTasks) == 0 {
		return
	}

	log.Printf("⏰ WATCHDOG: Detected %d timed-out tasks still in executions, forcing termination", len(timedOutTasks))
	systemlog.Warnf("Timeout watchdog detected %d stuck tasks - forcing termination", len(timedOutTasks))

	for _, task := range timedOutTasks {
		tw.forceTerminateTimedOutTask(task.TaskID, task.AgentTag, task.PID)
	}
}

// forceTerminateTimedOutTask forcibly terminates a task that exceeded its timeout.
func (tw *TimeoutWatchdog) forceTerminateTimedOutTask(taskID, agentTag string, pid int) {
	log.Printf("WATCHDOG: Force terminating timed-out task %s (agent: %s)", taskID, agentTag)
	systemlog.Warnf("Timeout watchdog forcing termination of task %s", taskID)

	// Stop via agent-manager (primary path)
	if err := tw.stopRunViaAgentManager(taskID); err != nil {
		log.Printf("WARNING: Watchdog agent-manager stop failed for task %s: %v", taskID, err)
		systemlog.Warnf("Watchdog agent-manager stop failed for %s: %v", taskID, err)
	}

	// Unregister execution
	tw.registry.UnregisterExecution(taskID)
	log.Printf("WATCHDOG: Terminated and unregistered timed-out task %s", taskID)
	systemlog.Infof("Watchdog successfully terminated timed-out task %s", taskID)
}

// stopRunViaAgentManager stops a run using agent-manager by task ID.
func (tw *TimeoutWatchdog) stopRunViaAgentManager(taskID string) error {
	runID := tw.registry.GetRunIDForTask(taskID)
	if runID == "" {
		return nil // No run tracked, nothing to stop
	}

	if tw.agentSvc == nil {
		return fmt.Errorf("agent service not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := tw.agentSvc.StopRun(ctx, runID); err != nil {
		return fmt.Errorf("stop run %s: %w", runID, err)
	}
	return nil
}
