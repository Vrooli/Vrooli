package queue

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"strconv"
	"time"

	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/settings"
)

// executeTask executes a single task
func (qp *Processor) executeTask(task tasks.TaskItem) {
	log.Printf("Executing task %s: %s", task.ID, task.Title)
	
	// Track execution timing
	executionStartTime := time.Now()
	
	// Get timeout setting for timing info
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute
	
	// Generate the full prompt for the task
	prompt, err := qp.assembler.AssemblePromptForTask(task)
	if err != nil {
		executionTime := time.Since(executionStartTime)
		log.Printf("Failed to assemble prompt for task %s: %v", task.ID, err)
		qp.handleTaskFailureWithTiming(task, fmt.Sprintf("Prompt assembly failed: %v", err), executionStartTime, executionTime, timeoutDuration)
		return
	}
	
	// Update task progress
	task.CurrentPhase = "prompt_assembled"
	task.ProgressPercent = 25
	qp.storage.SaveQueueItem(task, "in-progress")
	qp.broadcastUpdate("task_progress", task)
	
	// Call Claude Code resource
	result, err := qp.callClaudeCode(prompt, task, executionStartTime, timeoutDuration)
	executionTime := time.Since(executionStartTime)
	
	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		qp.handleTaskFailureWithTiming(task, fmt.Sprintf("Claude Code execution failed: %v", err), executionStartTime, executionTime, timeoutDuration)
		return
	}
	
	// Process the result
	if result.Success {
		log.Printf("Task %s completed successfully in %v (timeout was %v)", task.ID, executionTime.Round(time.Second), timeoutDuration)
		
		// Update task with results including timing
		task.Results = map[string]interface{}{
			"success":         true,
			"message":         result.Message,
			"output":          result.Output,
			"execution_time":  executionTime.Round(time.Second).String(),
			"timeout_allowed": timeoutDuration.String(),
			"started_at":      executionStartTime.Format(time.RFC3339),
			"completed_at":    time.Now().Format(time.RFC3339),
		}
		task.ProgressPercent = 100
		task.CurrentPhase = "completed"
		task.Status = "completed"
		qp.storage.SaveQueueItem(task, "in-progress")
		qp.broadcastUpdate("task_completed", task)
		
		// Move to completed
		if err := qp.storage.MoveTask(task.ID, "in-progress", "completed"); err != nil {
			log.Printf("Failed to move task %s to completed: %v", task.ID, err)
		}
	} else {
		log.Printf("Task %s failed after %v: %s", task.ID, executionTime.Round(time.Second), result.Error)
		qp.handleTaskFailureWithTiming(task, result.Error, executionStartTime, executionTime, timeoutDuration)
	}
}

// handleTaskFailure handles a failed task (legacy - for backward compatibility)
func (qp *Processor) handleTaskFailure(task tasks.TaskItem, errorMsg string) {
	task.Results = map[string]interface{}{
		"success": false,
		"error":   errorMsg,
	}
	task.CurrentPhase = "failed"
	task.Status = "failed"
	qp.storage.SaveQueueItem(task, "in-progress")
	qp.broadcastUpdate("task_failed", task)
	
	if err := qp.storage.MoveTask(task.ID, "in-progress", "failed"); err != nil {
		log.Printf("Failed to move task %s to failed: %v", task.ID, err)
	}
}

// handleTaskFailureWithTiming handles a failed task with execution timing information
func (qp *Processor) handleTaskFailureWithTiming(task tasks.TaskItem, errorMsg string, startTime time.Time, executionTime time.Duration, timeoutAllowed time.Duration) {
	// Determine if this was a timeout failure
	isTimeout := strings.Contains(errorMsg, "timed out") || strings.Contains(errorMsg, "timeout")
	
	task.Results = map[string]interface{}{
		"success":         false,
		"error":           errorMsg,
		"execution_time":  executionTime.Round(time.Second).String(),
		"timeout_allowed": timeoutAllowed.String(),
		"started_at":      startTime.Format(time.RFC3339),
		"failed_at":       time.Now().Format(time.RFC3339),
		"timeout_failure": isTimeout,
	}
	
	task.CurrentPhase = "failed"
	task.Status = "failed"
	qp.storage.SaveQueueItem(task, "in-progress")
	qp.broadcastUpdate("task_failed", task)
	
	if err := qp.storage.MoveTask(task.ID, "in-progress", "failed"); err != nil {
		log.Printf("Failed to move task %s to failed: %v", task.ID, err)
	}
	
	// Log detailed timing information
	if isTimeout {
		log.Printf("Task %s TIMED OUT after %v (limit was %v)", 
			task.ID, executionTime.Round(time.Second), timeoutAllowed)
	} else {
		log.Printf("Task %s FAILED after %v (limit was %v)", 
			task.ID, executionTime.Round(time.Second), timeoutAllowed)
	}
}

// callClaudeCode calls the Claude Code resource using stdin to avoid argument length limits
func (qp *Processor) callClaudeCode(prompt string, task tasks.TaskItem, startTime time.Time, timeoutDuration time.Duration) (*tasks.ClaudeCodeResponse, error) {
	log.Printf("Executing Claude Code for task %s (prompt length: %d characters, timeout: %v)", task.ID, len(prompt), timeoutDuration)
	
	// Get Vrooli root directory
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		// Fallback to detecting from current path
		if wd, err := os.Getwd(); err == nil {
			// Navigate up to find Vrooli root (contains .vrooli directory)
			for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
				if _, err := os.Stat(filepath.Join(dir, ".vrooli")); err == nil {
					vrooliRoot = dir
					break
				}
			}
		}
	}
	if vrooliRoot == "" {
		vrooliRoot = "." // Fallback to current directory
	}
	
	// Set timeout (passed from caller)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	
	// Use stdin instead of command line argument to avoid "argument list too long"
	cmd := exec.CommandContext(ctx, "resource-claude-code", "run", "-")
	cmd.Dir = vrooliRoot
	
	// Apply settings to Claude execution via environment variables
	currentSettings := settings.GetSettings()
	cmd.Env = append(os.Environ(), 
		"MAX_TURNS="+strconv.Itoa(currentSettings.MaxTurns),
		"ALLOWED_TOOLS="+currentSettings.AllowedTools,
	)
	
	if currentSettings.SkipPermissions {
		cmd.Env = append(cmd.Env, "SKIP_PERMISSIONS=yes")
	} else {
		cmd.Env = append(cmd.Env, "SKIP_PERMISSIONS=no")
	}
	
	log.Printf("Claude execution settings: MAX_TURNS=%d, ALLOWED_TOOLS=%s, SKIP_PERMISSIONS=%v, TIMEOUT=%dm", 
		currentSettings.MaxTurns, currentSettings.AllowedTools, currentSettings.SkipPermissions, currentSettings.TaskTimeout)
	
	// Set up pipes for stdin and stdout
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create stdin pipe: %v", err),
		}, nil
	}
	
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create stdout pipe: %v", err),
		}, nil
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to start Claude Code: %v", err),
		}, nil
	}
	
	// Register the running process for tracking
	qp.registerRunningProcess(task.ID, cmd, ctx, cancel)
	defer qp.unregisterRunningProcess(task.ID) // Always cleanup on exit
	
	// Send prompt via stdin in a goroutine
	go func() {
		defer stdinPipe.Close()
		if _, err := stdinPipe.Write([]byte(prompt)); err != nil {
			log.Printf("Error writing prompt to stdin for task %s: %v", task.ID, err)
		}
	}()
	
	// Read output from stdout
	output, err := io.ReadAll(stdoutPipe)
	if err != nil {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to read output: %v", err),
		}, nil
	}
	
	// Wait for completion
	waitErr := cmd.Wait()
	
	// Handle different exit scenarios
	if ctx.Err() == context.DeadlineExceeded {
		actualRuntime := time.Since(startTime).Round(time.Second)
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("⏰ TIMEOUT: Task execution exceeded %v limit (ran for %v)\n\nThe task was automatically terminated because it exceeded the configured timeout.\nConsider:\n- Increasing timeout in Settings if this is a complex task\n- Breaking the task into smaller parts\n- Checking if task is stuck in an infinite loop", timeoutDuration, actualRuntime),
		}, nil
	}
	
	if waitErr != nil {
		// Check if the process was terminated intentionally
		if _, wasTerminated := qp.getRunningProcess(task.ID); !wasTerminated {
			// Process was terminated by our termination logic
			return &tasks.ClaudeCodeResponse{
				Success: false,
				Error:   "Task execution was cancelled (moved out of in-progress)",
			}, nil
		}
		
		// Extract exit code
		if exitError, ok := waitErr.(*exec.ExitError); ok {
			log.Printf("Claude Code failed with exit code %d: %s", exitError.ExitCode(), string(output))
			return &tasks.ClaudeCodeResponse{
				Success: false,
				Error:   fmt.Sprintf("Claude Code execution failed (exit code %d): %s", exitError.ExitCode(), string(output)),
			}, nil
		}
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to execute Claude Code: %v", waitErr),
		}, nil
	}
	
	// Check output for error patterns even if exit code is 0
	outputStr := string(output)
	
	// Check for common error patterns that might not set exit code
	if strings.Contains(outputStr, "Error: Reached max turns") {
		log.Printf("Claude Code hit max turns limit for task %s", task.ID)
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   "Claude reached maximum turns limit. Consider breaking the task into smaller parts or increasing MAX_TURNS.",
			Output:  outputStr,
		}, nil
	}
	
	if strings.Contains(outputStr, "error:") || strings.Contains(outputStr, "Error:") {
		log.Printf("Claude Code returned error for task %s: %s", task.ID, outputStr)
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   "Claude execution failed - check output for details",
			Output:  outputStr,
		}, nil
	}
	
	// Success case
	log.Printf("Claude Code completed successfully for task %s (output length: %d characters)", task.ID, len(outputStr))
	
	return &tasks.ClaudeCodeResponse{
		Success: true,
		Message: "Task completed successfully",
		Output:  outputStr,
	}, nil
}

// broadcastUpdate sends updates to all connected WebSocket clients
func (qp *Processor) broadcastUpdate(updateType string, data interface{}) {
	update := map[string]interface{}{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}
	
	// Non-blocking send to broadcast channel
	select {
	case qp.broadcast <- update:
	default:
		log.Printf("Warning: WebSocket broadcast channel full, dropping update")
	}
}