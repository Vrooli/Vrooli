package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/paths"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

type taskExecution struct {
	taskID    string
	agentTag  string
	cmd       *exec.Cmd
	started   time.Time
	timeoutAt time.Time // When this execution will timeout
	timedOut  bool      // Whether timeout has already occurred
}

func (te *taskExecution) pid() int {
	if te == nil || te.cmd == nil || te.cmd.Process == nil {
		return 0
	}
	return te.cmd.Process.Pid
}

// isTimedOut returns true if the execution has exceeded its timeout
func (te *taskExecution) isTimedOut() bool {
	if te == nil {
		return false
	}
	if te.timedOut {
		return true
	}
	if !te.timeoutAt.IsZero() && time.Now().After(te.timeoutAt) {
		return true
	}
	return false
}

// Processor manages automated queue processing
type Processor struct {
	mu          sync.Mutex
	isRunning   bool
	isPaused    bool // Added for maintenance state awareness
	stopChannel chan bool
	wakeCh      chan struct{}
	storage     *tasks.Storage
	assembler   *prompts.Assembler

	// Live task executions keyed by task ID
	executions   map[string]*taskExecution
	executionsMu sync.RWMutex

	// Root of the Vrooli workspace for resource CLI commands
	vrooliRoot string

	// Scenario root (parent of queue dir) for local artifact writes
	scenarioRoot string

	// Folder where we persist per-task execution logs
	taskLogsDir string

	// Broadcast channel for WebSocket updates
	broadcast chan<- any

	// Rate limit pause management
	rateLimitPaused bool
	pauseUntil      time.Time
	pauseMutex      sync.Mutex

	// Task log buffers for streaming execution logs
	taskLogs      map[string]*TaskLogBuffer
	taskLogsMutex sync.RWMutex

	// Bookkeeping for queue activity
	lastProcessedMu sync.RWMutex
	lastProcessedAt time.Time

	// Execution history cache (simple time-based invalidation)
	executionHistoryCache     []ExecutionHistory
	executionHistoryCacheTime time.Time
	executionHistoryCacheMu   sync.RWMutex

	// Auto Steer integration for multi-dimensional improvement
	autoSteerIntegration *AutoSteerIntegration

	// Recycler to trigger recycling on task completion/failure
	recycler *recycler.Recycler

	// Centralized task coordinator for lifecycle + side effects orchestration.
	coord *tasks.Coordinator

	// Lifecycle control for background workers
	ctx    context.Context
	cancel context.CancelFunc
}

// slotSnapshot captures concurrency accounting for scheduling decisions.
type slotSnapshot struct {
	Slots     int
	Running   int
	Available int
}

// NewProcessor creates a new queue processor
func NewProcessor(storage *tasks.Storage, assembler *prompts.Assembler, broadcast chan<- any, recycler *recycler.Recycler) *Processor {
	ctx, cancel := context.WithCancel(context.Background())
	processor := &Processor{
		stopChannel: make(chan bool),
		wakeCh:      make(chan struct{}, 1),
		storage:     storage,
		assembler:   assembler,
		executions:  make(map[string]*taskExecution),
		broadcast:   broadcast,
		taskLogs:    make(map[string]*TaskLogBuffer),
		recycler:    recycler,
		ctx:         ctx,
		cancel:      cancel,
	}

	processor.vrooliRoot = paths.DetectVrooliRoot()
	processor.scenarioRoot = filepath.Dir(storage.QueueDir)
	processor.taskLogsDir = filepath.Join(storage.QueueDir, "..", "logs", "task-runs")
	if err := os.MkdirAll(processor.taskLogsDir, 0755); err != nil {
		log.Printf("Warning: unable to create task logs directory %s: %v", processor.taskLogsDir, err)
	}

	// Clean up orphaned processes
	processor.cleanupOrphanedProcesses()

	// Clean up old temporary prompt files
	go processor.cleanupOldPromptFiles()

	// Reconcile any stale in-progress tasks left behind from previous runs
	go processor.initialInProgressReconcile()

	// Start timeout enforcement watchdog (defense-in-depth)
	go processor.timeoutEnforcementWatchdog()

	return processor
}

// SetCoordinator injects a central coordinator for lifecycle-aware transitions.
func (qp *Processor) SetCoordinator(coord *tasks.Coordinator) {
	qp.coord = coord
}

// startTaskExecution moves a task into in-progress (if needed) and launches execution.
func (qp *Processor) startTaskExecution(task *tasks.TaskItem, currentStatus string, externalActive map[string]struct{}, ctx tasks.TransitionContext) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	previousStatus := currentStatus

	// Use coordinator when available to enforce a single rule path; skip runtime effects to avoid recursion.
	if qp.coord != nil && currentStatus != tasks.StatusInProgress {
		updated, outcome, err := qp.coord.ApplyTransition(tasks.TransitionRequest{
			TaskID:            task.ID,
			ToStatus:          tasks.StatusInProgress,
			TransitionContext: ctx,
		}, tasks.ApplyOptions{
			BroadcastEvent:     "task_status_changed",
			SkipRuntimeEffects: true,
			ForceResave:        true,
		})
		if err != nil {
			return fmt.Errorf("start pending task %s: %w", task.ID, err)
		}
		if updated != nil {
			task = updated
		}
		if outcome != nil {
			previousStatus = outcome.From
		}
	} else if currentStatus != "in-progress" {
		lc := tasks.Lifecycle{Store: qp.storage}
		var err error
		task, err = lc.StartPending(task.ID, ctx)
		if err != nil {
			return fmt.Errorf("start pending task %s: %w", task.ID, err)
		}
		previousStatus = "pending"
	}

	agentIdentifier := makeAgentTag(task.ID)
	if externalActive != nil {
		if _, active := externalActive[task.ID]; active {
			log.Printf("Detected lingering Claude agent for task %s; attempting cleanup before restart", task.ID)
			if err := qp.stopClaudeAgent(agentIdentifier, 0); err != nil {
				log.Printf("Warning: failed to stop lingering agent %s: %v", agentIdentifier, err)
			}
			delete(externalActive, task.ID)
		}
	}

	// Reserve execution slot immediately so reconciliation won't recycle the task
	qp.reserveExecution(task.ID, agentIdentifier, time.Now())

	if qp.coord == nil {
		qp.broadcastUpdate("task_status_changed", map[string]any{
			"task_id":    task.ID,
			"old_status": previousStatus,
			"new_status": "in-progress",
			"task":       task,
		})
	}

	// Process the task asynchronously
	go qp.executeTask(*task)
	return nil
}

// SetAutoSteerIntegration sets the Auto Steer integration for the processor
// This must be called after the processor is created but before processing starts
func (qp *Processor) SetAutoSteerIntegration(integration *AutoSteerIntegration) {
	qp.autoSteerIntegration = integration
	log.Println("✅ Auto Steer integration configured for queue processor")
	systemlog.Info("Auto Steer integration enabled")
}

// AutoSteerIntegration returns the configured Auto Steer integration, if any.
func (qp *Processor) AutoSteerIntegration() *AutoSteerIntegration {
	return qp.autoSteerIntegration
}

// Start begins the queue processing loop
func (qp *Processor) Start() {
	qp.mu.Lock()
	defer qp.mu.Unlock()

	if qp.isRunning {
		log.Println("Queue processor already running")
		return
	}

	qp.isRunning = true
	go qp.processLoop()
	qp.Wake()
	log.Println("Queue processor started")
}

// Stop halts the queue processing loop
func (qp *Processor) Stop() {
	qp.mu.Lock()
	defer qp.mu.Unlock()

	if !qp.isRunning {
		return
	}

	qp.executionsMu.RLock()
	runningCount := len(qp.executions)
	qp.executionsMu.RUnlock()

	qp.stopChannel <- true
	qp.isRunning = false
	if runningCount > 0 {
		log.Printf("Queue processor stopped; allowing %d running task(s) to finish", runningCount)
	} else {
		log.Println("Queue processor stopped")
	}
}

// Shutdown stops background workers and should be called during full application teardown.
func (qp *Processor) Shutdown() {
	qp.cancel()
}

// Pause temporarily pauses queue processing (maintenance mode)
func (qp *Processor) Pause() {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	qp.isPaused = true
	log.Println("Queue processor paused for maintenance")
}

// Resume resumes queue processing from maintenance mode
func (qp *Processor) Resume() {
	// ResumeWithReset returns a summary struct, not an error
	// Intentionally ignoring the summary as this is a simple resume call
	_ = qp.ResumeWithReset()
}

// ForceStartTask starts a specific task immediately, bypassing slot limits when allowOverflow is true.
// This is used for manual moves to Active where user intent trumps concurrency guardrails.
func (qp *Processor) ForceStartTask(taskID string, allowOverflow bool) error {
	if qp == nil {
		return fmt.Errorf("processor unavailable")
	}
	if taskID == "" {
		return fmt.Errorf("task id required")
	}

	// Prevent duplicate launches
	if qp.IsTaskRunning(taskID) {
		return nil
	}

	task, status, err := qp.storage.GetTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("load task %s: %w", taskID, err)
	}
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Respect lock: tasks explicitly blocked from auto-requeue should not be force-started unless already in-progress.
	if status == "pending" && !task.ProcessorAutoRequeue && !allowOverflow {
		return fmt.Errorf("task %s auto-requeue disabled; cannot start", taskID)
	}

	externalActive := qp.getExternalActiveTaskIDs()
	return qp.startTaskExecution(task, status, externalActive, tasks.TransitionContext{
		Manual:        true,
		ForceOverride: allowOverflow,
	})
}

// StartTaskIfSlotAvailable starts a pending task immediately when capacity is available.
// It respects auto-requeue locks and will not overflow the configured slot count.
func (qp *Processor) StartTaskIfSlotAvailable(taskID string) error {
	if qp == nil {
		return fmt.Errorf("processor unavailable")
	}
	if taskID == "" {
		return fmt.Errorf("task id required")
	}

	qp.mu.Lock()
	isRunning := qp.isRunning
	qp.mu.Unlock()
	if !isRunning {
		return nil
	}

	internal := qp.getInternalRunningTaskIDs()
	external := qp.getExternalActiveTaskIDs()
	snap := qp.computeSlotSnapshot(internal, external)
	if snap.Available <= 0 {
		return nil
	}

	task, status, err := qp.storage.GetTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("load task %s: %w", taskID, err)
	}
	if task == nil || status != "pending" {
		return nil
	}
	if qp.IsTaskRunning(taskID) {
		return nil
	}

	return qp.startTaskExecution(task, status, external, tasks.TransitionContext{
		Manual:        true,
		ForceOverride: false,
	})
}

// Wake requests the processor to immediately attempt to fill available slots.
func (qp *Processor) Wake() {
	if qp == nil {
		return
	}
	select {
	case qp.wakeCh <- struct{}{}:
	default:
	}
}

// processLoop is the main queue processing loop
func (qp *Processor) processLoop() {
	// Safety backstop to recover from any missed wake signals
	safetyTicker := time.NewTicker(scaleDuration(ProcessLoopSafetyInterval))
	defer safetyTicker.Stop()

	for {
		select {
		case <-qp.stopChannel:
			return
		case <-qp.wakeCh:
			qp.ProcessQueue()
		case <-safetyTicker.C:
			qp.ProcessQueue()
		}
	}
}

// ProcessQueue processes pending tasks and manually moved in-progress tasks
func (qp *Processor) ProcessQueue() {
	// Check if paused (maintenance mode)
	qp.mu.Lock()
	isPaused := qp.isPaused
	qp.mu.Unlock()

	if isPaused {
		// Skip processing while in maintenance mode
		return
	}

	// CRITICAL: Also check settings active state
	if !settings.IsActive() {
		// Skip processing if not active in settings
		return
	}

	// Check if rate limit paused
	qp.pauseMutex.Lock()
	if qp.rateLimitPaused {
		if time.Now().Before(qp.pauseUntil) {
			remaining := time.Until(qp.pauseUntil)
			log.Printf("⏸️ Queue paused due to rate limit. Resuming in %v", remaining.Round(time.Second))
			qp.pauseMutex.Unlock()

			// Broadcast pause status
			qp.broadcastUpdate("rate_limit_pause", map[string]any{
				"paused":         true,
				"pause_until":    qp.pauseUntil.Format(time.RFC3339),
				"remaining_secs": int(remaining.Seconds()),
			})
			return
		} else {
			// Pause has expired, resume processing
			qp.rateLimitPaused = false
			qp.pauseUntil = time.Time{}
			log.Printf("✅ Rate limit pause expired. Resuming queue processing.")

			// Broadcast resume
			qp.broadcastUpdate("rate_limit_resume", map[string]any{
				"paused": false,
			})
		}
	}
	qp.pauseMutex.Unlock()

	qp.setLastProcessed(time.Now())

	externalActive := qp.getExternalActiveTaskIDs()
	internalRunning := qp.getInternalRunningTaskIDs()
	movedTasks := qp.reconcileInProgressTasks(externalActive, internalRunning)
	if len(movedTasks) > 0 {
		log.Printf("Reconciliation moved %d orphaned tasks back to pending", len(movedTasks))
	}

	// Backstop: clean up any duplicate task files left by manual moves or race conditions.
	if err := qp.storage.CleanupDuplicates(); err != nil {
		log.Printf("Warning: duplicate task cleanup failed: %v", err)
	}

	snap := qp.computeSlotSnapshot(internalRunning, externalActive)

	// Get pending tasks (re-fetch if we moved any orphans)
	pendingTasks, err := qp.storage.GetQueueItems("pending")
	if err != nil {
		log.Printf("Error getting pending tasks: %v", err)
		return
	}

	if len(pendingTasks) == 0 {
		return // No tasks to process
	}

	if snap.Available <= 0 {
		log.Printf("Queue processor: %d tasks already executing, %d available slots", snap.Running, snap.Available)
		return
	}

	// Sort tasks by priority (critical > high > medium > low)
	priorityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}

	for availableSlots := snap.Available; availableSlots > 0; availableSlots-- {
		// Find highest priority task from all ready tasks
		var selectedTask *tasks.TaskItem
		var selectedIdx int
		highestPriority := 0

		for i, task := range pendingTasks {
			if !task.ProcessorAutoRequeue {
				continue
			}
			priority := priorityOrder[task.Priority]
			if priority > highestPriority {
				highestPriority = priority
				selectedTask = &pendingTasks[i]
				selectedIdx = i
			}
		}

		if selectedTask == nil {
			return
		}

		log.Printf("Processing task: %s - %s (from pending)", selectedTask.ID, selectedTask.Title)
		systemlog.Debugf("Queue selecting %s from pending (priority %s)", selectedTask.ID, selectedTask.Priority)

		if err := qp.startTaskExecution(selectedTask, "pending", externalActive, tasks.TransitionContext{Manual: false, ForceOverride: false}); err != nil {
			log.Printf("Failed to start task %s: %v", selectedTask.ID, err)
			systemlog.Errorf("Failed to start task %s: %v", selectedTask.ID, err)
			return
		}

		snap.Running++

		// Remove the selected task from the local slice to avoid reselection in the same pass
		pendingTasks = append(pendingTasks[:selectedIdx], pendingTasks[selectedIdx+1:]...)
		if len(pendingTasks) == 0 {
			return
		}
	}
}

func (qp *Processor) finalizeTaskStatus(task *tasks.TaskItem, toStatus string) error {
	// Persist latest task payload to its current bucket so ApplyTransition sees updated fields (results, metadata).
	currentStatus := task.Status
	if strings.TrimSpace(currentStatus) == "" {
		if status, err := qp.storage.CurrentStatus(task.ID); err == nil {
			currentStatus = status
		}
	}
	if currentStatus == "" {
		currentStatus = toStatus
	}
	if err := qp.storage.SaveQueueItemSkipCleanup(*task, currentStatus); err != nil {
		log.Printf("Failed to persist task %s before finalize: %v", task.ID, err)
	}

	// Prefer coordinator for consistency and side effects; skip runtime to avoid recursion from inside processor.
	if qp.coord != nil {
		outcomeTask, outcome, err := qp.coord.ApplyTransition(tasks.TransitionRequest{
			TaskID:   task.ID,
			ToStatus: toStatus,
			TransitionContext: tasks.TransitionContext{
				Manual: false,
			},
		}, tasks.ApplyOptions{
			BroadcastEvent:     "task_status_changed",
			SkipRuntimeEffects: true,
			ForceResave:        true,
		})
		if err != nil {
			log.Printf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
			systemlog.Errorf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
			return err
		}
		if outcomeTask != nil {
			task = outcomeTask
		}
		if outcome != nil {
			fromStatus := outcome.From
			systemlog.Debugf("Task %s finalized: %s -> %s", task.ID, fromStatus, toStatus)
		}
	} else {
		lc := tasks.Lifecycle{Store: qp.storage}
		outcome, err := lc.ApplyTransition(tasks.TransitionRequest{
			TaskID:   task.ID,
			ToStatus: toStatus,
			TransitionContext: tasks.TransitionContext{
				Manual: false,
			},
		})
		if err != nil {
			log.Printf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
			systemlog.Errorf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
			return err
		}

		if outcome.Task != nil {
			task = outcome.Task
		}

		fromStatus := outcome.From
		systemlog.Debugf("Task %s finalized: %s -> %s", task.ID, fromStatus, toStatus)
		qp.broadcastUpdate("task_status_changed", map[string]any{
			"task_id":    task.ID,
			"old_status": fromStatus,
			"new_status": toStatus,
			"task":       task,
		})
	}

	if status, err := qp.storage.CurrentStatus(task.ID); err == nil {
		systemlog.Debugf("Task %s post-finalize location: %s", task.ID, status)
	} else {
		systemlog.Warnf("Task %s post-finalize location unknown: %v", task.ID, err)
	}

	// Trigger recycler for eligible completions/failures
	if qp.recycler != nil && (toStatus == "completed" || toStatus == "failed") && task.ProcessorAutoRequeue {
		qp.recycler.Enqueue(task.ID)
	}

	return nil
}

// cleanupOldPromptFiles removes temporary prompt files older than 24 hours from /tmp
// This prevents /tmp from filling up over hundreds of task executions
func (qp *Processor) cleanupOldPromptFiles() {
	pattern := filepath.Join(os.TempDir(), PromptFilePrefix+"*.txt")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("Warning: failed to glob temporary prompt files: %v", err)
		return
	}

	if len(matches) == 0 {
		return
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	removedCount := 0

	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				log.Printf("Warning: failed to remove old prompt file %s: %v", path, err)
			} else {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		log.Printf("Cleaned up %d temporary prompt files older than 24 hours", removedCount)
		systemlog.Infof("Cleaned up %d temporary prompt files from /tmp", removedCount)
	}
}

// timeoutEnforcementWatchdog monitors for tasks that have exceeded their timeout
// This provides defense-in-depth backup enforcement if context.WithTimeout fails
// or if cleanup/finalization failures leave tasks stuck in executions map
func (qp *Processor) timeoutEnforcementWatchdog() {
	ticker := time.NewTicker(scaleDuration(TimeoutWatchdogInterval))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			qp.enforceTimeouts()
		case <-qp.ctx.Done():
			return
		}
	}
}

// enforceTimeouts checks all tracked executions and forcibly terminates timed-out tasks
func (qp *Processor) enforceTimeouts() {
	qp.executionsMu.RLock()
	timedOutTasks := make([]struct {
		taskID   string
		agentTag string
		pid      int
	}, 0)

	for taskID, exec := range qp.executions {
		if exec.isTimedOut() {
			timedOutTasks = append(timedOutTasks, struct {
				taskID   string
				agentTag string
				pid      int
			}{
				taskID:   taskID,
				agentTag: exec.agentTag,
				pid:      exec.pid(),
			})
		}
	}
	qp.executionsMu.RUnlock()

	if len(timedOutTasks) == 0 {
		return
	}

	log.Printf("⏰ WATCHDOG: Detected %d timed-out tasks still in executions, forcing termination", len(timedOutTasks))
	systemlog.Warnf("Timeout watchdog detected %d stuck tasks - forcing termination", len(timedOutTasks))

	for _, task := range timedOutTasks {
		qp.forceTerminateTimedOutTask(task.taskID, task.agentTag, task.pid)
	}
}

// computeSlotSnapshot centralizes slot accounting for scheduler and manual starts.
func (qp *Processor) computeSlotSnapshot(internalRunning, externalActive map[string]struct{}) slotSnapshot {
	running := len(internalRunning)
	for taskID := range externalActive {
		if _, tracked := internalRunning[taskID]; tracked {
			continue
		}
		running++
	}

	slots := settings.GetSettings().Slots
	if slots <= 0 {
		slots = 1
	}
	available := slots - running
	if available < 0 {
		available = 0
	}

	return slotSnapshot{
		Slots:     slots,
		Running:   running,
		Available: available,
	}
}

// forceTerminateTimedOutTask forcibly terminates a task that exceeded its timeout
// This is called by the watchdog when it detects a task stuck in executions after timeout
func (qp *Processor) forceTerminateTimedOutTask(taskID, agentTag string, pid int) {
	log.Printf("⏰ WATCHDOG: Force terminating timed-out task %s (agent: %s, pid: %d)", taskID, agentTag, pid)
	systemlog.Warnf("Timeout watchdog forcing termination of task %s", taskID)

	// Resolve agent tag if not available
	if agentTag == "" {
		agentTag = makeAgentTag(taskID)
	}

	// Terminate agent using canonical termination function
	if err := qp.terminateAgent(taskID, agentTag, pid); err != nil {
		log.Printf("WARNING: Watchdog termination failed for task %s: %v", taskID, err)
		systemlog.Errorf("Watchdog failed to terminate agent %s: %v", agentTag, err)
	}

	// Verify agent removed, retry if needed
	if err := qp.ensureAgentRemoved(agentTag, pid, "watchdog timeout enforcement"); err != nil {
		// Keep execution tracked if removal failed - reconciliation might help later
		systemlog.Errorf("CRITICAL: Watchdog failed to remove agent %s after timeout - keeping tracked: %v", agentTag, err)
		log.Printf("ERROR: Watchdog unable to remove timed-out agent %s - execution remains tracked: %v", agentTag, err)
		return
	}

	// Successfully cleaned up, safe to unregister
	qp.unregisterExecution(taskID)
	log.Printf("✅ WATCHDOG: Successfully terminated and cleaned up timed-out task %s", taskID)
	systemlog.Infof("Watchdog successfully terminated timed-out task %s", taskID)
}
