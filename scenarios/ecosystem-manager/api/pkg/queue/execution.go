package queue

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/tasks"
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

	// Store the assembled prompt in /tmp for debugging
	promptPath := filepath.Join("/tmp", fmt.Sprintf("ecosystem-prompt-%s.txt", task.ID))
	if err := os.WriteFile(promptPath, []byte(prompt), 0644); err != nil {
		log.Printf("Warning: Failed to save prompt to %s: %v", promptPath, err)
		// Don't fail the task, just log the warning
	} else {
		log.Printf("Saved assembled prompt to %s", promptPath)
	}

	// Update task progress
	task.CurrentPhase = "prompt_assembled"
	task.ProgressPercent = 25
	qp.storage.SaveQueueItem(task, "in-progress")
	qp.broadcastUpdate("task_progress", task)

	// Log prompt size for debugging
	promptSizeKB := float64(len(prompt)) / 1024.0
	promptSizeMB := promptSizeKB / 1024.0
	log.Printf("Task %s: Prompt size: %d characters (%.2f KB / %.2f MB)", task.ID, len(prompt), promptSizeKB, promptSizeMB)

	// Call Claude Code resource
	result, err := qp.callClaudeCode(prompt, task, executionStartTime, timeoutDuration)
	executionTime := time.Since(executionStartTime)

	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		qp.handleTaskFailureWithTiming(task, fmt.Sprintf("Claude Code execution failed: %v", err), executionStartTime, executionTime, timeoutDuration)
		return
	}

	// Process the result
	// Debug: always log the execution result for debugging
	log.Printf("🔍 Task %s execution result: Success=%v, RateLimited=%v, Error=%q", 
		task.ID, result.Success, result.RateLimited, result.Error)
	
	if result.Success {
		log.Printf("Task %s completed successfully in %v (timeout was %v)", task.ID, executionTime.Round(time.Second), timeoutDuration)

		// Update task with results including timing and prompt size
		task.Results = map[string]interface{}{
			"success":         true,
			"message":         result.Message,
			"output":          result.Output,
			"execution_time":  executionTime.Round(time.Second).String(),
			"timeout_allowed": timeoutDuration.String(),
			"started_at":      executionStartTime.Format(time.RFC3339),
			"completed_at":    time.Now().Format(time.RFC3339),
			"prompt_size":     fmt.Sprintf("%d chars (%.2f KB)", len(prompt), promptSizeKB),
		}
		task.ProgressPercent = 100
		task.CurrentPhase = "completed"
		task.Status = "completed"
		if err := qp.storage.SaveQueueItem(task, "in-progress"); err != nil {
			log.Printf("ERROR: Failed to save completed task %s: %v", task.ID, err)
			// Still try to move the task even if save failed
		}
		qp.broadcastUpdate("task_completed", task)

		// Move to completed
		if err := qp.storage.MoveTask(task.ID, "in-progress", "completed"); err != nil {
			log.Printf("Failed to move task %s to completed: %v", task.ID, err)
		}
	} else {
		// Check if this is a rate limit error
		if result.RateLimited {
			log.Printf("🚫 Task %s hit rate limit. Pausing queue for %d seconds", task.ID, result.RetryAfter)
			
			// Move task back to pending (don't mark as failed)
			task.CurrentPhase = "rate_limited"
			task.Notes = fmt.Sprintf("Rate limited at %s. Will retry after %d seconds.", 
				time.Now().Format(time.RFC3339), result.RetryAfter)
			qp.storage.SaveQueueItem(task, "in-progress")
			
			// Move back to pending queue for retry
			if err := qp.storage.MoveTask(task.ID, "in-progress", "pending"); err != nil {
				log.Printf("Failed to move rate-limited task %s back to pending: %v", task.ID, err)
			}
			
			// Trigger a pause of the queue processor
			qp.handleRateLimitPause(result.RetryAfter)
			
			// Broadcast rate limit event
			qp.broadcastUpdate("rate_limit_hit", map[string]interface{}{
				"task_id":     task.ID,
				"retry_after": result.RetryAfter,
				"pause_until": time.Now().Add(time.Duration(result.RetryAfter) * time.Second).Format(time.RFC3339),
			})
		} else {
			log.Printf("Task %s failed after %v: %s", task.ID, executionTime.Round(time.Second), result.Error)
			qp.handleTaskFailureWithTiming(task, result.Error, executionStartTime, executionTime, timeoutDuration)
		}
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
	if err := qp.storage.SaveQueueItem(task, "in-progress"); err != nil {
		log.Printf("ERROR: Failed to save failed task %s: %v", task.ID, err)
		// Still try to move the task even if save failed
	}
	qp.broadcastUpdate("task_failed", task)

	if err := qp.storage.MoveTask(task.ID, "in-progress", "failed"); err != nil {
		log.Printf("Failed to move task %s to failed: %v", task.ID, err)
	}
}

// handleTaskFailureWithTiming handles a failed task with execution timing information
func (qp *Processor) handleTaskFailureWithTiming(task tasks.TaskItem, errorMsg string, startTime time.Time, executionTime time.Duration, timeoutAllowed time.Duration) {
	// Determine if this was a timeout failure
	isTimeout := strings.Contains(errorMsg, "timed out") || strings.Contains(errorMsg, "timeout")

	// Get the prompt for size calculation (if available)
	prompt, _ := qp.assembler.AssemblePromptForTask(task)
	promptSizeKB := float64(len(prompt)) / 1024.0

	task.Results = map[string]interface{}{
		"success":         false,
		"error":           errorMsg,
		"execution_time":  executionTime.Round(time.Second).String(),
		"timeout_allowed": timeoutAllowed.String(),
		"started_at":      startTime.Format(time.RFC3339),
		"failed_at":       time.Now().Format(time.RFC3339),
		"timeout_failure": isTimeout,
		"prompt_size":     fmt.Sprintf("%d chars (%.2f KB)", len(prompt), promptSizeKB),
	}

	task.CurrentPhase = "failed"
	task.Status = "failed"
	if err := qp.storage.SaveQueueItem(task, "in-progress"); err != nil {
		log.Printf("ERROR: Failed to save failed task %s with timing: %v", task.ID, err)
		// Still try to move the task even if save failed
	}
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
	// Pass the timeout in seconds to resource-claude-code via environment variable
	timeoutSeconds := int(timeoutDuration.Seconds())
	cmd := exec.CommandContext(ctx, "resource-claude-code", "run", "-")
	cmd.Dir = vrooliRoot

	// Apply settings to Claude execution via environment variables
	currentSettings := settings.GetSettings()
	cmd.Env = append(os.Environ(),
		"MAX_TURNS="+strconv.Itoa(currentSettings.MaxTurns),
		"ALLOWED_TOOLS="+currentSettings.AllowedTools,
		"TIMEOUT="+strconv.Itoa(timeoutSeconds),
	)

	if currentSettings.SkipPermissions {
		cmd.Env = append(cmd.Env, "SKIP_PERMISSIONS=yes")
	} else {
		cmd.Env = append(cmd.Env, "SKIP_PERMISSIONS=no")
	}

	log.Printf("Claude execution settings: MAX_TURNS=%d, ALLOWED_TOOLS=%s, SKIP_PERMISSIONS=%v, TIMEOUT=%ds (%dm)",
		currentSettings.MaxTurns, currentSettings.AllowedTools, currentSettings.SkipPermissions, timeoutSeconds, currentSettings.TaskTimeout)
	log.Printf("Working directory: %s", cmd.Dir)
	log.Printf("Full environment: %v", cmd.Env)

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

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create stderr pipe: %v", err),
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

	// Read output from stdout and stderr
	output, err := io.ReadAll(stdoutPipe)
	if err != nil {
		// Ensure process is terminated on read error
		log.Printf("Failed to read stdout for task %s, terminating process: %v", task.ID, err)
		if cancel != nil {
			cancel() // Signal context cancellation
		}
		// Give process time to exit gracefully, then force kill if needed
		time.Sleep(100 * time.Millisecond)
		if cmd.Process != nil && cmd.ProcessState == nil {
			cmd.Process.Kill()
		}
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to read output: %v", err),
		}, nil
	}

	stderrOutput, err := io.ReadAll(stderrPipe)
	if err != nil {
		log.Printf("Warning: Failed to read stderr: %v", err)
		stderrOutput = []byte("(failed to read stderr)")
	}

	// Wait for completion
	waitErr := cmd.Wait()

	// Log the exit details for debugging
	if waitErr != nil {
		log.Printf("Command failed with error: %v", waitErr)
		log.Printf("STDERR: %s", string(stderrOutput))
		if exitError, ok := waitErr.(*exec.ExitError); ok {
			log.Printf("Exit code: %d", exitError.ExitCode())
		}
	} else {
		log.Printf("Command completed successfully")
	}

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

		// Extract exit code and check for rate limits in the error path too
		combinedOutput := string(output)
		if len(stderrOutput) > 0 {
			combinedOutput += "\n\nSTDERR:\n" + string(stderrOutput)
		}
		
		// Check for rate limits in error path
		lowerOutput := strings.ToLower(combinedOutput)
		isRateLimit := strings.Contains(lowerOutput, "usage limit") ||
					   strings.Contains(lowerOutput, "rate limit") ||
					   strings.Contains(lowerOutput, "ai usage limit reached") ||
					   strings.Contains(lowerOutput, "rate/usage limit reached") ||
					   strings.Contains(lowerOutput, "usage limit reached") ||
					   strings.Contains(lowerOutput, "claude ai usage limit reached") ||
					   strings.Contains(lowerOutput, "you've reached your claude usage limit") ||
					   strings.Contains(lowerOutput, "429") ||
					   strings.Contains(lowerOutput, "too many requests") ||
					   strings.Contains(lowerOutput, "quota exceeded") ||
					   strings.Contains(lowerOutput, "rate limits are critical")
		
		if exitError, ok := waitErr.(*exec.ExitError); ok {
			// Exit code 429 is often used for rate limits
			if exitError.ExitCode() == 429 {
				isRateLimit = true
			}
			
			if isRateLimit {
				log.Printf("🚫 Rate limit detected in error path for task %s (exit code %d)", task.ID, exitError.ExitCode())
				log.Printf("Rate limit output: %s", combinedOutput)
				return &tasks.ClaudeCodeResponse{
					Success:       false,
					Error:        "RATE_LIMIT: API rate limit reached",
					RateLimited:  true,
					RetryAfter:   qp.extractRetryAfter(combinedOutput),
					Output:       string(output),
				}, nil
			}
			
			log.Printf("Claude Code failed with exit code %d: %s", exitError.ExitCode(), combinedOutput)
			return &tasks.ClaudeCodeResponse{
				Success: false,
				Error:   fmt.Sprintf("Claude Code execution failed (exit code %d): %s", exitError.ExitCode(), combinedOutput),
			}, nil
		}
		
		if isRateLimit {
			log.Printf("🚫 Rate limit detected in error path for task %s", task.ID)
			log.Printf("Rate limit output: %s", combinedOutput)
			return &tasks.ClaudeCodeResponse{
				Success:       false,
				Error:        "RATE_LIMIT: API rate limit reached",
				RateLimited:  true,
				RetryAfter:   qp.extractRetryAfter(combinedOutput),
				Output:       string(output),
			}, nil
		}
		
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to execute Claude Code: %v", waitErr),
		}, nil
	}

	// Check output for error patterns even if exit code is 0
	outputStr := string(output)
	stderrStr := string(stderrOutput)
	combinedOutput := outputStr + "\n" + stderrStr

	// Check for rate limit errors FIRST - be more comprehensive
	lowerOutput := strings.ToLower(combinedOutput)
	isRateLimit := strings.Contains(lowerOutput, "usage limit") ||
				   strings.Contains(lowerOutput, "rate limit") ||
				   strings.Contains(lowerOutput, "ai usage limit reached") ||
				   strings.Contains(lowerOutput, "rate/usage limit reached") ||
				   strings.Contains(lowerOutput, "usage limit reached") ||
				   strings.Contains(lowerOutput, "claude ai usage limit reached") ||
				   strings.Contains(lowerOutput, "you've reached your claude usage limit") ||
				   strings.Contains(lowerOutput, "429") ||
				   strings.Contains(lowerOutput, "too many requests") ||
				   strings.Contains(lowerOutput, "quota exceeded") ||
				   strings.Contains(lowerOutput, "rate limits are critical")
	
	// Also check for exit code patterns that indicate rate limiting
	if waitErr != nil {
		if exitError, ok := waitErr.(*exec.ExitError); ok {
			// Exit code 429 is often used for rate limits
			if exitError.ExitCode() == 429 {
				isRateLimit = true
			}
		}
	}
	
	if isRateLimit {
		log.Printf("🚫 Rate limit detected for task %s", task.ID)
		log.Printf("Rate limit output: %s", combinedOutput)
		return &tasks.ClaudeCodeResponse{
			Success:       false,
			Error:        "RATE_LIMIT: API rate limit reached",
			RateLimited:  true,
			RetryAfter:   qp.extractRetryAfter(combinedOutput),
			Output:       outputStr,
		}, nil
	}

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

// extractRetryAfter attempts to extract retry-after duration from rate limit error messages
func (qp *Processor) extractRetryAfter(output string) int {
	// Default to 30 minutes if we can't parse
	defaultRetry := 1800 // 30 minutes in seconds
	lowerOutput := strings.ToLower(output)
	
	// Look for common time patterns
	if strings.Contains(lowerOutput, "5 hour") || strings.Contains(lowerOutput, "5-hour") || 
	   strings.Contains(lowerOutput, "every 5 hours") {
		return 5 * 3600 // 5 hours
	}
	
	if strings.Contains(lowerOutput, "4 hour") || strings.Contains(lowerOutput, "4-hour") {
		return 4 * 3600 // 4 hours  
	}
	
	if strings.Contains(lowerOutput, "1 hour") || strings.Contains(lowerOutput, "1-hour") {
		return 3600 // 1 hour
	}
	
	// Look for "retry_after" or "retry-after" patterns
	if strings.Contains(lowerOutput, "retry") && (strings.Contains(lowerOutput, "after") || strings.Contains(lowerOutput, "_after")) {
		// Try to extract number after retry_after or retry-after
		parts := strings.FieldsFunc(output, func(r rune) bool {
			return r == ':' || r == '=' || r == ' ' || r == '\t' || r == '\n'
		})
		
		for i, part := range parts {
			if strings.Contains(strings.ToLower(part), "retry") && i+1 < len(parts) {
				if seconds, err := strconv.Atoi(strings.Trim(parts[i+1], "\"'")); err == nil && seconds > 0 {
					// Cap at 4 hours maximum
					if seconds > 14400 {
						return 14400
					}
					// Minimum 5 minutes
					if seconds < 300 {
						return 300 
					}
					return seconds
				}
			}
		}
	}
	
	// If we see "critical" rate limits, use a longer default
	if strings.Contains(lowerOutput, "critical") {
		return 3600 // 1 hour for critical rate limits
	}
	
	return defaultRetry
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
