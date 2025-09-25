package queue

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

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

	// New process manager for robust process lifecycle
	processManager *ProcessManager

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
		processManager:  NewProcessManager(), // Initialize the new process manager
		broadcast:       broadcast,
		taskLogs:        make(map[string]*TaskLogBuffer),
	}

	processor.vrooliRoot = detectVrooliRoot()
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

	// Terminate all managed processes gracefully
	if qp.processManager != nil {
		log.Println("Terminating all managed processes...")
		qp.processManager.TerminateAll(10 * time.Second)
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
	qp.mu.Lock()
	defer qp.mu.Unlock()

	// If processor isn't running at all, start it
	if !qp.isRunning {
		qp.isRunning = true
		go qp.processLoop()
		log.Println("Queue processor started from Resume()")
	}

	qp.isPaused = false
	log.Println("Queue processor resumed from maintenance")
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
	qp.reconcileInProgressTasks(externalActive, internalRunning)

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
		selectedTask.StartedAt = time.Now().Format(time.RFC3339)
	}
	selectedTask.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := qp.storage.SaveQueueItem(*selectedTask, "in-progress"); err != nil {
		log.Printf("Failed to persist in-progress task %s: %v", selectedTask.ID, err)
	}

	agentIdentifier := fmt.Sprintf("ecosystem-%s", selectedTask.ID)
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

// Process registry management
func (qp *Processor) reserveExecution(taskID, agentID string, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	qp.executionsMu.Lock()
	if existing, ok := qp.executions[taskID]; ok {
		if agentID != "" {
			existing.agentTag = agentID
		}
		if existing.started.IsZero() {
			existing.started = startedAt
		}
	} else {
		qp.executions[taskID] = &taskExecution{
			taskID:   taskID,
			agentTag: agentID,
			started:  startedAt,
		}
	}
	qp.executionsMu.Unlock()
}

func (qp *Processor) registerExecution(taskID, agentID string, cmd *exec.Cmd, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	qp.executionsMu.Lock()
	execState, exists := qp.executions[taskID]
	if !exists {
		execState = &taskExecution{taskID: taskID}
		qp.executions[taskID] = execState
	}
	if agentID != "" {
		execState.agentTag = agentID
	}
	execState.cmd = cmd
	if execState.started.IsZero() {
		execState.started = startedAt
	}
	qp.executionsMu.Unlock()

	if cmd != nil && cmd.Process != nil {
		log.Printf("Registered execution %d for task %s", cmd.Process.Pid, taskID)
	} else {
		log.Printf("Registered execution record for task %s (pid unknown)", taskID)
	}
}

func (qp *Processor) unregisterExecution(taskID string) {
	qp.executionsMu.Lock()
	if exec, exists := qp.executions[taskID]; exists {
		log.Printf("Unregistered execution %d for task %s", exec.pid(), taskID)
		delete(qp.executions, taskID)
	}
	qp.executionsMu.Unlock()
}

func (qp *Processor) getExecution(taskID string) (*taskExecution, bool) {
	qp.executionsMu.RLock()
	exec, exists := qp.executions[taskID]
	qp.executionsMu.RUnlock()
	return exec, exists
}

func (qp *Processor) TerminateRunningProcess(taskID string) error {
	exec, exists := qp.getExecution(taskID)
	if !exists {
		return fmt.Errorf("no running process found for task %s", taskID)
	}

	agentIdentifier := exec.agentTag
	if agentIdentifier == "" {
		agentIdentifier = fmt.Sprintf("ecosystem-%s", taskID)
	}

	pid := exec.pid()
	usedProcessManager := false
	if qp.processManager != nil && qp.processManager.IsProcessActive(taskID) {
		log.Printf("Using ProcessManager to terminate task %s", taskID)
		if err := qp.processManager.TerminateProcess(taskID, 5*time.Second); err != nil {
			log.Printf("ProcessManager termination failed for task %s: %v", taskID, err)
		} else {
			usedProcessManager = true
		}
	}

	if !usedProcessManager {
		qp.stopClaudeAgent(agentIdentifier, pid)
		qp.cleanupClaudeAgentRegistry()
	}
	qp.ResetTaskLogs(taskID)
	qp.unregisterExecution(taskID)

	return nil
}

func (qp *Processor) stopClaudeAgent(agentIdentifier string, pid int) {
	if agentIdentifier != "" {
		cmd := exec.Command("resource-claude-code", "agents", "stop", agentIdentifier)
		if qp.vrooliRoot != "" {
			cmd.Dir = qp.vrooliRoot
		}
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to stop claude-code agent %s via CLI: %v", agentIdentifier, err)
		} else {
			log.Printf("Successfully stopped claude-code agent %s", agentIdentifier)
			return
		}
	}

	if pid == 0 && agentIdentifier != "" {
		if pids := qp.agentProcessPIDs(agentIdentifier); len(pids) > 0 {
			pid = pids[0]
			for _, extra := range pids[1:] {
				qp.terminateProcessByPID(extra)
			}
		}
	}

	if pid > 0 {
		qp.terminateProcessByPID(pid)
	}
}

func (qp *Processor) cleanupClaudeAgentRegistry() {
	cleanupCmd := exec.Command("resource-claude-code", "agents", "cleanup")
	if err := cleanupCmd.Run(); err != nil {
		log.Printf("Warning: Failed to cleanup claude-code agents: %v", err)
	}
}

func (qp *Processor) agentProcessPIDs(agentTag string) []int {
	pattern := fmt.Sprintf("resource-claude-code run --tag %s", agentTag)
	cmd := exec.Command("pgrep", "-f", pattern)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	fields := strings.Fields(string(output))
	var pids []int
	for _, field := range fields {
		if pid, err := strconv.Atoi(strings.TrimSpace(field)); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

func (qp *Processor) waitForAgentShutdown(agentTag string, primaryPID int) {
	deadline := time.Now().Add(5 * time.Second)

	for {
		agentActive := qp.agentExists(agentTag)
		primaryAlive := primaryPID > 0 && qp.isProcessAlive(primaryPID)

		if !agentActive && !primaryAlive {
			return
		}

		pids := qp.agentProcessPIDs(agentTag)
		for _, pid := range pids {
			if pid == primaryPID {
				continue
			}
			if err := KillProcessGroup(pid); err != nil {
				log.Printf("Warning: failed to terminate agent process group for %s (pid %d): %v", agentTag, pid, err)
			}
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}

		if primaryAlive {
			if err := KillProcessGroup(primaryPID); err != nil {
				log.Printf("Warning: failed to terminate primary agent process group for %s (pid %d): %v", agentTag, primaryPID, err)
			}
			_ = syscall.Kill(primaryPID, syscall.SIGKILL)
		}

		if time.Now().After(deadline) {
			log.Printf("Warning: agent %s still appears active after forced shutdown", agentTag)
			return
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (qp *Processor) agentExists(agentTag string) bool {
	tags := qp.getExternalActiveTaskIDsInternal()
	_, exists := tags[strings.TrimPrefix(agentTag, "ecosystem-")]
	return exists
}

func (qp *Processor) terminateProcessByPID(pid int) {
	if pid <= 0 {
		return
	}

	if err := exec.Command("kill", "-TERM", strconv.Itoa(pid)).Run(); err != nil {
		log.Printf("Failed to send TERM to PID %d: %v", pid, err)
	}
	time.Sleep(200 * time.Millisecond)
	if qp.isProcessAlive(pid) {
		if err := exec.Command("kill", "-KILL", strconv.Itoa(pid)).Run(); err != nil {
			log.Printf("Failed to force kill PID %d: %v", pid, err)
		}
	}
}

func (qp *Processor) finalizeTaskStatus(task *tasks.TaskItem, toStatus string) {
	movedTask, fromStatus, err := qp.storage.MoveTaskTo(task.ID, toStatus)
	if err != nil {
		log.Printf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
		systemlog.Warnf("Failed to finalize task %s into %s: %v", task.ID, toStatus, err)
		return
	}

	if movedTask != nil && fromStatus == toStatus {
		systemlog.Debugf("Task %s already present in %s during finalize", task.ID, toStatus)
	}

	task.Status = toStatus
	task.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := qp.storage.SaveQueueItem(*task, toStatus); err != nil {
		log.Printf("ERROR: Unable to persist task %s in %s: %v", task.ID, toStatus, err)
		systemlog.Errorf("Unable to persist task %s in %s: %v", task.ID, toStatus, err)
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
		systemlog.Warnf("Task %s post-finalize location unknown: %v", task.ID, err)
	}
}

func (qp *Processor) ListRunningProcesses() []string {
	qp.executionsMu.RLock()
	defer qp.executionsMu.RUnlock()

	taskIDs := make([]string, 0, len(qp.executions))
	for taskID := range qp.executions {
		taskIDs = append(taskIDs, taskID)
	}
	return taskIDs
}

func (qp *Processor) GetRunningProcessesInfo() []ProcessInfo {
	qp.executionsMu.RLock()
	defer qp.executionsMu.RUnlock()

	processes := make([]ProcessInfo, 0, len(qp.executions))
	now := time.Now()

	for taskID, execState := range qp.executions {
		duration := now.Sub(execState.started)
		processes = append(processes, ProcessInfo{
			TaskID:    taskID,
			ProcessID: execState.pid(),
			StartTime: execState.started.Format(time.RFC3339),
			Duration:  duration.Round(time.Second).String(),
			AgentID:   execState.agentTag,
		})
	}

	return processes
}

type ProcessInfo struct {
	TaskID    string `json:"task_id"`
	ProcessID int    `json:"process_id"`
	StartTime string `json:"start_time"`
	Duration  string `json:"duration"`
	AgentID   string `json:"agent_id,omitempty"`
}

const maxTaskLogEntries = 2000

// TaskLogBuffer keeps a bounded rolling log for a running or recently completed task
type TaskLogBuffer struct {
	AgentID      string
	ProcessID    int
	Entries      []LogEntry
	LastSequence int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Completed    bool
}

// LogEntry represents a single line of task execution output
type LogEntry struct {
	Sequence  int64     `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// initTaskLogBuffer prepares a fresh log buffer for a task execution
func (qp *Processor) initTaskLogBuffer(taskID, agentID string, pid int) {
	qp.taskLogsMutex.Lock()
	defer qp.taskLogsMutex.Unlock()

	qp.taskLogs[taskID] = &TaskLogBuffer{
		AgentID:      agentID,
		ProcessID:    pid,
		Entries:      make([]LogEntry, 0, 64),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastSequence: 0,
	}
}

// appendTaskLog stores a log entry and emits websocket updates
func (qp *Processor) appendTaskLog(taskID, agentID, stream, message string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    stream,
		Level:     map[string]string{"stderr": "error"}[stream],
		Message:   message,
	}

	if entry.Level == "" {
		entry.Level = "info"
	}

	qp.taskLogsMutex.Lock()
	buffer, exists := qp.taskLogs[taskID]
	if !exists {
		buffer = &TaskLogBuffer{
			AgentID:   agentID,
			ProcessID: 0,
			Entries:   make([]LogEntry, 0, 64),
			CreatedAt: time.Now(),
		}
		qp.taskLogs[taskID] = buffer
	}
	if buffer.AgentID == "" {
		buffer.AgentID = agentID
	}
	buffer.LastSequence++
	entry.Sequence = buffer.LastSequence
	buffer.UpdatedAt = entry.Timestamp
	buffer.Entries = append(buffer.Entries, entry)
	if len(buffer.Entries) > maxTaskLogEntries {
		buffer.Entries = buffer.Entries[len(buffer.Entries)-maxTaskLogEntries:]
	}
	qp.taskLogsMutex.Unlock()

	qp.broadcastUpdate("log_entry", map[string]interface{}{
		"task_id":   taskID,
		"agent_id":  buffer.AgentID,
		"stream":    entry.Stream,
		"level":     entry.Level,
		"message":   entry.Message,
		"sequence":  entry.Sequence,
		"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
	})

	return entry
}

// finalizeTaskLogs writes the buffered log output to disk and annotates completion state
func (qp *Processor) finalizeTaskLogs(taskID string, completed bool) {
	var bufferCopy *TaskLogBuffer

	qp.taskLogsMutex.Lock()
	if buffer, exists := qp.taskLogs[taskID]; exists {
		buffer.Completed = completed
		buffer.UpdatedAt = time.Now()
		tmp := *buffer
		tmp.Entries = append([]LogEntry(nil), buffer.Entries...)
		bufferCopy = &tmp
	}
	qp.taskLogsMutex.Unlock()

	if bufferCopy != nil {
		if err := qp.writeTaskLogsToFile(taskID, bufferCopy); err != nil {
			log.Printf("Warning: failed to persist task logs for %s: %v", taskID, err)
		}
	}
}

// clearTaskLogs removes the log buffer for a task (used when resetting state)
func (qp *Processor) clearTaskLogs(taskID string) {
	qp.taskLogsMutex.Lock()
	delete(qp.taskLogs, taskID)
	qp.taskLogsMutex.Unlock()
}

// ResetTaskLogs removes any cached logs for a task (used when task is retried)
func (qp *Processor) ResetTaskLogs(taskID string) {
	qp.clearTaskLogs(taskID)
}

func (qp *Processor) writeTaskLogsToFile(taskID string, buffer *TaskLogBuffer) error {
	if qp.taskLogsDir == "" || buffer == nil {
		return nil
	}

	if err := os.MkdirAll(qp.taskLogsDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(qp.taskLogsDir, fmt.Sprintf("%s.log", taskID))
	file, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	metadata := fmt.Sprintf("Task: %s\nAgent: %s\nProcessID: %d\nCompleted: %v\nCreatedAt: %s\nUpdatedAt: %s\n\n",
		taskID, buffer.AgentID, buffer.ProcessID, buffer.Completed,
		buffer.CreatedAt.Format(time.RFC3339), buffer.UpdatedAt.Format(time.RFC3339))
	if _, err := writer.WriteString(metadata); err != nil {
		return err
	}

	for _, entry := range buffer.Entries {
		line := fmt.Sprintf("%s [%s] (%s) %s\n",
			entry.Timestamp.Format(time.RFC3339Nano), entry.Stream, entry.Level, entry.Message)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// GetTaskLogs returns log entries newer than the requested sequence number
func (qp *Processor) GetTaskLogs(taskID string, afterSeq int64) ([]LogEntry, int64, bool, string, bool, int) {
	qp.taskLogsMutex.RLock()
	buffer, exists := qp.taskLogs[taskID]
	qp.taskLogsMutex.RUnlock()

	isRunning := qp.IsTaskRunning(taskID)
	if !exists {
		return []LogEntry{}, afterSeq, isRunning, "", false, 0
	}

	entries := make([]LogEntry, 0, len(buffer.Entries))
	for _, entry := range buffer.Entries {
		if entry.Sequence > afterSeq {
			entries = append(entries, entry)
		}
	}

	return entries, buffer.LastSequence, isRunning, buffer.AgentID, buffer.Completed, buffer.ProcessID
}

// IsTaskRunning returns true if the task currently has a live managed process
func (qp *Processor) IsTaskRunning(taskID string) bool {
	if qp.processManager != nil && qp.processManager.IsProcessActive(taskID) {
		return true
	}
	_, exists := qp.getExecution(taskID)
	return exists
}

// GetQueueStatus returns current queue processor status and metrics
func (qp *Processor) GetQueueStatus() map[string]interface{} {
	// Get current maintenance state
	qp.mu.Lock()
	isPaused := qp.isPaused
	isRunning := qp.isRunning
	qp.mu.Unlock()

	// Check rate limit pause status
	rateLimitPaused, pauseUntil := qp.IsRateLimitPaused()
	var rateLimitInfo map[string]interface{}
	if rateLimitPaused {
		remaining := pauseUntil.Sub(time.Now())
		rateLimitInfo = map[string]interface{}{
			"paused":         true,
			"pause_until":    pauseUntil.Format(time.RFC3339),
			"remaining_secs": int(remaining.Seconds()),
		}
	} else {
		rateLimitInfo = map[string]interface{}{
			"paused": false,
		}
	}

	// Count tasks by status
	inProgressTasks, _ := qp.storage.GetQueueItems("in-progress")
	pendingTasks, _ := qp.storage.GetQueueItems("pending")
	completedTasks, _ := qp.storage.GetQueueItems("completed")
	failedTasks, _ := qp.storage.GetQueueItems("failed")
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
		if qp.processManager != nil && qp.processManager.IsProcessActive(task.ID) {
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
	var lastProcessedAt interface{}
	if !lastProcessed.IsZero() {
		lastProcessedAt = lastProcessed.Format(time.RFC3339)
	}

	return map[string]interface{}{
		"processor_active":  processorActive,
		"settings_active":   settingsActive,
		"maintenance_state": map[bool]string{true: "inactive", false: "active"}[isPaused],
		"rate_limit_info":   rateLimitInfo,
		"max_concurrent":    maxConcurrent,
		"executing_count":   executingCount,
		"running_count":     executingCount,
		"available_slots":   availableSlots,
		"pending_count":     len(pendingTasks),
		"in_progress_count": len(inProgressTasks),
		"completed_count":   len(completedTasks),
		"failed_count":      len(failedTasks),
		"review_count":      len(reviewTasks),
		"ready_in_progress": readyInProgress,
		"refresh_interval":  currentSettings.RefreshInterval, // from settings
		"processor_running": processorActive,
		"timestamp":         time.Now().Unix(),
		"last_processed_at": lastProcessedAt,
	}
}

func (qp *Processor) setLastProcessed(t time.Time) {
	qp.lastProcessedMu.Lock()
	qp.lastProcessedAt = t
	qp.lastProcessedMu.Unlock()
}

func (qp *Processor) getLastProcessed() time.Time {
	qp.lastProcessedMu.RLock()
	defer qp.lastProcessedMu.RUnlock()
	return qp.lastProcessedAt
}

func (qp *Processor) getInternalRunningTaskIDs() map[string]struct{} {
	qp.executionsMu.RLock()
	defer qp.executionsMu.RUnlock()

	ids := make(map[string]struct{}, len(qp.executions))
	for taskID := range qp.executions {
		ids[taskID] = struct{}{}
	}
	return ids
}

func (qp *Processor) getExternalActiveTaskIDs() map[string]struct{} {
	return qp.getExternalActiveTaskIDsInternal()
}

func (qp *Processor) getExternalActiveTaskIDsInternal() map[string]struct{} {
	tags, err := qp.fetchActiveClaudeAgents()
	if err != nil {
		log.Printf("Warning: failed to list claude-code agents: %v", err)
	}
	if tags == nil {
		tags = make(map[string]struct{})
	}

	processTags := qp.detectAgentsViaProcesses()
	for tag := range processTags {
		tags[tag] = struct{}{}
	}

	active := make(map[string]struct{}, len(tags))
	for tag := range tags {
		if !strings.HasPrefix(tag, "ecosystem-") {
			continue
		}
		id := strings.TrimPrefix(tag, "ecosystem-")
		if id != "" {
			active[id] = struct{}{}
		}
	}

	return active
}

func (qp *Processor) fetchActiveClaudeAgents() (map[string]struct{}, error) {
	cmd := exec.Command("resource-claude-code", "agents", "list")
	if qp.vrooliRoot != "" {
		cmd.Dir = qp.vrooliRoot
	}

	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		lower := strings.ToLower(trimmed)
		if trimmed == "" || strings.Contains(lower, "no agents found") {
			return map[string]struct{}{}, nil
		}
		return nil, fmt.Errorf("claude-code agents list failed: %w (output: %s)", err, trimmed)
	}

	if trimmed == "" {
		return map[string]struct{}{}, nil
	}

	return parseClaudeAgentList(trimmed), nil
}

func parseClaudeAgentList(output string) map[string]struct{} {
	result := make(map[string]struct{})

	if strings.EqualFold(strings.TrimSpace(output), "no agents found") {
		return result
	}

	var jsonAgents []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonAgents); err == nil {
		for _, agent := range jsonAgents {
			if id, ok := agent["id"].(string); ok && id != "" {
				result[id] = struct{}{}
			}
			if tag, ok := agent["tag"].(string); ok && tag != "" {
				result[tag] = struct{}{}
			}
		}
		return result
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "no agents found") {
			continue
		}
		for _, field := range strings.Fields(line) {
			if field == "" {
				continue
			}
			result[field] = struct{}{}
		}
	}

	return result
}

func (qp *Processor) detectAgentsViaProcesses() map[string]struct{} {
	result := make(map[string]struct{})

	cmd := exec.Command("ps", "-eo", "pid=,command=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return result
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, "resource-claude-code") || !strings.Contains(line, " run") {
			continue
		}
		fields := strings.Fields(line)
		for idx, field := range fields {
			if field == "--tag" && idx+1 < len(fields) {
				tag := fields[idx+1]
				if tag != "" {
					result[tag] = struct{}{}
				}
				break
			}
			if strings.HasPrefix(field, "--tag=") {
				tag := strings.TrimPrefix(field, "--tag=")
				if tag != "" {
					result[tag] = struct{}{}
				}
				break
			}
		}
	}

	return result
}

func (qp *Processor) initialInProgressReconcile() {
	// Allow the system to finish startup before running reconciliation
	time.Sleep(2 * time.Second)

	external := qp.getExternalActiveTaskIDs()
	internal := qp.getInternalRunningTaskIDs()
	qp.reconcileInProgressTasks(external, internal)
}

func (qp *Processor) reconcileInProgressTasks(externalActive, internalRunning map[string]struct{}) {
	inProgressTasks, err := qp.storage.GetQueueItems("in-progress")
	if err != nil {
		log.Printf("Error checking in-progress tasks: %v", err)
		return
	}

	for _, task := range inProgressTasks {
		if qp.IsTaskRunning(task.ID) {
			continue
		}
		if _, active := externalActive[task.ID]; active {
			continue
		}

		agentIdentifier := fmt.Sprintf("ecosystem-%s", task.ID)
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
		task.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := qp.storage.SaveQueueItem(task, "pending"); err != nil {
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
	}
}

func (qp *Processor) ensureAgentInactive(agentTag string) error {
	if agentTag == "" {
		return nil
	}

	tags, err := qp.fetchActiveClaudeAgents()
	if err != nil {
		return err
	}

	if _, exists := tags[agentTag]; !exists {
		// Double-check with process list before returning success
		if _, exists = qp.detectAgentsViaProcesses()[agentTag]; !exists {
			return nil
		}
	}

	qp.stopClaudeAgent(agentTag, 0)
	time.Sleep(500 * time.Millisecond)

	tags, err = qp.fetchActiveClaudeAgents()
	if err != nil {
		return err
	}

	if _, exists := tags[agentTag]; exists {
		return fmt.Errorf("claude agent %s is still active", agentTag)
	}

	if _, exists := qp.detectAgentsViaProcesses()[agentTag]; exists {
		return fmt.Errorf("claude agent %s process still active", agentTag)
		return nil
	}

	qp.stopClaudeAgent(agentTag, 0)
	time.Sleep(500 * time.Millisecond)

	tags, err = qp.fetchActiveClaudeAgents()
	if err != nil {
		return err
	}

	if _, exists := tags[agentTag]; exists {
		return fmt.Errorf("claude agent %s is still active", agentTag)
	}

	if _, exists := qp.detectAgentsViaProcesses()[agentTag]; exists {
		return fmt.Errorf("claude agent %s process still active", agentTag)
	}

	return nil
}

// Process persistence and health monitoring methods removed in favor of
// lightweight in-memory execution tracking. We still aggressively clean up
// orphaned Claude agents discovered via process inspection on startup.

// cleanupOrphanedProcesses kills any processes that match our pattern but aren't in registry
func (qp *Processor) cleanupOrphanedProcesses() {
	// This is called on startup to clean up any processes from previous crashes
	log.Println("Scanning for orphaned claude-code processes...")

	cmd := exec.Command("pgrep", "-f", "resource-claude-code")
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 when no matches found, which is fine
		return
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	orphanedCount := 0

	qp.executionsMu.RLock()
	knownPIDs := make(map[int]bool)
	for _, execState := range qp.executions {
		if pid := execState.pid(); pid > 0 {
			knownPIDs[pid] = true
		}
	}
	qp.executionsMu.RUnlock()

	for _, pidStr := range pids {
		if pidStr == "" {
			continue
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// If this PID is not in our known processes, it's orphaned
		if !knownPIDs[pid] {
			log.Printf("Found orphaned claude-code process: PID=%d", pid)
			if err := qp.killProcess(pid); err != nil {
				log.Printf("Failed to kill orphaned process %d: %v", pid, err)
			} else {
				log.Printf("Successfully killed orphaned process %d", pid)
				orphanedCount++
			}
		}
	}

	if orphanedCount > 0 {
		log.Printf("Cleaned up %d orphaned claude-code processes", orphanedCount)
	} else {
		log.Println("No orphaned claude-code processes found")
	}
}

// isProcessAlive checks if a process with the given PID is still running
func (qp *Processor) isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// On Unix systems, kill -0 checks if process exists without actually sending a signal
	if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err != nil {
		return false
	}
	return true
}

// killProcess attempts to terminate a process gracefully then forcefully
func (qp *Processor) killProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// First try SIGTERM for graceful shutdown
	if err := exec.Command("kill", "-TERM", strconv.Itoa(pid)).Run(); err != nil {
		// If SIGTERM failed, try SIGKILL
		if killErr := exec.Command("kill", "-KILL", strconv.Itoa(pid)).Run(); killErr != nil {
			return fmt.Errorf("failed to kill process %d: TERM failed (%v), KILL failed (%v)", pid, err, killErr)
		}
	}

	// Give it a moment to die
	time.Sleep(100 * time.Millisecond)

	// Verify it's dead
	if qp.isProcessAlive(pid) {
		return fmt.Errorf("process %d still alive after kill attempts", pid)
	}

	return nil
}

// handleRateLimitPause pauses the queue processor due to rate limiting
func (qp *Processor) handleRateLimitPause(retryAfterSeconds int) {
	qp.pauseMutex.Lock()
	defer qp.pauseMutex.Unlock()

	// Cap the pause duration at 4 hours
	if retryAfterSeconds > 14400 {
		retryAfterSeconds = 14400
	}

	// Minimum pause of 5 minutes
	if retryAfterSeconds < 300 {
		retryAfterSeconds = 300
	}

	pauseDuration := time.Duration(retryAfterSeconds) * time.Second
	qp.rateLimitPaused = true
	qp.pauseUntil = time.Now().Add(pauseDuration)

	log.Printf("🛑 RATE LIMIT HIT: Pausing queue processor for %v (until %s)",
		pauseDuration, qp.pauseUntil.Format(time.RFC3339))

	// Broadcast the pause event
	qp.broadcastUpdate("rate_limit_pause_started", map[string]interface{}{
		"pause_duration": retryAfterSeconds,
		"pause_until":    qp.pauseUntil.Format(time.RFC3339),
		"reason":         "API rate limit reached",
	})
}

// IsRateLimitPaused checks if the processor is currently paused due to rate limits
func (qp *Processor) IsRateLimitPaused() (bool, time.Time) {
	qp.pauseMutex.Lock()
	defer qp.pauseMutex.Unlock()

	if qp.rateLimitPaused && time.Now().Before(qp.pauseUntil) {
		return true, qp.pauseUntil
	}
	return false, time.Time{}
}

// ResetRateLimitPause manually resets the rate limit pause
func (qp *Processor) ResetRateLimitPause() {
	qp.pauseMutex.Lock()
	defer qp.pauseMutex.Unlock()

	wasRateLimited := qp.rateLimitPaused
	qp.rateLimitPaused = false
	qp.pauseUntil = time.Time{}

	if wasRateLimited {
		log.Printf("✅ Rate limit pause manually reset. Queue processing resumed.")

		// Broadcast the resume event
		qp.broadcastUpdate("rate_limit_manual_reset", map[string]interface{}{
			"paused": false,
			"manual": true,
		})
	}
}

func detectVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}

	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for dir != string(filepath.Separator) && dir != "." {
			if _, statErr := os.Stat(filepath.Join(dir, ".vrooli")); statErr == nil {
				return dir
			}
			dir = filepath.Dir(dir)
		}
	}

	return "."
}
