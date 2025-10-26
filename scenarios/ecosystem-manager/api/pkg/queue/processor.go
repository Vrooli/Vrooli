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
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

type taskExecution struct {
	taskID   string
	agentTag string
	cmd      *exec.Cmd
	started  time.Time
}

func (te *taskExecution) pid() int {
	if te == nil || te.cmd == nil || te.cmd.Process == nil {
		return 0
	}
	return te.cmd.Process.Pid
}

// Processor manages automated queue processing
type Processor struct {
	mu              sync.Mutex
	isRunning       bool
	isPaused        bool // Added for maintenance state awareness
	stopChannel     chan bool
	processInterval time.Duration
	storage         *tasks.Storage
	assembler       *prompts.Assembler

	// Live task executions keyed by task ID
	executions   map[string]*taskExecution
	executionsMu sync.RWMutex

	// Root of the Vrooli workspace for resource CLI commands
	vrooliRoot string

	// Folder where we persist per-task execution logs
	taskLogsDir string

	// Broadcast channel for WebSocket updates
	broadcast chan<- interface{}

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
}

// NewProcessor creates a new queue processor
func NewProcessor(interval time.Duration, storage *tasks.Storage, assembler *prompts.Assembler, broadcast chan<- interface{}) *Processor {
	processor := &Processor{
		processInterval: interval,
		stopChannel:     make(chan bool),
		storage:         storage,
		assembler:       assembler,
		executions:      make(map[string]*taskExecution),
		broadcast:       broadcast,
		taskLogs:        make(map[string]*TaskLogBuffer),
	}

	processor.vrooliRoot = paths.DetectVrooliRoot()
	processor.taskLogsDir = filepath.Join(storage.QueueDir, "..", "logs", "task-runs")
	if err := os.MkdirAll(processor.taskLogsDir, 0755); err != nil {
		log.Printf("Warning: unable to create task logs directory %s: %v", processor.taskLogsDir, err)
	}

	// Clean up orphaned processes
	processor.cleanupOrphanedProcesses()

	// Reconcile any stale in-progress tasks left behind from previous runs
	go processor.initialInProgressReconcile()

	return processor
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
	log.Println("Queue processor started")
}

// Stop halts the queue processing loop
func (qp *Processor) Stop() {
	qp.mu.Lock()
	defer qp.mu.Unlock()

	if !qp.isRunning {
		return
	}
	qp.stopChannel <- true
	qp.isRunning = false
	log.Println("Queue processor stopped")

	// Terminate all tracked executions
	qp.executionsMu.RLock()
	taskIDs := make([]string, 0, len(qp.executions))
	for taskID := range qp.executions {
		taskIDs = append(taskIDs, taskID)
	}
	qp.executionsMu.RUnlock()

	for _, taskID := range taskIDs {
		if err := qp.TerminateRunningProcess(taskID); err != nil {
			log.Printf("Warning: failed to terminate task %s during shutdown: %v", taskID, err)
		}
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

// processLoop is the main queue processing loop
func (qp *Processor) processLoop() {
	ticker := time.NewTicker(qp.processInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			qp.ProcessQueue()
		case <-qp.stopChannel:
			return
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
			remaining := qp.pauseUntil.Sub(time.Now())
			log.Printf("⏸️ Queue paused due to rate limit. Resuming in %v", remaining.Round(time.Second))
			qp.pauseMutex.Unlock()

			// Broadcast pause status
			qp.broadcastUpdate("rate_limit_pause", map[string]interface{}{
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
			qp.broadcastUpdate("rate_limit_resume", map[string]interface{}{
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

	// Find highest priority task from all ready tasks
	var selectedTask *tasks.TaskItem
	highestPriority := 0

	for i, task := range pendingTasks {
		if !task.ProcessorAutoRequeue {
			continue
		}
		priority := priorityOrder[task.Priority]
		if priority > highestPriority {
			highestPriority = priority
			selectedTask = &pendingTasks[i]
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
		qp.stopClaudeAgent(agentIdentifier, 0)
		delete(externalActive, selectedTask.ID)
	}

	// Reserve execution slot immediately so reconciliation won't recycle the task
	qp.reserveExecution(selectedTask.ID, agentIdentifier, time.Now())

	qp.broadcastUpdate("task_status_changed", map[string]interface{}{
		"task_id":    selectedTask.ID,
		"old_status": previousStatus,
		"new_status": "in-progress",
		"task":       selectedTask,
	})

	// Process the task asynchronously
	go qp.executeTask(*selectedTask)
}

func (qp *Processor) finalizeTaskStatus(task *tasks.TaskItem, toStatus string) error {
	movedTask, fromStatus, err := qp.storage.MoveTaskTo(task.ID, toStatus)
	if err != nil {
		wrappedErr := fmt.Errorf("finalize task %s: move from %s to %s: %w", task.ID, fromStatus, toStatus, err)
		log.Printf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
		systemlog.Errorf("Failed to finalize task %s into %s: %v", task.ID, toStatus, wrappedErr)
		return wrappedErr
	}

	if movedTask != nil && fromStatus == toStatus {
		systemlog.Debugf("Task %s already present in %s during finalize", task.ID, toStatus)
	}

	task.Status = toStatus
	task.UpdatedAt = timeutil.NowRFC3339()

	// Use SkipCleanup since MoveTaskTo already cleaned up duplicates
	if err := qp.storage.SaveQueueItemSkipCleanup(*task, toStatus); err != nil {
		wrappedErr := fmt.Errorf("finalize task %s: save in %s after move: %w", task.ID, toStatus, err)
		log.Printf("ERROR: Unable to persist task %s in %s: %v", task.ID, toStatus, err)
		systemlog.Errorf("Unable to persist task %s in %s: %v", task.ID, toStatus, wrappedErr)
		return wrappedErr
	}

	systemlog.Debugf("Task %s finalized: %s -> %s", task.ID, fromStatus, toStatus)
	qp.broadcastUpdate("task_status_changed", map[string]interface{}{
		"task_id":    task.ID,
		"old_status": fromStatus,
		"new_status": toStatus,
		"task":       task,
	})

	if status, err := qp.storage.CurrentStatus(task.ID); err == nil {
		systemlog.Debugf("Task %s post-finalize location: %s", task.ID, status)
	} else {
		wrappedErr := fmt.Errorf("verify task %s post-finalize location: %w", task.ID, err)
		systemlog.Warnf("Task %s post-finalize location unknown: %v", task.ID, wrappedErr)
	}

	return nil
}
