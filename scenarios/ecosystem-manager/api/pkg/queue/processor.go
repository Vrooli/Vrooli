package queue

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/paths"
	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
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
}

// NewProcessor creates a new queue processor
func NewProcessor(storage *tasks.Storage, assembler *prompts.Assembler, broadcast chan<- any, recycler *recycler.Recycler) *Processor {
	processor := &Processor{
		stopChannel: make(chan bool),
		wakeCh:      make(chan struct{}, 1),
		storage:     storage,
		assembler:   assembler,
		executions:  make(map[string]*taskExecution),
		broadcast:   broadcast,
		taskLogs:    make(map[string]*TaskLogBuffer),
		recycler:    recycler,
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
	safetyTicker := time.NewTicker(30 * time.Second)
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

	executingCount := len(internalRunning)
	for taskID := range externalActive {
		if _, tracked := internalRunning[taskID]; !tracked {
			executingCount++
		}
	}

	// Get pending tasks (re-fetch if we moved any orphans)
	pendingTasks, err := qp.storage.GetQueueItems("pending")
	if err != nil {
		log.Printf("Error getting pending tasks: %v", err)
		return
	}

	if len(pendingTasks) == 0 {
		return // No tasks to process
	}

	// Limit concurrent tasks based on settings
	currentSettings := settings.GetSettings()
	maxConcurrent := currentSettings.Slots
	availableSlots := maxConcurrent - executingCount
	if availableSlots <= 0 {
		log.Printf("Queue processor: %d tasks already executing, %d available slots", executingCount, availableSlots)
		return
	}

	// Sort tasks by priority (critical > high > medium > low)
	priorityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}

	for availableSlots > 0 {
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

		movedTask, previousStatus, err := qp.storage.MoveTaskTo(selectedTask.ID, "in-progress")
		if err != nil {
			log.Printf("Failed to move task %s to in-progress: %v", selectedTask.ID, err)
			systemlog.Errorf("Failed to move task %s to in-progress: %v", selectedTask.ID, err)
			return
		}
		if movedTask != nil {
			selectedTask = movedTask
		}

		selectedTask.Status = "in-progress"
		selectedTask.CurrentPhase = "in-progress"
		selectedTask.Results = nil
		selectedTask.CompletedAt = ""
		if selectedTask.StartedAt == "" {
			selectedTask.StartedAt = timeutil.NowRFC3339()
		}
		selectedTask.UpdatedAt = timeutil.NowRFC3339()

		// Use SkipCleanup since MoveTaskTo already cleaned up duplicates
		if err := qp.storage.SaveQueueItemSkipCleanup(*selectedTask, "in-progress"); err != nil {
			log.Printf("Failed to persist in-progress task %s: %v", selectedTask.ID, err)
		}

		agentIdentifier := makeAgentTag(selectedTask.ID)
		if _, active := externalActive[selectedTask.ID]; active {
			log.Printf("Detected lingering Claude agent for task %s; attempting cleanup before restart", selectedTask.ID)
			if err := qp.stopClaudeAgent(agentIdentifier, 0); err != nil {
				log.Printf("Warning: failed to stop lingering agent %s: %v", agentIdentifier, err)
			}
			delete(externalActive, selectedTask.ID)
		}

		// Reserve execution slot immediately so reconciliation won't recycle the task
		qp.reserveExecution(selectedTask.ID, agentIdentifier, time.Now())

		qp.broadcastUpdate("task_status_changed", map[string]any{
			"task_id":    selectedTask.ID,
			"old_status": previousStatus,
			"new_status": "in-progress",
			"task":       selectedTask,
		})

		// Process the task asynchronously
		go qp.executeTask(*selectedTask)

		executingCount++
		availableSlots--

		// Remove the selected task from the local slice to avoid reselection in the same pass
		pendingTasks = append(pendingTasks[:selectedIdx], pendingTasks[selectedIdx+1:]...)
		if len(pendingTasks) == 0 {
			return
		}
	}
}

func (qp *Processor) finalizeTaskStatus(task *tasks.TaskItem, toStatus string) error {
	movedTask, fromStatus, err := qp.storage.MoveTaskTo(task.ID, toStatus)
	if err != nil {
		log.Printf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
		systemlog.Errorf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
		return fmt.Errorf("move to %s: %w", toStatus, err)
	}

	if movedTask != nil && fromStatus == toStatus {
		systemlog.Debugf("Task %s already present in %s during finalize", task.ID, toStatus)
	}

	task.Status = toStatus
	task.UpdatedAt = timeutil.NowRFC3339()

	// Use SkipCleanup since MoveTaskTo already cleaned up duplicates
	if err := qp.storage.SaveQueueItemSkipCleanup(*task, toStatus); err != nil {
		log.Printf("ERROR: Unable to persist task %s in %s: %v", task.ID, toStatus, err)
		systemlog.Errorf("Unable to persist task %s in %s: %v", task.ID, toStatus, err)
		return fmt.Errorf("save after move: %w", err)
	}

	systemlog.Debugf("Task %s finalized: %s -> %s", task.ID, fromStatus, toStatus)
	qp.broadcastUpdate("task_status_changed", map[string]any{
		"task_id":    task.ID,
		"old_status": fromStatus,
		"new_status": toStatus,
		"task":       task,
	})

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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		qp.enforceTimeouts()
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
