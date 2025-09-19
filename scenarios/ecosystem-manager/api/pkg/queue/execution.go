package queue

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
		qp.handleTaskFailureWithTiming(task, fmt.Sprintf("Prompt assembly failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil)
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
		qp.handleTaskFailureWithTiming(task, fmt.Sprintf("Claude Code execution failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil)
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
		task.CompletedAt = time.Now().Format(time.RFC3339)
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
			// Store rate limit info in results, NOT in notes (preserve user notes)
			if task.Results == nil {
				task.Results = make(map[string]interface{})
			}
			task.Results["rate_limit_info"] = map[string]interface{}{
				"hit_at":      time.Now().Format(time.RFC3339),
				"retry_after": result.RetryAfter,
				"message":     fmt.Sprintf("Rate limited at %s. Will retry after %d seconds.", time.Now().Format(time.RFC3339), result.RetryAfter),
			}
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
			var extras map[string]interface{}
			if result.MaxTurnsExceeded {
				extras = map[string]interface{}{"max_turns_exceeded": true}
			}
			qp.handleTaskFailureWithTiming(task, result.Error, result.Output, executionStartTime, executionTime, timeoutDuration, extras)
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
func (qp *Processor) handleTaskFailureWithTiming(task tasks.TaskItem, errorMsg string, output string, startTime time.Time, executionTime time.Duration, timeoutAllowed time.Duration, extras map[string]interface{}) {
	// Determine if this was a timeout failure
	isTimeout := strings.Contains(errorMsg, "timed out") || strings.Contains(errorMsg, "timeout")

	// Get the prompt for size calculation (if available)
	prompt, _ := qp.assembler.AssemblePromptForTask(task)
	promptSizeKB := float64(len(prompt)) / 1024.0

	results := map[string]interface{}{
		"success":         false,
		"error":           errorMsg,
		"output":          output,
		"execution_time":  executionTime.Round(time.Second).String(),
		"timeout_allowed": timeoutAllowed.String(),
		"started_at":      startTime.Format(time.RFC3339),
		"failed_at":       time.Now().Format(time.RFC3339),
		"timeout_failure": isTimeout,
		"prompt_size":     fmt.Sprintf("%d chars (%.2f KB)", len(prompt), promptSizeKB),
	}
	for k, v := range extras {
		results[k] = v
	}

	task.Results = results

	task.CurrentPhase = "failed"
	task.Status = "failed"
	task.CompletedAt = time.Now().Format(time.RFC3339)
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

// callClaudeCode executes Claude Code while streaming logs for real-time monitoring
func (qp *Processor) callClaudeCode(prompt string, task tasks.TaskItem, startTime time.Time, timeoutDuration time.Duration) (*tasks.ClaudeCodeResponse, error) {
	log.Printf("Executing Claude Code for task %s (prompt length: %d characters, timeout: %v)", task.ID, len(prompt), timeoutDuration)

	// Resolve Vrooli root to ensure resource CLI works no matter where binary runs
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		if wd, err := os.Getwd(); err == nil {
			for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
				if _, statErr := os.Stat(filepath.Join(dir, ".vrooli")); statErr == nil {
					vrooliRoot = dir
					break
				}
			}
		}
	}
	if vrooliRoot == "" {
		vrooliRoot = "."
	}

	timeoutSeconds := int(timeoutDuration.Seconds())
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	agentTag := fmt.Sprintf("ecosystem-%s", task.ID)
	cmd := exec.CommandContext(ctx, "resource-claude-code", "run", "--tag", agentTag, "-")
	cmd.Dir = vrooliRoot

	currentSettings := settings.GetSettings()
	skipPermissionsValue := "no"
	if currentSettings.SkipPermissions {
		skipPermissionsValue = "yes"
	}

	cmd.Env = append(os.Environ(),
		"MAX_TURNS="+strconv.Itoa(currentSettings.MaxTurns),
		"ALLOWED_TOOLS="+currentSettings.AllowedTools,
		"TIMEOUT="+strconv.Itoa(timeoutSeconds),
		"SKIP_PERMISSIONS="+skipPermissionsValue,
		"AGENT_TAG="+agentTag,
	)

	log.Printf("Claude execution settings: MAX_TURNS=%d, ALLOWED_TOOLS=%s, SKIP_PERMISSIONS=%v, TIMEOUT=%ds (%dm)",
		currentSettings.MaxTurns, currentSettings.AllowedTools, currentSettings.SkipPermissions, timeoutSeconds, currentSettings.TaskTimeout)
	log.Printf("Working directory: %s", cmd.Dir)

	cmd.Stdin = strings.NewReader(prompt)

	// Ensure the agent runs in its own process group so we can terminate it cleanly
	SetProcessGroup(cmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to obtain stdout pipe for task %s: %v", task.ID, err)
		return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to start Claude Code stdout pipe: %v", err)}, nil
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to obtain stderr pipe for task %s: %v", task.ID, err)
		return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to start Claude Code stderr pipe: %v", err)}, nil
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start Claude Code for task %s: %v", task.ID, err)
		return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to start Claude Code: %v", err)}, nil
	}

	qp.registerRunningProcess(task.ID, cmd, ctx, cancel, agentTag)
	defer qp.unregisterRunningProcess(task.ID)

	qp.initTaskLogBuffer(task.ID, agentTag, cmd.Process.Pid)
	defer qp.markTaskLogsCompleted(task.ID)

	qp.appendTaskLog(task.ID, agentTag, "stdout", fmt.Sprintf("▶ Claude Code agent %s started (pid %d)", agentTag, cmd.Process.Pid))
	qp.broadcastUpdate("task_started", map[string]interface{}{
		"task_id":    task.ID,
		"agent_id":   agentTag,
		"process_id": cmd.Process.Pid,
		"start_time": startTime.Format(time.RFC3339),
	})

	task.CurrentPhase = "executing_claude"
	task.ProgressPercent = 50
	task.StartedAt = startTime.Format(time.RFC3339)
	if err := qp.storage.SaveQueueItem(task, "in-progress"); err != nil {
		log.Printf("Warning: failed to persist in-progress task %s: %v", task.ID, err)
	}
	qp.broadcastUpdate("task_executing", task)

	var stdoutBuilder, stderrBuilder, combinedBuilder strings.Builder
	var combinedMu sync.Mutex

	streamPipe := func(stream string, reader io.ReadCloser) {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			qp.appendTaskLog(task.ID, agentTag, stream, line)
			combinedMu.Lock()
			if stream == "stderr" {
				stderrBuilder.WriteString(line)
				stderrBuilder.WriteByte('\n')
			} else {
				stdoutBuilder.WriteString(line)
				stdoutBuilder.WriteByte('\n')
			}
			combinedBuilder.WriteString(line)
			combinedBuilder.WriteByte('\n')
			combinedMu.Unlock()
		}
		if scanErr := scanner.Err(); scanErr != nil && scanErr != io.EOF {
			log.Printf("Error reading %s for task %s: %v", stream, task.ID, scanErr)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); streamPipe("stdout", stdoutPipe) }()
	go func() { defer wg.Done(); streamPipe("stderr", stderrPipe) }()

	readsDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(readsDone)
	}()

	waitErrChan, pmErr := qp.processManager.StartProcessWithCoordination(task.ID, cmd, ctx, cancel, readsDone)
	if pmErr != nil {
		log.Printf("Process manager coordination failed for task %s: %v (falling back to direct wait)", task.ID, pmErr)
		fallbackChan := make(chan error, 1)
		go func() {
			fallbackChan <- cmd.Wait()
			close(fallbackChan)
		}()
		waitErrChan = fallbackChan
	}

	waitErr := <-waitErrChan
	wg.Wait()

	stdoutOutput := stdoutBuilder.String()
	stderrOutput := stderrBuilder.String()
	combinedOutput := combinedBuilder.String()
	if strings.TrimSpace(combinedOutput) == "" && strings.TrimSpace(stdoutOutput) == "" && strings.TrimSpace(stderrOutput) == "" {
		combinedOutput = "(no output captured from Claude Code)"
	}

	// Timeout takes precedence regardless of wait error
	if ctx.Err() == context.DeadlineExceeded {
		actualRuntime := time.Since(startTime).Round(time.Second)
		msg := fmt.Sprintf("⏰ TIMEOUT: Task execution exceeded %v limit (ran for %v)\n\nThe task was automatically terminated because it exceeded the configured timeout.\nConsider:\n- Increasing timeout in Settings if this is a complex task\n- Breaking the task into smaller parts\n- Checking for blocking steps in the prompt", timeoutDuration, actualRuntime)
		qp.appendTaskLog(task.ID, agentTag, "stderr", msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg, Output: combinedOutput}, nil
	}

	// Detect manual termination (context cancelled) before evaluating generic errors
	if ctx.Err() == context.Canceled {
		qp.appendTaskLog(task.ID, agentTag, "stderr", "⛔ Task execution was cancelled")
		return &tasks.ClaudeCodeResponse{Success: false, Error: "Task execution was cancelled", Output: combinedOutput}, nil
	}

	if waitErr != nil {
		if response, handled := qp.handleNonZeroExit(waitErr, combinedOutput, task, agentTag, currentSettings.MaxTurns); handled {
			return response, nil
		}

		log.Printf("Claude Code process errored for task %s: %v", task.ID, waitErr)
		return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to execute Claude Code: %v", waitErr), Output: combinedOutput}, nil
	}

	lowerOutput := strings.ToLower(combinedOutput)
	if strings.Contains(lowerOutput, "usage limit") ||
		strings.Contains(lowerOutput, "rate limit") ||
		strings.Contains(lowerOutput, "ai usage limit reached") ||
		strings.Contains(lowerOutput, "rate/usage limit reached") ||
		strings.Contains(lowerOutput, "claude ai usage limit reached") ||
		strings.Contains(lowerOutput, "you've reached your claude usage limit") ||
		strings.Contains(lowerOutput, "429") ||
		strings.Contains(lowerOutput, "too many requests") ||
		strings.Contains(lowerOutput, "quota exceeded") ||
		strings.Contains(lowerOutput, "rate limits are critical") {
		retryAfter := qp.extractRetryAfter(combinedOutput)
		qp.appendTaskLog(task.ID, agentTag, "stderr", fmt.Sprintf("🚫 Rate limit detected. Suggested backoff %d seconds", retryAfter))
		return &tasks.ClaudeCodeResponse{
			Success:     false,
			Error:       "RATE_LIMIT: API rate limit reached",
			RateLimited: true,
			RetryAfter:  retryAfter,
			Output:      combinedOutput,
		}, nil
	}

	if detectMaxTurnsExceeded(combinedOutput) {
		maxTurnsMsg := fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", currentSettings.MaxTurns)
		qp.appendTaskLog(task.ID, agentTag, "stderr", maxTurnsMsg)
		return &tasks.ClaudeCodeResponse{
			Success:          false,
			Error:            maxTurnsMsg,
			Output:           combinedOutput,
			MaxTurnsExceeded: true,
		}, nil
	}

	if strings.Contains(strings.ToLower(combinedOutput), "error:") {
		msg := "Claude execution reported an error – review output for details."
		qp.appendTaskLog(task.ID, agentTag, "stderr", msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg, Output: combinedOutput}, nil
	}

	log.Printf("Claude Code completed successfully for task %s (output length: %d characters)", task.ID, len(combinedOutput))
	qp.appendTaskLog(task.ID, agentTag, "stdout", "✅ Claude Code execution finished")

	task.CurrentPhase = "claude_completed"
	task.ProgressPercent = 90
	if err := qp.storage.SaveQueueItem(task, "in-progress"); err != nil {
		log.Printf("Warning: failed to persist claude completion for task %s: %v", task.ID, err)
	}
	qp.broadcastUpdate("claude_execution_complete", task)

	return &tasks.ClaudeCodeResponse{
		Success: true,
		Message: "Task completed successfully",
		Output:  combinedOutput,
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

func (qp *Processor) handleNonZeroExit(waitErr error, combinedOutput string, task tasks.TaskItem, agentTag string, maxTurns int) (*tasks.ClaudeCodeResponse, bool) {
	// Determine exit code if available
	exitCode, hasExit := exitCodeFromError(waitErr)
	if !hasExit {
		return nil, false
	}

	if detectMaxTurnsExceeded(combinedOutput) {
		msg := fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", maxTurns)
		qp.appendTaskLog(task.ID, agentTag, "stderr", msg)
		return &tasks.ClaudeCodeResponse{
			Success:          false,
			Error:            msg,
			Output:           combinedOutput,
			MaxTurnsExceeded: true,
		}, true
	}

	lowerOutput := strings.ToLower(combinedOutput)
	isRateLimit := strings.Contains(lowerOutput, "usage limit") ||
		strings.Contains(lowerOutput, "rate limit") ||
		strings.Contains(lowerOutput, "ai usage limit reached") ||
		strings.Contains(lowerOutput, "rate/usage limit reached") ||
		strings.Contains(lowerOutput, "claude ai usage limit reached") ||
		strings.Contains(lowerOutput, "you've reached your claude usage limit") ||
		strings.Contains(lowerOutput, "429") ||
		strings.Contains(lowerOutput, "too many requests") ||
		strings.Contains(lowerOutput, "quota exceeded") ||
		strings.Contains(lowerOutput, "rate limits are critical")

	if exitCode == 429 {
		isRateLimit = true
	}

	if isRateLimit {
		retryAfter := qp.extractRetryAfter(combinedOutput)
		qp.appendTaskLog(task.ID, agentTag, "stderr", fmt.Sprintf("🚫 Rate limit hit. Pausing for %d seconds", retryAfter))
		return &tasks.ClaudeCodeResponse{
			Success:     false,
			Error:       "RATE_LIMIT: API rate limit reached",
			RateLimited: true,
			RetryAfter:  retryAfter,
			Output:      combinedOutput,
		}, true
	}

	log.Printf("Claude Code exited with non-zero status for task %s (code %d)", task.ID, exitCode)
	return &tasks.ClaudeCodeResponse{
		Success: false,
		Error:   fmt.Sprintf("Claude Code execution failed (exit code %d): %s", exitCode, combinedOutput),
		Output:  combinedOutput,
	}, true
}

func exitCodeFromError(err error) (int, bool) {
	var exitCoder interface {
		ExitCode() int
	}
	if errors.As(err, &exitCoder) {
		return exitCoder.ExitCode(), true
	}

	var statusErr interface {
		ExitStatus() int
	}
	if errors.As(err, &statusErr) {
		return statusErr.ExitStatus(), true
	}

	return 0, false
}

func detectMaxTurnsExceeded(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "max turns") && strings.Contains(lower, "reached")
}

// broadcastUpdate sends updates to all connected WebSocket clients
func (qp *Processor) broadcastUpdate(updateType string, data interface{}) {
	// Send the typed update directly, not wrapped in another object
	// The WebSocket manager will wrap it properly
	select {
	case qp.broadcast <- map[string]interface{}{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}:
		log.Printf("Broadcast %s update for task", updateType)
	default:
		log.Printf("Warning: WebSocket broadcast channel full, dropping update")
	}
}
