package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ProcessManager handles robust process lifecycle management
type ProcessManager struct {
	mu              sync.Mutex
	activeProcesses map[string]*ManagedProcess
}

// ManagedProcess wraps a process with management capabilities
type ManagedProcess struct {
	TaskID    string
	Cmd       *exec.Cmd
	Context   context.Context
	Cancel    context.CancelFunc
	StartTime time.Time
	WaitDone  chan struct{} // Signals when Wait() has completed
	WaitErr   chan error    // Receives the result from Wait()
}

// NewProcessManager creates a new process manager
func NewProcessManager() *ProcessManager {
	// Initialize the global process reaper
	InitProcessReaper()

	return &ProcessManager{
		activeProcesses: make(map[string]*ManagedProcess),
	}
}

// StartProcess starts a managed process with proper setup
// Returns a channel that will receive the wait result (nil on success)
func (pm *ProcessManager) StartProcess(taskID string, cmd *exec.Cmd, ctx context.Context, cancel context.CancelFunc) (<-chan error, error) {
	// Check if the process is already started
	if cmd.Process == nil {
		// Set up process group for better control
		SetProcessGroup(cmd)

		// Start the command
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start process: %w", err)
		}
	}

	// Create managed process
	mp := &ManagedProcess{
		TaskID:    taskID,
		Cmd:       cmd,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		WaitDone:  make(chan struct{}),
		WaitErr:   make(chan error, 1),
	}

	// Register the process
	pm.mu.Lock()
	pm.activeProcesses[taskID] = mp
	pm.mu.Unlock()

	// Start a goroutine to wait for the process
	// This ensures we always call Wait() to prevent zombies
	go func() {
		defer close(mp.WaitDone)
		defer close(mp.WaitErr)

		// Wait for the process to complete
		err := waitWithReapedFallback(cmd)

		// Log completion
		if err != nil {
			log.Printf("Process %d for task %s exited with error: %v", cmd.Process.Pid, taskID, err)
		} else {
			log.Printf("Process %d for task %s completed successfully", cmd.Process.Pid, taskID)
		}

		// Send the wait result to listeners
		mp.WaitErr <- err

		// Unregister the process
		pm.mu.Lock()
		delete(pm.activeProcesses, taskID)
		pm.mu.Unlock()
	}()

	log.Printf("Started managed process %d for task %s", cmd.Process.Pid, taskID)
	return mp.WaitErr, nil
}

// StartProcessWithCoordination starts a process but waits for pipe reading to complete before calling Wait()
func (pm *ProcessManager) StartProcessWithCoordination(taskID string, cmd *exec.Cmd, ctx context.Context, cancel context.CancelFunc, readsDone <-chan struct{}) (<-chan error, error) {
	if cmd.Process == nil {
		return nil, fmt.Errorf("command not started")
	}

	// Create managed process
	mp := &ManagedProcess{
		TaskID:    taskID,
		Cmd:       cmd,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		WaitDone:  make(chan struct{}),
		WaitErr:   make(chan error, 1),
	}

	// Register the process
	pm.mu.Lock()
	pm.activeProcesses[taskID] = mp
	pm.mu.Unlock()

	// Start a goroutine to wait for the process
	// This ensures we always call Wait() to prevent zombies
	// BUT we wait for pipe reading to complete first
	go func() {
		defer close(mp.WaitDone)
		defer close(mp.WaitErr)

		// CRITICAL: Wait for all pipe reading to complete before calling Wait()
		// This prevents the "file already closed" race condition
		<-readsDone

		// Now it's safe to call Wait() - pipes have been read
		err := waitWithReapedFallback(cmd)

		// Log completion
		if err != nil {
			log.Printf("Process %d for task %s exited with error: %v", cmd.Process.Pid, taskID, err)
		} else {
			log.Printf("Process %d for task %s completed successfully", cmd.Process.Pid, taskID)
		}

		// Send wait result for downstream consumers
		mp.WaitErr <- err

		// Unregister the process
		pm.mu.Lock()
		delete(pm.activeProcesses, taskID)
		pm.mu.Unlock()
	}()

	log.Printf("Started coordinated managed process %d for task %s", cmd.Process.Pid, taskID)
	return mp.WaitErr, nil
}

// waitWithReapedFallback ensures we surface the correct exit status even if the
// process has already been reaped by the SIGCHLD handler.
func waitWithReapedFallback(cmd *exec.Cmd) error {
	err := cmd.Wait()
	if err == nil {
		return nil
	}

	if !isNoChildError(err) {
		return err
	}

	if cmd.Process == nil {
		return nil
	}

	if status, ok := ConsumeReapedStatus(cmd.Process.Pid); ok {
		if status.Exited() {
			exitCode := status.ExitStatus()
			if exitCode == 0 {
				return nil
			}
			return &waitStatusError{pid: cmd.Process.Pid, status: status}
		}

		if status.Signaled() {
			return &waitStatusError{pid: cmd.Process.Pid, status: status}
		}
	}

	return nil
}

func isNoChildError(err error) bool {
	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		return syscallErr.Err == syscall.ECHILD
	}

	if errors.Is(err, syscall.ECHILD) {
		return true
	}

	if strings.Contains(strings.ToLower(err.Error()), "no child processes") {
		return true
	}

	return false
}

type waitStatusError struct {
	pid    int
	status syscall.WaitStatus
}

func (e *waitStatusError) Error() string {
	if e.status.Signaled() {
		return fmt.Sprintf("process %d terminated by signal %d", e.pid, e.status.Signal())
	}
	return fmt.Sprintf("process %d exited with status %d", e.pid, e.status.ExitStatus())
}

func (e *waitStatusError) ExitStatus() int {
	return e.status.ExitStatus()
}

func (e *waitStatusError) Signal() syscall.Signal {
	if e.status.Signaled() {
		return e.status.Signal()
	}
	return 0
}

// TerminateProcess gracefully terminates a process with escalation
func (pm *ProcessManager) TerminateProcess(taskID string, timeout time.Duration) error {
	pm.mu.Lock()
	mp, exists := pm.activeProcesses[taskID]
	pm.mu.Unlock()

	if !exists {
		return fmt.Errorf("no active process found for task %s", taskID)
	}

	pid := mp.Cmd.Process.Pid
	log.Printf("Terminating process %d for task %s", pid, taskID)

	// Cancel context first
	if mp.Cancel != nil {
		mp.Cancel()
	}

	// Try graceful termination with SIGTERM
	if err := mp.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM to process %d: %v", pid, err)
	}

	// Wait for process to exit or timeout
	select {
	case <-mp.WaitDone:
		log.Printf("Process %d terminated gracefully", pid)
		return nil

	case <-time.After(timeout):
		log.Printf("Process %d didn't terminate within %v, escalating to SIGKILL", pid, timeout)

		// Try to kill the entire process group
		if err := KillProcessGroup(pid); err != nil {
			log.Printf("Failed to kill process group: %v", err)
			// Fall back to killing just the process
			if err := mp.Cmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill process %d: %v", pid, err)
			}
		}

		// Wait a bit more for the kill to take effect
		select {
		case <-mp.WaitDone:
			log.Printf("Process %d killed successfully", pid)
			return nil
		case <-time.After(1 * time.Second):
			// Process is likely already dead but Wait() hasn't been called
			// The reaper will handle it
			log.Printf("Process %d may be zombie, relying on reaper", pid)
			return fmt.Errorf("process may be zombie")
		}
	}
}

// TerminateAll terminates all managed processes
func (pm *ProcessManager) TerminateAll(timeout time.Duration) {
	pm.mu.Lock()
	tasks := make([]string, 0, len(pm.activeProcesses))
	for taskID := range pm.activeProcesses {
		tasks = append(tasks, taskID)
	}
	pm.mu.Unlock()

	var wg sync.WaitGroup
	for _, taskID := range tasks {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := pm.TerminateProcess(id, timeout); err != nil {
				log.Printf("Error terminating process for task %s: %v", id, err)
			}
		}(taskID)
	}

	wg.Wait()
	log.Printf("All processes terminated")
}

// GetActiveCount returns the number of active processes
func (pm *ProcessManager) GetActiveCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.activeProcesses)
}

// IsProcessActive checks if a process is still active
func (pm *ProcessManager) IsProcessActive(taskID string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	_, exists := pm.activeProcesses[taskID]
	return exists
}
