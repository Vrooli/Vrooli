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
	"time"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// executeTask executes a single task
func (qp *Processor) executeTask(task tasks.TaskItem) {
	log.Printf("Executing task %s: %s", task.ID, task.Title)
	
	// Generate the full prompt for the task
	prompt, err := qp.assembler.AssemblePromptForTask(task)
	if err != nil {
		log.Printf("Failed to assemble prompt for task %s: %v", task.ID, err)
		qp.handleTaskFailure(task, fmt.Sprintf("Prompt assembly failed: %v", err))
		return
	}
	
	// Update task progress
	task.CurrentPhase = "prompt_assembled"
	task.ProgressPercent = 25
	qp.storage.SaveQueueItem(task, "in-progress")
	qp.broadcastUpdate("task_progress", task)
	
	// Call Claude Code resource
	result, err := qp.callClaudeCode(prompt, task)
	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		qp.handleTaskFailure(task, fmt.Sprintf("Claude Code execution failed: %v", err))
		return
	}
	
	// Process the result
	if result.Success {
		log.Printf("Task %s completed successfully", task.ID)
		
		// Update task with results
		task.Results = map[string]interface{}{
			"success": true,
			"message": result.Message,
			"output":  result.Output,
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
		log.Printf("Task %s failed: %s", task.ID, result.Error)
		qp.handleTaskFailure(task, result.Error)
	}
}

// handleTaskFailure handles a failed task
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

// callClaudeCode calls the Claude Code resource using stdin to avoid argument length limits
func (qp *Processor) callClaudeCode(prompt string, task tasks.TaskItem) (*tasks.ClaudeCodeResponse, error) {
	log.Printf("Executing Claude Code for task %s (prompt length: %d characters)", task.ID, len(prompt))
	
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
	
	// Set timeout (30 minutes for complex tasks)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	// Use stdin instead of command line argument to avoid "argument list too long"
	cmd := exec.CommandContext(ctx, "resource-claude-code", "run", "-")
	cmd.Dir = vrooliRoot
	
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
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   "Claude Code execution timed out after 30 minutes",
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