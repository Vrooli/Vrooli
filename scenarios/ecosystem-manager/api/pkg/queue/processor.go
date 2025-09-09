package queue

import (
	"context"
	"fmt"
	"log"
	"os/exec"
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
}

// NewProcessor creates a new queue processor
func NewProcessor(interval time.Duration, storage *tasks.Storage, assembler *prompts.Assembler, broadcast chan<- interface{}) *Processor {
	return &Processor{
		processInterval:  interval,
		stopChannel:      make(chan bool),
		storage:         storage,
		assembler:       assembler,
		runningProcesses: make(map[string]*tasks.RunningProcess),
		broadcast:       broadcast,
	}
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
}

func (qp *Processor) unregisterRunningProcess(taskID string) {
	qp.runningProcessesMutex.Lock()
	defer qp.runningProcessesMutex.Unlock()
	
	if process, exists := qp.runningProcesses[taskID]; exists {
		log.Printf("Unregistered process %d for task %s", process.ProcessID, taskID)
		delete(qp.runningProcesses, taskID)
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
	TaskID      string `json:"task_id"`
	ProcessID   int    `json:"process_id"`
	StartTime   string `json:"start_time"`
	Duration    string `json:"duration"`
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
		"processor_active":    !isPaused && isRunning,
		"maintenance_state":   map[bool]string{true: "inactive", false: "active"}[isPaused],
		"max_concurrent":      maxConcurrent,
		"executing_count":     executingCount,
		"available_slots":     availableSlots,
		"pending_count":       len(pendingTasks),
		"ready_in_progress":   readyInProgress,
		"refresh_interval":    currentSettings.RefreshInterval, // from settings
		"processor_running":   isRunning && !isPaused,
		"timestamp":           time.Now().Unix(),
	}
}