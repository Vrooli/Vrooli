package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/agentmanager"
	"github.com/ecosystem-manager/api/pkg/internal/paths"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/steering"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

type taskExecution struct {
	taskID    string
	agentTag  string
	runID     string    // agent-manager run ID for StopRun calls
	started   time.Time
	timeoutAt time.Time // When this execution will timeout
	timedOut  bool      // Whether timeout has already occurred
}

// getRunID returns the agent-manager run ID, empty if not set
func (te *taskExecution) getRunID() string {
	if te == nil {
		return ""
	}
	return te.runID
}

// getRunIDForTask returns the agent-manager run ID for a task, empty if not found
func (qp *Processor) getRunIDForTask(taskID string) string {
	return qp.registry.GetRunIDForTask(taskID)
}

// stopRunViaAgentManager stops a run using agent-manager by task ID
// Returns nil if run was successfully stopped or wasn't tracked
func (qp *Processor) stopRunViaAgentManager(taskID string) error {
	runID := qp.getRunIDForTask(taskID)
	if runID == "" {
		return nil // No run tracked, nothing to stop
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := qp.agentSvc.StopRun(ctx, runID); err != nil {
		return fmt.Errorf("stop run %s: %w", runID, err)
	}
	return nil
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
	storage     tasks.StorageAPI
	assembler   *prompts.Assembler

	// Execution registry for tracking running tasks (interface for testability)
	registry ExecutionRegistryAPI

	// Root of the Vrooli workspace for resource CLI commands
	vrooliRoot string

	// Scenario root (parent of queue dir) for local artifact writes
	scenarioRoot string

	// Folder where we persist per-task execution logs
	taskLogsDir string

	// Broadcast channel for WebSocket updates
	broadcast chan<- any

	// Rate limit pause management
	rateLimiter *RateLimiter

	// Task log buffering and persistence
	taskLogger *TaskLogger

	// Bookkeeping for queue activity
	lastProcessedMu sync.RWMutex
	lastProcessedAt time.Time

	// Execution history cache (simple time-based invalidation)
	executionHistoryCache     []ExecutionHistory
	executionHistoryCacheTime time.Time
	executionHistoryCacheMu   sync.RWMutex

	// Auto Steer integration for multi-dimensional improvement
	autoSteerIntegration *AutoSteerIntegration

	// Recycler to trigger recycling on task completion/failure (interface for testability)
	recycler recycler.RecyclerAPI

	// Centralized task coordinator for lifecycle + side effects orchestration.
	coord *tasks.Coordinator

	// Agent manager service for delegating agent execution (interface for testability)
	agentSvc agentmanager.AgentServiceAPI

	// Execution manager for task execution lifecycle (Phase 3.1 extraction)
	executionManager *ExecutionManager

	// History manager for execution history persistence (Phase 3.2 extraction)
	historyManager *HistoryManager

	// Insight manager for insight report generation and persistence (Phase 3.3 extraction)
	insightManager *InsightManager

	// Timeout watchdog for enforcing task timeouts
	watchdog *TimeoutWatchdog

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

// ProcessorDeps contains all dependencies for the Processor.
// Using a deps struct allows for clean dependency injection and testability.
type ProcessorDeps struct {
	// Required dependencies
	Storage   tasks.StorageAPI
	Assembler *prompts.Assembler
	Broadcast chan<- any

	// Optional dependencies - if nil, defaults will be created
	Registry         ExecutionRegistryAPI
	AgentSvc         agentmanager.AgentServiceAPI
	Recycler         recycler.RecyclerAPI
	RateLimiter      *RateLimiter
	TaskLogger       *TaskLogger
	SteeringRegistry steering.RegistryAPI

	// Configuration
	VrooliRoot   string
	ScenarioRoot string
	TaskLogsDir  string
}

// NewProcessor creates a new queue processor with explicit dependencies.
// Background workers are NOT started - call Start() after construction.
func NewProcessor(deps ProcessorDeps) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	// Derive paths if not provided
	vrooliRoot := deps.VrooliRoot
	if vrooliRoot == "" {
		vrooliRoot = paths.DetectVrooliRoot()
	}
	scenarioRoot := deps.ScenarioRoot
	if scenarioRoot == "" && deps.Storage != nil {
		scenarioRoot = filepath.Dir(deps.Storage.GetQueueDir())
	}
	taskLogsDir := deps.TaskLogsDir
	if taskLogsDir == "" && deps.Storage != nil {
		taskLogsDir = filepath.Join(deps.Storage.GetQueueDir(), "..", "logs", "task-runs")
	}

	// Create default implementations for optional dependencies
	registry := deps.Registry
	if registry == nil {
		registry = NewExecutionRegistry()
	}

	rateLimiter := deps.RateLimiter
	if rateLimiter == nil {
		rateLimiter = NewRateLimiter(deps.Broadcast)
	}

	taskLogger := deps.TaskLogger
	if taskLogger == nil && taskLogsDir != "" {
		taskLogger = NewTaskLogger(taskLogsDir, deps.Broadcast)
	}

	// Create HistoryManager first (shared by Processor and ExecutionManager)
	historyManager := NewHistoryManager(taskLogsDir)

	// Create the processor
	p := &Processor{
		stopChannel:    make(chan bool, 1), // Buffered to prevent deadlock when Stop() holds mutex
		wakeCh:         make(chan struct{}, 1),
		storage:        deps.Storage,
		assembler:      deps.Assembler,
		registry:       registry,
		broadcast:      deps.Broadcast,
		rateLimiter:    rateLimiter,
		taskLogger:     taskLogger,
		recycler:       deps.Recycler,
		agentSvc:       deps.AgentSvc,
		historyManager: historyManager,
		ctx:            ctx,
		cancel:         cancel,
		vrooliRoot:     vrooliRoot,
		scenarioRoot:   scenarioRoot,
		taskLogsDir:    taskLogsDir,
	}

	// Create ExecutionManager with shared dependencies (including HistoryManager)
	p.executionManager = NewExecutionManager(ExecutionManagerDeps{
		Storage:          deps.Storage,
		Assembler:        deps.Assembler,
		AgentSvc:         deps.AgentSvc,
		Registry:         registry,
		TaskLogger:       taskLogger,
		Broadcast:        deps.Broadcast,
		SteeringRegistry: deps.SteeringRegistry,
		HistoryManager:   historyManager,
		// AutoSteerIntegration is set separately via SetAutoSteerIntegration
		TaskLogsDir: taskLogsDir,
	})

	// Wire up callbacks
	p.executionManager.SetWakeFunc(p.Wake)
	p.executionManager.SetFinalizeFunc(p.finalizeTaskStatus)

	// Create InsightManager with shared dependencies
	p.insightManager = NewInsightManager(InsightManagerDeps{
		TaskLogsDir:    taskLogsDir,
		HistoryManager: historyManager,
		AgentSvc:       deps.AgentSvc,
		Assembler:      deps.Assembler,
		Storage:        deps.Storage,
	})

	return p
}

// NewProcessorWithDefaults creates a processor with default implementations for all optional dependencies.
// This is the convenience constructor for production use that mirrors the original behavior.
// Background workers ARE started automatically for backward compatibility.
func NewProcessorWithDefaults(storage tasks.StorageAPI, assembler *prompts.Assembler, broadcast chan<- any, rec recycler.RecyclerAPI) *Processor {
	vrooliRoot := paths.DetectVrooliRoot()

	// Create agent service with default config
	agentSvc := agentmanager.NewAgentService(agentmanager.Config{
		TaskProfileKey:     "ecosystem-manager-tasks",
		InsightsProfileKey: "ecosystem-manager-insights",
		Timeout:            30 * time.Second,
		VrooliRoot:         vrooliRoot,
	})

	processor := NewProcessor(ProcessorDeps{
		Storage:   storage,
		Assembler: assembler,
		Broadcast: broadcast,
		Recycler:  rec,
		AgentSvc:  agentSvc,
	})

	// Initialize background workers for backward compatibility
	processor.InitializeWorkers()

	return processor
}

// InitializeWorkers sets up and starts all background workers.
// This should be called after construction when using NewProcessor directly.
// NewProcessorWithDefaults calls this automatically.
// Note: This does NOT start the main processLoop - call Start() separately for that.
func (qp *Processor) InitializeWorkers() {
	// Initialize agent profiles (non-blocking, log warnings)
	if qp.agentSvc != nil {
		go func() {
			initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := qp.agentSvc.Initialize(initCtx); err != nil {
				log.Printf("[agent-manager] Warning: failed to initialize profiles: %v", err)
			}
		}()
	}

	// Clean up orphaned processes
	qp.cleanupOrphanedProcesses()

	// Clean up old temporary prompt files
	go qp.cleanupOldPromptFiles()

	// Reconcile any stale in-progress tasks left behind from previous runs
	go qp.initialInProgressReconcile()

	// Create and start timeout enforcement watchdog (defense-in-depth)
	if qp.watchdog == nil && qp.registry != nil && qp.agentSvc != nil {
		qp.watchdog = NewTimeoutWatchdog(qp.registry, qp.agentSvc)
		qp.watchdog.Start()
	}
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
		outcome, err := lc.ApplyTransition(tasks.TransitionRequest{
			TaskID:            task.ID,
			ToStatus:          tasks.StatusInProgress,
			TransitionContext: ctx,
		})
		if err != nil {
			return fmt.Errorf("start pending task %s: %w", task.ID, err)
		}
		task = outcome.Task
		previousStatus = outcome.From
	}

	agentIdentifier := makeAgentTag(task.ID)
	// Note: Lingering agent cleanup is handled by agent-manager.
	// If a task was previously running, agent-manager will handle the duplicate tag.

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
	// Propagate to ExecutionManager
	if qp.executionManager != nil {
		qp.executionManager.SetAutoSteerIntegration(integration)
	}
	log.Println("✅ Auto Steer integration configured for queue processor")
	systemlog.Info("Auto Steer integration enabled")
}

// AutoSteerIntegration returns the configured Auto Steer integration, if any.
func (qp *Processor) AutoSteerIntegration() *AutoSteerIntegration {
	return qp.autoSteerIntegration
}

// SetSteeringRegistry sets the steering registry for unified steering strategy dispatch.
// This must be called after the processor is created but before processing starts.
func (qp *Processor) SetSteeringRegistry(registry steering.RegistryAPI) {
	// Propagate to ExecutionManager (steering is handled there)
	if qp.executionManager != nil {
		qp.executionManager.SetSteeringRegistry(registry)
	}
	log.Println("✅ Steering registry configured for queue processor")
	systemlog.Info("Steering registry enabled")
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

	runningCount := qp.registry.Count()

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
	if qp.watchdog != nil {
		qp.watchdog.Stop()
	}
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
	intent := tasks.IntentManual
	if allowOverflow {
		intent = tasks.IntentReconcile // Use reconcile intent to bypass constraints when overflow is allowed
	}
	return qp.startTaskExecution(task, status, externalActive, tasks.TransitionContext{
		Intent: intent,
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
		Intent: tasks.IntentAuto,
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

// GetSlotSnapshot returns the current slot availability.
// Implements SchedulerAPI.
func (qp *Processor) GetSlotSnapshot() SlotSnapshot {
	externalActive := qp.getExternalActiveTaskIDs()
	internalRunning := qp.getInternalRunningTaskIDs()
	snap := qp.computeSlotSnapshot(internalRunning, externalActive)
	return SlotSnapshot{
		Slots:     snap.Slots,
		Running:   snap.Running,
		Available: snap.Available,
	}
}

// IsRunning returns whether the processor is currently running.
// Implements SchedulerAPI.
func (qp *Processor) IsRunning() bool {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	return qp.isRunning
}

// IsPaused returns whether the processor is in maintenance mode.
// Implements SchedulerAPI.
func (qp *Processor) IsPaused() bool {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	return qp.isPaused
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

	// Check if rate limit paused (includes auto-expiration and broadcasting)
	if status := qp.rateLimiter.CheckLimit(); status.IsPaused {
		return
	}

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

		if err := qp.startTaskExecution(selectedTask, "pending", externalActive, tasks.TransitionContext{Intent: tasks.IntentAuto}); err != nil {
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
				Intent: tasks.IntentAuto,
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
				Intent: tasks.IntentAuto,
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

// UpdateAgentProfiles propagates current settings to agent-manager profiles.
// This should be called when agent-related settings change (runner_type, max_turns, etc.)
func (qp *Processor) UpdateAgentProfiles(ctx context.Context) error {
	if qp.agentSvc == nil {
		return nil
	}
	return qp.agentSvc.UpdateProfiles(ctx)
}
