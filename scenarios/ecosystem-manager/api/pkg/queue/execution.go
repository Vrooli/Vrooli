package queue

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// executeTask executes a single task
func (qp *Processor) executeTask(task tasks.TaskItem) {
	log.Printf("Executing task %s: %s", task.ID, task.Title)
	systemlog.Infof("Executing task %s (%s)", task.ID, task.Title)

	// Track execution timing
	executionStartTime := time.Now()
	cleanupReserved := true
	defer func() {
		if cleanupReserved {
			qp.unregisterExecution(task.ID)
		}
	}()

	// Get timeout setting for timing info
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute

	// Generate the full prompt for the task
	assembly, err := qp.assembler.AssemblePromptForTask(task)
	if err != nil {
		executionTime := time.Since(executionStartTime)
		log.Printf("Failed to assemble prompt for task %s: %v", task.ID, err)
		qp.handleTaskFailureWithTiming(&task, fmt.Sprintf("Prompt assembly failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil)
		return
	}
	prompt := assembly.Prompt

	// Store the assembled prompt in /tmp for debugging
	promptPath := filepath.Join("/tmp", fmt.Sprintf("ecosystem-prompt-%s.txt", task.ID))
	if err := os.WriteFile(promptPath, []byte(prompt), 0644); err != nil {
		log.Printf("Warning: Failed to save prompt to %s: %v", promptPath, err)
		// Don't fail the task, just log the warning
	} else {
		log.Printf("Saved assembled prompt to %s", promptPath)
	}

	// Update task phase before execution starts
	task.CurrentPhase = "prompt_assembled"
	qp.storage.SaveQueueItem(task, "in-progress")
	qp.broadcastUpdate("task_progress", task)

	// Log prompt size for debugging
	promptSizeKB := float64(len(prompt)) / 1024.0
	promptSizeMB := promptSizeKB / 1024.0
	log.Printf("Task %s: Prompt size: %d characters (%.2f KB / %.2f MB)", task.ID, len(prompt), promptSizeKB, promptSizeMB)

	// Call Claude Code resource
	result, cleanup, err := qp.callClaudeCode(prompt, task, executionStartTime, timeoutDuration)
	defer cleanup()
	executionTime := time.Since(executionStartTime)
	if execState, ok := qp.getExecution(task.ID); ok && execState.cmd != nil {
		cleanupReserved = false
	}

	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		qp.handleTaskFailureWithTiming(&task, fmt.Sprintf("Claude Code execution failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil)
		return
	}

	// Process the result
	// Debug: always log the execution result for debugging
	log.Printf("🔍 Task %s execution result: Success=%v, RateLimited=%v, Error=%q",
		task.ID, result.Success, result.RateLimited, result.Error)

	if result.Success {
		summary := fmt.Sprintf("Task %s completed successfully in %v (timeout %v)", task.ID, executionTime.Round(time.Second), timeoutDuration)
		log.Println(summary)
		systemlog.Info(summary)

		// Increment task counter for max_tasks tracking
		qp.tasksProcessedMu.Lock()
		qp.tasksProcessedCount++
		qp.tasksProcessedMu.Unlock()

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
		task.CurrentPhase = "completed"
		task.Status = "completed"
		task.CompletedAt = time.Now().Format(time.RFC3339)
		task.CompletionCount++
		task.LastCompletedAt = task.CompletedAt
		qp.broadcastUpdate("task_completed", task)
		qp.finalizeTaskStatus(&task, "completed")
	} else {
		// Check if this is a rate limit error
		if result.RateLimited {
			log.Printf("🚫 Task %s hit rate limit. Pausing queue for %d seconds", task.ID, result.RetryAfter)

			// Move task back to pending (don't mark as failed)
			task.CurrentPhase = "rate_limited"
			task.Status = "pending"
			// Store rate limit info in results, NOT in notes (preserve user notes)
			if task.Results == nil {
				task.Results = make(map[string]interface{})
			}
			task.Results["rate_limit_info"] = map[string]interface{}{
				"hit_at":      time.Now().Format(time.RFC3339),
				"retry_after": result.RetryAfter,
				"message":     fmt.Sprintf("Rate limited at %s. Will retry after %d seconds.", time.Now().Format(time.RFC3339), result.RetryAfter),
			}

			// Move back to pending queue for retry
			qp.finalizeTaskStatus(&task, "pending")

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
			systemlog.Warnf("Task %s failed: %s", task.ID, result.Error)
			var extras map[string]interface{}
			if result.MaxTurnsExceeded {
				extras = map[string]interface{}{"max_turns_exceeded": true}
			}
			qp.handleTaskFailureWithTiming(&task, result.Error, result.Output, executionStartTime, executionTime, timeoutDuration, extras)
		}
	}
}

// handleTaskFailure handles a failed task (legacy - for backward compatibility)
func (qp *Processor) handleTaskFailure(task *tasks.TaskItem, errorMsg string) {
	task.Results = map[string]interface{}{
		"success": false,
		"error":   errorMsg,
	}
	task.CurrentPhase = "failed"
	task.Status = "failed"
	qp.broadcastUpdate("task_failed", *task)

	qp.finalizeTaskStatus(task, "failed")
}

// handleTaskFailureWithTiming handles a failed task with execution timing information
func (qp *Processor) handleTaskFailureWithTiming(task *tasks.TaskItem, errorMsg string, output string, startTime time.Time, executionTime time.Duration, timeoutAllowed time.Duration, extras map[string]interface{}) {
	// Determine if this was a timeout failure
	isTimeout := strings.Contains(errorMsg, "timed out") || strings.Contains(errorMsg, "timeout")

	// Get the prompt for size calculation (if available)
	assembly, _ := qp.assembler.AssemblePromptForTask(*task)
	prompt := assembly.Prompt
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
	qp.broadcastUpdate("task_failed", *task)

	qp.finalizeTaskStatus(task, "failed")

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
func (qp *Processor) callClaudeCode(prompt string, task tasks.TaskItem, startTime time.Time, timeoutDuration time.Duration) (*tasks.ClaudeCodeResponse, func(), error) {
	cleanup := func() {}
	log.Printf("Executing Claude Code for task %s (prompt length: %d characters, timeout: %v)", task.ID, len(prompt), timeoutDuration)

	// Resolve Vrooli root to ensure resource CLI works no matter where binary runs
	vrooliRoot := qp.vrooliRoot
	if vrooliRoot == "" {
		vrooliRoot = detectVrooliRoot()
	}

	if err := qp.ensureValidClaudeConfig(); err != nil {
		msg := fmt.Sprintf("Claude configuration validation failed: %v", err)
		log.Printf(msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg}, cleanup, nil
	}

	timeoutSeconds := int(timeoutDuration.Seconds())
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	agentTag := fmt.Sprintf("ecosystem-%s", task.ID)
	if err := qp.ensureAgentInactive(agentTag); err != nil {
		msg := fmt.Sprintf("Unable to start Claude agent %s: %v", agentTag, err)
		log.Printf(msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg}, cleanup, nil
	}

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
		return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to start Claude Code stdout pipe: %v", err)}, cleanup, nil
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to obtain stderr pipe for task %s: %v", task.ID, err)
		return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to start Claude Code stderr pipe: %v", err)}, cleanup, nil
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start Claude Code for task %s: %v", task.ID, err)
		return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to start Claude Code: %v", err)}, cleanup, nil
	}

	qp.registerExecution(task.ID, agentTag, cmd, startTime)
	cleanup = func() {
		qp.stopClaudeAgent(agentTag, cmd.Process.Pid)
		qp.cleanupClaudeAgentRegistry()
		if qp.isProcessAlive(cmd.Process.Pid) {
			if err := KillProcessGroup(cmd.Process.Pid); err != nil {
				log.Printf("Warning: failed to kill process group for %s (pid %d): %v", task.ID, cmd.Process.Pid, err)
			}
		}
		qp.waitForAgentShutdown(agentTag, cmd.Process.Pid)
		qp.unregisterExecution(task.ID)
	}

	qp.initTaskLogBuffer(task.ID, agentTag, cmd.Process.Pid)
	completed := false
	defer func() {
		qp.finalizeTaskLogs(task.ID, completed)
	}()

	qp.appendTaskLog(task.ID, agentTag, "stdout", fmt.Sprintf("▶ Claude Code agent %s started (pid %d)", agentTag, cmd.Process.Pid))
	qp.broadcastUpdate("task_started", map[string]interface{}{
		"task_id":    task.ID,
		"agent_id":   agentTag,
		"process_id": cmd.Process.Pid,
		"start_time": startTime.Format(time.RFC3339),
	})

	task.CurrentPhase = "executing_claude"
	task.StartedAt = startTime.Format(time.RFC3339)
	if err := qp.storage.SaveQueueItem(task, "in-progress"); err != nil {
		log.Printf("Warning: failed to persist in-progress task %s: %v", task.ID, err)
	}
	qp.broadcastUpdate("task_executing", task)

	var stdoutBuilder, stderrBuilder, combinedBuilder strings.Builder
	var combinedMu sync.Mutex
	var lastActivity int64
	atomic.StoreInt64(&lastActivity, time.Now().UnixNano())

	idleLimit := time.Duration(math.Min(float64(timeoutDuration)/2, float64(5*time.Minute)))
	if idleLimit < 2*time.Minute {
		idleLimit = 2 * time.Minute
	}

	readsDone := make(chan struct{})
	stopWatch := make(chan struct{})
	var stopWatchOnce sync.Once
	defer stopWatchOnce.Do(func() {
		close(stopWatch)
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-readsDone:
				return
			case <-stopWatch:
				return
			case <-ticker.C:
				last := time.Unix(0, atomic.LoadInt64(&lastActivity))
				if time.Since(last) > idleLimit {
					message := fmt.Sprintf("⚠️  No Claude output for %v. Cancelling execution.", idleLimit)
					qp.appendTaskLog(task.ID, agentTag, "stderr", message)
					cancel()
					return
				}
			}
		}
	}()

	streamPipe := func(stream string, reader io.ReadCloser) {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			qp.appendTaskLog(task.ID, agentTag, stream, line)
			atomic.StoreInt64(&lastActivity, time.Now().UnixNano())
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
	stopWatchOnce.Do(func() {
		close(stopWatch)
	})

	ctxErr := ctx.Err()

	// Timeout takes precedence regardless of wait error
	if ctxErr == context.DeadlineExceeded {
		actualRuntime := time.Since(startTime).Round(time.Second)
		msg := fmt.Sprintf("⏰ TIMEOUT: Task execution exceeded %v limit (ran for %v)\n\nThe task was automatically terminated because it exceeded the configured timeout.\nConsider:\n- Increasing timeout in Settings if this is a complex task\n- Breaking the task into smaller parts\n- Checking for blocking steps in the prompt", timeoutDuration, actualRuntime)
		qp.appendTaskLog(task.ID, agentTag, "stderr", msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg, Output: combinedOutput}, cleanup, nil
	}

	if waitErr != nil {
		if ctxErr == context.Canceled {
			if strings.Contains(combinedOutput, "## Task Completion Summary") || strings.Contains(combinedOutput, "## Summary") {
				qp.appendTaskLog(task.ID, agentTag, "stderr", "INFO: Idle watchdog cancelled context, but Claude returned a summary. Treating as success.")
				waitErr = nil
				// Let execution continue as a success path below
			}
		}

		if waitErr != nil {
			if detectMaxTurnsExceeded(combinedOutput) {
				maxTurnsMsg := fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", currentSettings.MaxTurns)
				qp.appendTaskLog(task.ID, agentTag, "stderr", maxTurnsMsg)
				return &tasks.ClaudeCodeResponse{
					Success:          false,
					Error:            maxTurnsMsg,
					Output:           combinedOutput,
					MaxTurnsExceeded: true,
				}, cleanup, nil
			}

			if response, handled := qp.handleNonZeroExit(waitErr, combinedOutput, task, agentTag, currentSettings.MaxTurns); handled {
				return response, cleanup, nil
			}

			log.Printf("Claude Code process errored for task %s: %v", task.ID, waitErr)
			return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to execute Claude Code: %v", waitErr), Output: combinedOutput}, cleanup, nil
		}
	}

	if ctxErr == context.Canceled {
		qp.appendTaskLog(task.ID, agentTag, "stderr", "INFO: Claude process completed after idle watchdog cancellation signal")
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
		}, cleanup, nil
	}

	if detectMaxTurnsExceeded(combinedOutput) {
		maxTurnsMsg := fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", currentSettings.MaxTurns)
		qp.appendTaskLog(task.ID, agentTag, "stderr", maxTurnsMsg)
		return &tasks.ClaudeCodeResponse{
			Success:          false,
			Error:            maxTurnsMsg,
			Output:           combinedOutput,
			MaxTurnsExceeded: true,
		}, cleanup, nil
	}

	if strings.Contains(strings.ToLower(combinedOutput), "error:") {
		msg := "Claude execution reported an error – review output for details."
		qp.appendTaskLog(task.ID, agentTag, "stderr", msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg, Output: combinedOutput}, cleanup, nil
	}

	log.Printf("Claude Code completed successfully for task %s (output length: %d characters)", task.ID, len(combinedOutput))
	qp.appendTaskLog(task.ID, agentTag, "stdout", "✅ Claude Code execution finished")

	task.CurrentPhase = "claude_completed"
	if err := qp.storage.SaveQueueItem(task, "in-progress"); err != nil {
		log.Printf("Warning: failed to persist claude completion for task %s: %v", task.ID, err)
	}
	qp.broadcastUpdate("claude_execution_complete", task)

	completed = true
	return &tasks.ClaudeCodeResponse{
		Success: true,
		Message: "Task completed successfully",
		Output:  combinedOutput,
	}, cleanup, nil
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
		return 1800 // 30 minutes (reduced from 1 hour to be less disruptive)
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
		return 1800 // 30 minutes for critical rate limits
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
	// More specific rate limit detection to avoid false positives
	// Avoid matching benign phrases like "rate limiting headers" or "rate limit middleware"
	isRateLimit := strings.Contains(lowerOutput, "ai usage limit reached") ||
		strings.Contains(lowerOutput, "rate/usage limit reached") ||
		strings.Contains(lowerOutput, "claude ai usage limit reached") ||
		strings.Contains(lowerOutput, "you've reached your claude usage limit") ||
		strings.Contains(lowerOutput, "usage limit reached") ||
		strings.Contains(lowerOutput, "rate limit reached") ||
		strings.Contains(lowerOutput, "rate limit exceeded") ||
		strings.Contains(lowerOutput, "too many requests") ||
		strings.Contains(lowerOutput, "quota exceeded") ||
		strings.Contains(lowerOutput, "rate limits are critical") ||
		strings.Contains(lowerOutput, "error 429")

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

func (qp *Processor) ensureValidClaudeConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	configPaths := []string{
		filepath.Join(homeDir, ".claude", ".claude.json"),
		filepath.Join(homeDir, ".claude.json"),
	}

	validFound := false
	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read claude config %s: %w", path, err)
		}

		if isValidClaudeConfig(data) {
			validFound = true
			continue
		}

		if err := qp.resetClaudeConfig(path, data); err != nil {
			return err
		}
		log.Printf("Detected invalid Claude config at %s; reset to defaults for automation execution.", path)
		validFound = true
	}

	if !validFound {
		path := configPaths[0]
		if err := qp.resetClaudeConfig(path, nil); err != nil {
			return err
		}
		log.Printf("Claude config not found; created default config at %s", path)
	}

	return nil
}

func (qp *Processor) resetClaudeConfig(path string, original []byte) error {
	trimmed := bytes.TrimSpace(original)
	if len(trimmed) > 0 {
		backupPath := fmt.Sprintf("%s.invalid.%d", path, time.Now().Unix())
		if err := os.WriteFile(backupPath, original, 0o600); err != nil {
			log.Printf("Warning: failed to back up invalid Claude config %s: %v", path, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure claude config directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte("{\n}\n"), 0o600); err != nil {
		return fmt.Errorf("write default claude config to %s: %w", path, err)
	}

	return nil
}

func isValidClaudeConfig(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return false
	}

	return true
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
