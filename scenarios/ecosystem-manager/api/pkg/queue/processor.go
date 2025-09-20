package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// Processor manages automated queue processing
type Processor struct {
	mu              sync.Mutex
	isRunning       bool
	isPaused        bool // Added for maintenance state awareness
	stopChannel     chan bool
	processInterval time.Duration
	storage         *tasks.Storage
	assembler       *prompts.Assembler

	// Running processes registry (legacy - will be phased out)
	runningProcesses      map[string]*tasks.RunningProcess
	runningProcessesMutex sync.RWMutex

	// New process manager for robust process lifecycle
	processManager *ProcessManager

	// Broadcast channel for WebSocket updates
	broadcast chan<- interface{}

	// Process persistence file
	processFile string

	// Process health monitoring
	healthCheckInterval time.Duration
	stopHealthCheck     chan bool

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
		processInterval:     interval,
		stopChannel:         make(chan bool),
		storage:             storage,
		assembler:           assembler,
		runningProcesses:    make(map[string]*tasks.RunningProcess),
		processManager:      NewProcessManager(), // Initialize the new process manager
		broadcast:           broadcast,
		processFile:         filepath.Join(os.TempDir(), "ecosystem-manager-processes.json"),
		healthCheckInterval: 30 * time.Second,
		stopHealthCheck:     make(chan bool),
		taskLogs:            make(map[string]*TaskLogBuffer),
	}

	// Load persisted processes on startup
	processor.loadPersistedProcesses()

	// Clean up orphaned processes
	processor.cleanupOrphanedProcesses()

	// Start process health monitoring
	go processor.processHealthMonitor()

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

	// Stop health monitoring first
	select {
	case qp.stopHealthCheck <- true:
	default:
	}

	// Stop queue processing
	qp.stopChannel <- true
	qp.isRunning = false
	log.Println("Queue processor stopped")

	// Terminate all managed processes gracefully
	if qp.processManager != nil {
		log.Println("Terminating all managed processes...")
		qp.processManager.TerminateAll(10 * time.Second)
	}

	// Clean up process persistence file
	qp.clearPersistedProcesses()
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

	// Check current in-progress tasks
	inProgressTasks, err := qp.storage.GetQueueItems("in-progress")
	if err != nil {
		log.Printf("Error checking in-progress tasks: %v", err)
		return
	}

	for _, task := range inProgressTasks {
		if qp.IsTaskRunning(task.ID) || qp.processManager.IsProcessActive(task.ID) {
			continue
		}

		agentIdentifier := fmt.Sprintf("ecosystem-%s", task.ID)
		qp.stopClaudeAgent(agentIdentifier, 0)
		systemlog.Warnf("Detected orphan in-progress task %s; relocating to pending", task.ID)
		if err := qp.storage.MoveTask(task.ID, "in-progress", "pending"); err != nil {
			log.Printf("Failed to move orphan task %s back to pending: %v", task.ID, err)
			systemlog.Errorf("Failed to move orphan task %s back to pending: %v", task.ID, err)
		} else {
			// Newly moved tasks will be picked up from pending in this or next iteration
		}
	}

	// Count tasks that are actually executing (check the running processes registry)
	qp.runningProcessesMutex.RLock()
	executingCount := len(qp.runningProcesses)
	qp.runningProcessesMutex.RUnlock()

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

	// Move task to in-progress if it's not already there
	if err := qp.storage.MoveTask(selectedTask.ID, "pending", "in-progress"); err != nil {
		log.Printf("Failed to move task to in-progress: %v", err)
		return
	}
	// Broadcast that the task has moved to in-progress
	selectedTask.Status = "in-progress"
	selectedTask.CurrentPhase = "in-progress"
	qp.broadcastUpdate("task_status_changed", map[string]interface{}{
		"task_id":    selectedTask.ID,
		"old_status": "pending",
		"new_status": "in-progress",
		"task":       selectedTask,
	})

	// Process the task asynchronously
	go qp.executeTask(*selectedTask)
}

// Process registry management
func (qp *Processor) registerRunningProcess(taskID string, cmd *exec.Cmd, ctx context.Context, cancel context.CancelFunc, agentID string) {
	qp.runningProcessesMutex.Lock()
	defer qp.runningProcessesMutex.Unlock()

	process := &tasks.RunningProcess{
		TaskID:    taskID,
		Cmd:       cmd,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		ProcessID: cmd.Process.Pid,
		AgentID:   agentID,
	}

	qp.runningProcesses[taskID] = process
	log.Printf("Registered process %d for task %s", process.ProcessID, taskID)

	// Persist the updated process list
	go qp.persistProcesses()
}

func (qp *Processor) unregisterRunningProcess(taskID string) {
	qp.runningProcessesMutex.Lock()
	defer qp.runningProcessesMutex.Unlock()

	if process, exists := qp.runningProcesses[taskID]; exists {
		log.Printf("Unregistered process %d for task %s", process.ProcessID, taskID)
		delete(qp.runningProcesses, taskID)

		// Persist the updated process list
		go qp.persistProcesses()
	}
}

func (qp *Processor) getRunningProcess(taskID string) (*tasks.RunningProcess, bool) {
	qp.runningProcessesMutex.RLock()
	defer qp.runningProcessesMutex.RUnlock()

	process, exists := qp.runningProcesses[taskID]
	return process, exists
}

func (qp *Processor) TerminateRunningProcess(taskID string) error {
	process, hasProcess := qp.getRunningProcess(taskID)

	agentIdentifier := fmt.Sprintf("ecosystem-%s", taskID)
	if hasProcess && process.AgentID != "" {
		agentIdentifier = process.AgentID
	}

	pid := 0
	if hasProcess {
		pid = process.ProcessID
	}

	if qp.processManager != nil && qp.processManager.IsProcessActive(taskID) {
		log.Printf("Using ProcessManager to terminate task %s", taskID)
		if err := qp.processManager.TerminateProcess(taskID, 5*time.Second); err != nil {
			log.Printf("ProcessManager termination failed for task %s: %v", taskID, err)
		} else {
			qp.stopClaudeAgent(agentIdentifier, pid)
			qp.cleanupClaudeAgentRegistry()
			qp.ResetTaskLogs(taskID)
			qp.unregisterRunningProcess(taskID)
			return nil
		}
	}

	return qp.legacyTerminateRunningProcess(taskID, agentIdentifier, pid)
}

func (qp *Processor) legacyTerminateRunningProcess(taskID, agentIdentifier string, pid int) error {
	qp.runningProcessesMutex.Lock()
	process, exists := qp.runningProcesses[taskID]
	if !exists {
		qp.runningProcessesMutex.Unlock()
		return fmt.Errorf("no running process found for task %s", taskID)
	}

	cmd, _ := process.Cmd.(*exec.Cmd)
	var cancel context.CancelFunc
	if c, ok := process.Cancel.(context.CancelFunc); ok {
		cancel = c
	}
	var ctx context.Context
	if c, ok := process.Context.(context.Context); ok {
		ctx = c
	}
	if agentIdentifier == "" {
		agentIdentifier = process.AgentID
		if agentIdentifier == "" {
			agentIdentifier = fmt.Sprintf("ecosystem-%s", taskID)
		}
	}
	if pid == 0 {
		pid = process.ProcessID
	}
	qp.runningProcessesMutex.Unlock()

	log.Printf("Using legacy termination for process %d (task %s)", pid, taskID)

	qp.stopClaudeAgent(agentIdentifier, pid)
	time.Sleep(500 * time.Millisecond)

	if cancel != nil {
		cancel()
	}

	graceful := false
	if ctx != nil {
		select {
		case <-ctx.Done():
			log.Printf("Process %d for task %s terminated gracefully", pid, taskID)
			graceful = true
		case <-time.After(5 * time.Second):
			log.Printf("Process %d for task %s did not terminate within grace period", pid, taskID)
		}
	}

	if !graceful && cmd != nil && cmd.Process != nil {
		if pid > 0 {
			if err := KillProcessGroup(pid); err != nil {
				log.Printf("Failed to kill process group -%d: %v", pid, err)
			}
		}
		time.Sleep(500 * time.Millisecond)

		if qp.isProcessAlive(pid) {
			log.Printf("Force killing process %d for task %s", pid, taskID)
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Error force killing process %d: %v", pid, err)
			}
		}
	}

	qp.cleanupClaudeAgentRegistry()

	qp.runningProcessesMutex.Lock()
	delete(qp.runningProcesses, taskID)
	qp.runningProcessesMutex.Unlock()
	go qp.persistProcesses()

	qp.ResetTaskLogs(taskID)
	return nil
}

func (qp *Processor) stopClaudeAgent(agentIdentifier string, pid int) {
	if agentIdentifier != "" {
		if err := exec.Command("resource-claude-code", "agents", "stop", agentIdentifier).Run(); err != nil {
			log.Printf("Failed to stop claude-code agent %s: %v", agentIdentifier, err)
		} else {
			log.Printf("Successfully stopped claude-code agent %s", agentIdentifier)
			return
		}
	}

	if pid > 0 {
		if err := exec.Command("resource-claude-code", "agents", "stop", strconv.Itoa(pid)).Run(); err != nil {
			log.Printf("Failed to stop claude-code agent by PID %d: %v", pid, err)
		} else {
			log.Printf("Successfully stopped claude-code agent by PID %d", pid)
		}
	}
}

func (qp *Processor) cleanupClaudeAgentRegistry() {
	cleanupCmd := exec.Command("resource-claude-code", "agents", "cleanup")
	if err := cleanupCmd.Run(); err != nil {
		log.Printf("Warning: Failed to cleanup claude-code agents: %v", err)
	}
}

func (qp *Processor) finalizeTaskStatus(task tasks.TaskItem, fromStatus, toStatus string) {
	if fromStatus != toStatus {
		if err := qp.storage.MoveTask(task.ID, fromStatus, toStatus); err != nil {
			log.Printf("Failed to move task %s from %s to %s: %v", task.ID, fromStatus, toStatus, err)
			systemlog.Warnf("Failed to move task %s from %s to %s: %v", task.ID, fromStatus, toStatus, err)
			if saveErr := qp.storage.SaveQueueItem(task, toStatus); saveErr != nil {
				log.Printf("ERROR: Unable to finalize task %s in %s after move failure: %v", task.ID, toStatus, saveErr)
				systemlog.Errorf("Unable to finalize task %s in %s after move failure: %v", task.ID, toStatus, saveErr)
			} else {
				if _, delErr := qp.storage.DeleteTask(task.ID); delErr != nil {
					log.Printf("WARNING: Task %s may still exist in %s after fallback save: %v", task.ID, fromStatus, delErr)
					systemlog.Warnf("Task %s may still exist in %s after fallback save: %v", task.ID, fromStatus, delErr)
				}
			}
		} else {
			systemlog.Debugf("Task %s moved from %s to %s", task.ID, fromStatus, toStatus)
		}
	} else {
		if err := qp.storage.SaveQueueItem(task, toStatus); err != nil {
			log.Printf("ERROR: Unable to finalize task %s in %s: %v", task.ID, toStatus, err)
			systemlog.Errorf("Unable to finalize task %s in %s: %v", task.ID, toStatus, err)
		}
		systemlog.Debugf("Task %s state updated in %s without move", task.ID, toStatus)
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
	qp.runningProcessesMutex.RLock()
	defer qp.runningProcessesMutex.RUnlock()

	var taskIDs []string
	for taskID := range qp.runningProcesses {
		taskIDs = append(taskIDs, taskID)
	}
	return taskIDs
}

func (qp *Processor) GetRunningProcessesInfo() []ProcessInfo {
	qp.runningProcessesMutex.RLock()
	defer qp.runningProcessesMutex.RUnlock()

	var processes []ProcessInfo
	now := time.Now()

	for taskID, process := range qp.runningProcesses {
		duration := now.Sub(process.StartTime)
		processes = append(processes, ProcessInfo{
			TaskID:    taskID,
			ProcessID: process.ProcessID,
			StartTime: process.StartTime.Format(time.RFC3339),
			Duration:  duration.Round(time.Second).String(),
			AgentID:   process.AgentID,
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

// markTaskLogsCompleted flags the log buffer as finished while retaining the output
func (qp *Processor) markTaskLogsCompleted(taskID string) {
	qp.taskLogsMutex.Lock()
	if buffer, exists := qp.taskLogs[taskID]; exists {
		buffer.Completed = true
		buffer.UpdatedAt = time.Now()
	}
	qp.taskLogsMutex.Unlock()
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
	_, exists := qp.getRunningProcess(taskID)
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

	// Count actually executing tasks using process registry (more accurate)
	qp.runningProcessesMutex.RLock()
	executingCount := len(qp.runningProcesses)
	qp.runningProcessesMutex.RUnlock()

	// Count ready-to-execute tasks in in-progress
	readyInProgress := 0
	for _, task := range inProgressTasks {
		if _, isRunning := qp.getRunningProcess(task.ID); !isRunning {
			readyInProgress++
		}
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

// Process persistence and health monitoring methods

// PersistentProcess represents a process that can be serialized/deserialized
type PersistentProcess struct {
	TaskID    string    `json:"task_id"`
	ProcessID int       `json:"process_id"`
	StartTime time.Time `json:"start_time"`
}

// loadPersistedProcesses loads running processes from disk on startup
func (qp *Processor) loadPersistedProcesses() {
	data, err := os.ReadFile(qp.processFile)
	if err != nil {
		// File doesn't exist or can't be read - this is normal for first startup
		return
	}

	var persistedProcesses []PersistentProcess
	if err := json.Unmarshal(data, &persistedProcesses); err != nil {
		log.Printf("Error unmarshaling persisted processes: %v", err)
		return
	}

	log.Printf("Loaded %d persisted processes from %s", len(persistedProcesses), qp.processFile)

	// Check which processes are still alive and clean up dead ones
	aliveCount := 0
	for _, pp := range persistedProcesses {
		if qp.isProcessAlive(pp.ProcessID) {
			log.Printf("Found alive orphaned process: TaskID=%s PID=%d (running since %v)",
				pp.TaskID, pp.ProcessID, pp.StartTime.Format(time.RFC3339))

			// Attempt to kill orphaned process
			if err := qp.killProcess(pp.ProcessID); err != nil {
				log.Printf("Failed to kill orphaned process %d: %v", pp.ProcessID, err)
			} else {
				log.Printf("Successfully killed orphaned process %d", pp.ProcessID)
			}
			aliveCount++
		}
	}

	if aliveCount > 0 {
		log.Printf("Cleaned up %d orphaned processes from previous run", aliveCount)
	}

	// Clear the persistence file since we handled all processes
	qp.clearPersistedProcesses()
}

// persistProcesses saves current running processes to disk
func (qp *Processor) persistProcesses() {
	qp.runningProcessesMutex.RLock()
	defer qp.runningProcessesMutex.RUnlock()

	if len(qp.runningProcesses) == 0 {
		// No processes to persist, remove file if it exists
		qp.clearPersistedProcesses()
		return
	}

	var persistedProcesses []PersistentProcess
	for taskID, process := range qp.runningProcesses {
		persistedProcesses = append(persistedProcesses, PersistentProcess{
			TaskID:    taskID,
			ProcessID: process.ProcessID,
			StartTime: process.StartTime,
		})
	}

	data, err := json.MarshalIndent(persistedProcesses, "", "  ")
	if err != nil {
		log.Printf("Error marshaling processes for persistence: %v", err)
		return
	}

	if err := os.WriteFile(qp.processFile, data, 0644); err != nil {
		log.Printf("Error writing processes to persistence file: %v", err)
		return
	}

	log.Printf("Persisted %d running processes to %s", len(persistedProcesses), qp.processFile)
}

// clearPersistedProcesses removes the persistence file
func (qp *Processor) clearPersistedProcesses() {
	if err := os.Remove(qp.processFile); err != nil && !os.IsNotExist(err) {
		log.Printf("Error removing process persistence file: %v", err)
	}
}

// processHealthMonitor runs in background to check process health
func (qp *Processor) processHealthMonitor() {
	ticker := time.NewTicker(qp.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			qp.checkProcessHealth()
		case <-qp.stopHealthCheck:
			log.Println("Process health monitor stopped")
			return
		}
	}
}

// checkProcessHealth verifies all registered processes are still alive
func (qp *Processor) checkProcessHealth() {
	qp.runningProcessesMutex.Lock()
	defer qp.runningProcessesMutex.Unlock()

	deadProcesses := []string{}

	for taskID, process := range qp.runningProcesses {
		if !qp.isProcessAlive(process.ProcessID) {
			log.Printf("Process %d for task %s is no longer alive - removing from registry",
				process.ProcessID, taskID)
			deadProcesses = append(deadProcesses, taskID)
		}
	}

	// Remove dead processes from registry
	for _, taskID := range deadProcesses {
		delete(qp.runningProcesses, taskID)
	}

	if len(deadProcesses) > 0 {
		log.Printf("Cleaned up %d dead processes from registry", len(deadProcesses))
		// Update persistence file
		go qp.persistProcesses()
	}
}

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

	qp.runningProcessesMutex.RLock()
	knownPIDs := make(map[int]bool)
	for _, process := range qp.runningProcesses {
		knownPIDs[process.ProcessID] = true
	}
	qp.runningProcessesMutex.RUnlock()

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
