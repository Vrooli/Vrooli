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

	// Running processes registry
	runningProcesses      map[string]*tasks.RunningProcess
	runningProcessesMutex sync.RWMutex

	// Broadcast channel for WebSocket updates
	broadcast chan<- interface{}

	// Process persistence file
	processFile string

	// Process health monitoring
	healthCheckInterval time.Duration
	stopHealthCheck     chan bool
}

// NewProcessor creates a new queue processor
func NewProcessor(interval time.Duration, storage *tasks.Storage, assembler *prompts.Assembler, broadcast chan<- interface{}) *Processor {
	processor := &Processor{
		processInterval:     interval,
		stopChannel:         make(chan bool),
		storage:             storage,
		assembler:           assembler,
		runningProcesses:    make(map[string]*tasks.RunningProcess),
		broadcast:           broadcast,
		processFile:         filepath.Join(os.TempDir(), "ecosystem-manager-processes.json"),
		healthCheckInterval: 30 * time.Second,
		stopHealthCheck:     make(chan bool),
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

	// Check current in-progress tasks
	inProgressTasks, err := qp.storage.GetQueueItems("in-progress")
	if err != nil {
		log.Printf("Error checking in-progress tasks: %v", err)
		return
	}

	// Count tasks that are actually executing (check the running processes registry)
	qp.runningProcessesMutex.RLock()
	executingCount := len(qp.runningProcesses)
	qp.runningProcessesMutex.RUnlock()

	var readyToExecute []tasks.TaskItem

	for _, task := range inProgressTasks {
		// Check if this task is actually running
		if _, isRunning := qp.getRunningProcess(task.ID); !isRunning {
			// Task was manually moved to in-progress but not started yet
			readyToExecute = append(readyToExecute, task)
		}
	}

	// Get pending tasks
	pendingTasks, err := qp.storage.GetQueueItems("pending")
	if err != nil {
		log.Printf("Error getting pending tasks: %v", err)
		return
	}

	// Combine pending and ready-to-execute in-progress tasks
	allReadyTasks := append(pendingTasks, readyToExecute...)

	if len(allReadyTasks) == 0 {
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
	var taskCurrentStatus string
	highestPriority := 0

	for i, task := range allReadyTasks {
		priority := priorityOrder[task.Priority]
		if priority > highestPriority {
			highestPriority = priority
			selectedTask = &allReadyTasks[i]
			// Determine if task is from pending or already in-progress
			if tasks.ContainsTask(pendingTasks, task) {
				taskCurrentStatus = "pending"
			} else {
				taskCurrentStatus = "in-progress"
			}
		}
	}

	if selectedTask == nil {
		return
	}

	log.Printf("Processing task: %s - %s (from %s)", selectedTask.ID, selectedTask.Title, taskCurrentStatus)

	// Move task to in-progress if it's not already there
	if taskCurrentStatus == "pending" {
		if err := qp.storage.MoveTask(selectedTask.ID, "pending", "in-progress"); err != nil {
			log.Printf("Failed to move task to in-progress: %v", err)
			return
		}
	}

	// Process the task asynchronously
	go qp.executeTask(*selectedTask)
}

// Process registry management
func (qp *Processor) registerRunningProcess(taskID string, cmd *exec.Cmd, ctx context.Context, cancel context.CancelFunc) {
	qp.runningProcessesMutex.Lock()
	defer qp.runningProcessesMutex.Unlock()

	process := &tasks.RunningProcess{
		TaskID:    taskID,
		Cmd:       cmd,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		ProcessID: cmd.Process.Pid,
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
	qp.runningProcessesMutex.Lock()
	defer qp.runningProcessesMutex.Unlock()

	process, exists := qp.runningProcesses[taskID]
	if !exists {
		return fmt.Errorf("no running process found for task %s", taskID)
	}

	log.Printf("Terminating process %d for task %s", process.ProcessID, taskID)

	// First try graceful cancellation via context
	if cancel, ok := process.Cancel.(context.CancelFunc); ok {
		cancel()
	}

	// Give it 5 seconds to shut down gracefully
	select {
	case <-time.After(5 * time.Second):
		// If still running, force kill
		if cmd, ok := process.Cmd.(*exec.Cmd); ok && cmd.Process != nil {
			log.Printf("Force killing process %d for task %s", process.ProcessID, taskID)
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Error force killing process: %v", err)
			}
		}
	case <-func() <-chan struct{} {
		done := make(chan struct{})
		if ctx, ok := process.Context.(context.Context); ok {
			go func() {
				<-ctx.Done()
				close(done)
			}()
		}
		return done
	}():
		// Process terminated gracefully
		log.Printf("Process %d for task %s terminated gracefully", process.ProcessID, taskID)
	}

	// Clean up registry
	delete(qp.runningProcesses, taskID)
	return nil
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
		})
	}

	return processes
}

type ProcessInfo struct {
	TaskID    string `json:"task_id"`
	ProcessID int    `json:"process_id"`
	StartTime string `json:"start_time"`
	Duration  string `json:"duration"`
}

// GetQueueStatus returns current queue processor status and metrics
func (qp *Processor) GetQueueStatus() map[string]interface{} {
	// Get current maintenance state
	qp.mu.Lock()
	isPaused := qp.isPaused
	isRunning := qp.isRunning
	qp.mu.Unlock()

	// Count tasks by status
	inProgressTasks, _ := qp.storage.GetQueueItems("in-progress")
	pendingTasks, _ := qp.storage.GetQueueItems("pending")

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

	return map[string]interface{}{
		"processor_active":  !isPaused && isRunning,
		"maintenance_state": map[bool]string{true: "inactive", false: "active"}[isPaused],
		"max_concurrent":    maxConcurrent,
		"executing_count":   executingCount,
		"available_slots":   availableSlots,
		"pending_count":     len(pendingTasks),
		"ready_in_progress": readyInProgress,
		"refresh_interval":  currentSettings.RefreshInterval, // from settings
		"processor_running": isRunning && !isPaused,
		"timestamp":         time.Now().Unix(),
	}
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
