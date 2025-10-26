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

	"github.com/ecosystem-manager/api/pkg/internal/paths"
	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// completeTaskCleanup handles unregistering execution and cleaning up agent after finalization
// This should be called ONLY after finalizeTaskStatus succeeds.
//
// IMPORTANT: When finalization fails, call cleanup() directly instead to terminate the agent,
// but leave cleanupReserved=true so the defer block handles unregistration.
// This ensures:
// 1. Agent is terminated even if we can't move the file (prevents orphaned processes)
// 2. Execution registry is cleaned up by defer (ensures reconciliation sees consistent state)
func (qp *Processor) completeTaskCleanup(taskID string, cleanupFunc func(), cleanupReserved *bool) {
	// Finalization succeeded, safe to unregister and disable defer cleanup
	*cleanupReserved = false
	qp.unregisterExecution(taskID)

	// Now safe to cleanup agent - task file has been moved out of in-progress
	cleanupFunc()
}

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
		wrappedErr := fmt.Errorf("assemble prompt for task %s: %w", task.ID, err)
		log.Printf("Failed to assemble prompt for task %s: %v", task.ID, wrappedErr)
		if finalizeErr := qp.handleTaskFailureWithTiming(&task, fmt.Sprintf("Prompt assembly failed: %v", wrappedErr), "", executionStartTime, executionTime, timeoutDuration, nil); finalizeErr != nil {
			log.Printf("CRITICAL: Failed to finalize task %s after prompt assembly failure: %v", task.ID, finalizeErr)
			systemlog.Errorf("Failed to finalize task %s after prompt assembly failure: %v - task will remain in executions", task.ID, finalizeErr)
			// Keep cleanupReserved=true so defer will unregister execution
			// Note: No agent process exists yet at this point, no cleanup needed
			return
		}
		// Finalization succeeded, safe to unregister and disable defer cleanup
		cleanupReserved = false
		qp.unregisterExecution(task.ID)
		return
	}
	prompt := assembly.Prompt

	// Store the assembled prompt in temp directory for debugging
	promptPath := filepath.Join(os.TempDir(), fmt.Sprintf("ecosystem-prompt-%s.txt", task.ID))
	if err := os.WriteFile(promptPath, []byte(prompt), PromptFilePermissions); err != nil {
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
	promptSizeKB := float64(len(prompt)) / BytesPerKilobyte
	promptSizeMB := promptSizeKB / KilobytesPerMegabyte
	log.Printf("Task %s: Prompt size: %d characters (%.2f KB / %.2f MB)", task.ID, len(prompt), promptSizeKB, promptSizeMB)

	// Call Claude Code resource
	result, cleanup, err := qp.callClaudeCode(prompt, task, executionStartTime, timeoutDuration)
	executionTime := time.Since(executionStartTime)

	// Handle Claude Code execution errors
	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		if finalizeErr := qp.handleTaskFailureWithTiming(&task, fmt.Sprintf("Claude Code execution failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil); finalizeErr != nil {
			wrappedErr := NewFinalizationError(task.ID, "execution_failure", finalizeErr)
			log.Printf("CRITICAL: Failed to finalize task %s after execution failure: %v", task.ID, finalizeErr)
			systemlog.Errorf("Failed to finalize task %s after execution failure: %v", task.ID, wrappedErr)
			// CRITICAL: Must cleanup agent even when finalization fails to prevent orphaned processes
			// Don't unregister yet - let defer handle that after cleanup
			cleanup()
			// Keep cleanupReserved=true so defer will unregister execution
			return
		}
		qp.completeTaskCleanup(task.ID, cleanup, &cleanupReserved)
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

		// Update task with results including timing and prompt size
		completedAt := timeutil.NowRFC3339()
		task.Results = map[string]interface{}{
			"success":         true,
			"message":         result.Message,
			"output":          result.Output,
			"execution_time":  executionTime.Round(time.Second).String(),
			"timeout_allowed": timeoutDuration.String(),
			"started_at":      executionStartTime.Format(time.RFC3339),
			"completed_at":    completedAt,
			"prompt_size":     fmt.Sprintf("%d chars (%.2f KB)", len(prompt), promptSizeKB),
		}
		task.CurrentPhase = "completed"
		task.Status = "completed"
		task.CompletedAt = completedAt
		task.CompletionCount++
		task.LastCompletedAt = task.CompletedAt

		// CRITICAL: Finalize status to disk BEFORE broadcasting to prevent UI/disk state divergence
		// This also prevents reconciliation race condition where task appears orphaned
		if finalizeErr := qp.finalizeTaskStatus(&task, "completed"); finalizeErr != nil {
			wrappedErr := NewFinalizationError(task.ID, "success", finalizeErr)
			log.Printf("CRITICAL: Failed to finalize completed task %s: %v", task.ID, wrappedErr)
			systemlog.Errorf("Failed to finalize completed task %s: %v - task will remain in executions for reconciler", task.ID, wrappedErr)
			// CRITICAL: Call cleanup for safety - agent should have exited naturally, but
			// cleanup() safely handles already-terminated processes (checks isProcessAlive first)
			cleanup()
			// Keep cleanupReserved=true so defer will unregister execution
			// Task file is still in-progress, so reconciler will move it back to pending
			return
		}

		// Only broadcast after successful finalization
		qp.broadcastUpdate("task_completed", task)

		// Finalization succeeded, safe to unregister and cleanup
		// Agent should have exited naturally, but cleanup() handles already-terminated processes safely
		qp.completeTaskCleanup(task.ID, cleanup, &cleanupReserved)
	} else {
		// Task failed or was rate limited
		// CRITICAL: Don't call cleanup() yet - wait until after finalization succeeds
		// to avoid race condition with reconciliation

		// Check if this is a rate limit error
		if result.RateLimited {
			log.Printf("🚫 Task %s hit rate limit. Pausing queue for %d seconds", task.ID, result.RetryAfter)

			// Move task back to pending (don't mark as failed)
			task.CurrentPhase = "rate_limited"
			task.Status = "pending"
			// Store rate limit info in results, NOT in notes (preserve user notes)
			hitAt := timeutil.NowRFC3339()
			if task.Results == nil {
				task.Results = make(map[string]interface{})
			}
			task.Results["rate_limit_info"] = map[string]interface{}{
				"hit_at":      hitAt,
				"retry_after": result.RetryAfter,
				"message":     fmt.Sprintf("Rate limited at %s. Will retry after %d seconds.", hitAt, result.RetryAfter),
			}

			// Move back to pending queue for retry
			if finalizeErr := qp.finalizeTaskStatus(&task, "pending"); finalizeErr != nil {
				wrappedErr := NewFinalizationError(task.ID, "rate_limit", finalizeErr)
				log.Printf("CRITICAL: Failed to finalize rate-limited task %s: %v", task.ID, wrappedErr)
				systemlog.Errorf("Failed to move rate-limited task %s to pending: %v - task will remain in executions", task.ID, wrappedErr)
				// CRITICAL: Must cleanup agent even when finalization fails to prevent orphaned processes
				cleanup()
				// Keep cleanupReserved=true so defer will unregister execution
				return
			}

			// Finalization succeeded, cleanup agent and unregister
			qp.completeTaskCleanup(task.ID, cleanup, &cleanupReserved)

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
			if finalizeErr := qp.handleTaskFailureWithTiming(&task, result.Error, result.Output, executionStartTime, executionTime, timeoutDuration, extras); finalizeErr != nil {
				wrappedErr := NewFinalizationError(task.ID, "failure", finalizeErr)
				log.Printf("CRITICAL: Failed to finalize failed task %s: %v", task.ID, wrappedErr)
				systemlog.Errorf("Failed to finalize failed task %s: %v - task will remain in executions", task.ID, wrappedErr)
				// CRITICAL: Must cleanup agent even when finalization fails to prevent orphaned processes
				// This is especially important for MAX_TURNS cases where agent may still be running
				cleanup()
				// Keep cleanupReserved=true so defer will unregister execution
				return
			}
			// Finalization succeeded (handleTaskFailureWithTiming calls finalizeTaskStatus)
			qp.completeTaskCleanup(task.ID, cleanup, &cleanupReserved)
		}
	}
}

// handleTaskFailureWithTiming handles a failed task with execution timing information
func (qp *Processor) handleTaskFailureWithTiming(task *tasks.TaskItem, errorMsg string, output string, startTime time.Time, executionTime time.Duration, timeoutAllowed time.Duration, extras map[string]interface{}) error {
	// Determine if this was a timeout failure
	isTimeout := strings.Contains(errorMsg, "timed out") || strings.Contains(errorMsg, "timeout")

	// Get the prompt for size calculation (if available)
	assembly, _ := qp.assembler.AssemblePromptForTask(*task)
	prompt := assembly.Prompt
	promptSizeKB := float64(len(prompt)) / BytesPerKilobyte

	failedAt := timeutil.NowRFC3339()
	results := map[string]interface{}{
		"success":         false,
		"error":           errorMsg,
		"output":          output,
		"execution_time":  executionTime.Round(time.Second).String(),
		"timeout_allowed": timeoutAllowed.String(),
		"started_at":      startTime.Format(time.RFC3339),
		"failed_at":       failedAt,
		"timeout_failure": isTimeout,
		"prompt_size":     fmt.Sprintf("%d chars (%.2f KB)", len(prompt), promptSizeKB),
	}
	for k, v := range extras {
		results[k] = v
	}

	task.Results = results

	task.CurrentPhase = "failed"
	task.Status = "failed"
	task.CompletedAt = timeutil.NowRFC3339()

	// Log detailed timing information
	if isTimeout {
		log.Printf("Task %s TIMED OUT after %v (limit was %v)",
			task.ID, executionTime.Round(time.Second), timeoutAllowed)
	} else {
		log.Printf("Task %s FAILED after %v (limit was %v)",
			task.ID, executionTime.Round(time.Second), timeoutAllowed)
	}

	// CRITICAL: Finalize to disk BEFORE broadcasting to prevent UI/disk state divergence
	if err := qp.finalizeTaskStatus(task, "failed"); err != nil {
		log.Printf("CRITICAL: Failed to finalize failed task %s: %v", task.ID, err)
		systemlog.Errorf("Failed to finalize failed task %s: %v - task will remain in executions", task.ID, err)
		return fmt.Errorf("failed to finalize task status: %w", err)
	}

	// Only broadcast after successful finalization
	qp.broadcastUpdate("task_failed", *task)

	return nil
}

// callClaudeCode executes Claude Code while streaming logs for real-time monitoring
func (qp *Processor) callClaudeCode(prompt string, task tasks.TaskItem, startTime time.Time, timeoutDuration time.Duration) (*tasks.ClaudeCodeResponse, func(), error) {
	cleanup := func() {}
	log.Printf("Executing Claude Code for task %s (prompt length: %d characters, timeout: %v)", task.ID, len(prompt), timeoutDuration)

	// Resolve Vrooli root to ensure resource CLI works no matter where binary runs
	vrooliRoot := qp.vrooliRoot
	if vrooliRoot == "" {
		vrooliRoot = paths.DetectVrooliRoot()
	}

	if err := qp.ensureValidClaudeConfig(); err != nil {
		msg := fmt.Sprintf("Claude configuration validation failed: %v", err)
		log.Printf(msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg}, cleanup, nil
	}

	timeoutSeconds := int(timeoutDuration.Seconds())
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	agentTag := makeAgentTag(task.ID)
	if err := qp.ensureAgentInactive(agentTag); err != nil {
		msg := fmt.Sprintf("Unable to start Claude agent %s: %v", agentTag, err)
		log.Printf(msg)
		return &tasks.ClaudeCodeResponse{Success: false, Error: msg}, cleanup, nil
	}

	cmd := exec.CommandContext(ctx, ClaudeCodeResourceCommand, "run", "--tag", agentTag, "-")
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

	// Cleanup function handles agent termination for error/timeout cases
	// Uses killProcessGracefully as the canonical termination mechanism for consistency
	cleanup = func() {
		pid := cmd.Process.Pid

		// Check if process is still alive before attempting cleanup
		if !qp.isProcessAlive(pid) {
			// Process already exited naturally - just clean up registry
			qp.cleanupClaudeAgentRegistry()
			return
		}

		// Process is still running - need to terminate it
		log.Printf("Terminating active Claude agent %s (pid %d) for task %s", agentTag, pid, task.ID)

		// Try graceful agent stop via CLI first
		qp.stopClaudeAgent(agentTag, pid)
		qp.cleanupClaudeAgentRegistry()

		// Use canonical process termination logic for consistency
		// This handles SIGTERM -> wait -> SIGKILL -> verify sequence
		if qp.isProcessAlive(pid) {
			if err := qp.killProcessGracefully(pid); err != nil {
				log.Printf("Warning: failed to gracefully terminate process %d for task %s: %v", pid, task.ID, err)
				// Final fallback: force kill process group
				if err := KillProcessGroup(pid); err != nil {
					log.Printf("Warning: failed to kill process group for %s (pid %d): %v", task.ID, pid, err)
				}
			}
		}

		qp.waitForAgentShutdown(agentTag, pid)
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
		"start_time": timeutil.NowRFC3339(),
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

	idleLimit := time.Duration(math.Min(float64(timeoutDuration)*MaxIdleTimeoutFactor, float64(DefaultIdleTimeoutCap)))
	if idleLimit < MinIdleTimeout {
		idleLimit = MinIdleTimeout
	}

	readsDone := make(chan struct{})
	stopWatch := make(chan struct{})
	var stopWatchOnce sync.Once
	defer stopWatchOnce.Do(func() {
		close(stopWatch)
	})

	go func() {
		ticker := time.NewTicker(IdleCheckInterval)
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
		buf := make([]byte, ScannerBufferSize)
		scanner.Buffer(buf, ScannerMaxTokenSize)
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

	// Wait for pipes to finish reading before calling Wait()
	// This prevents "file already closed" races
	waitErrChan := make(chan error, 1)
	go func() {
		<-readsDone
		waitErrChan <- cmd.Wait()
		close(waitErrChan)
	}()

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

			if response, handled := qp.handleNonZeroExit(waitErr, combinedOutput, task, agentTag, currentSettings.MaxTurns, time.Since(startTime)); handled {
				return response, cleanup, nil
			}

			log.Printf("Claude Code process errored for task %s: %v", task.ID, waitErr)
			return &tasks.ClaudeCodeResponse{Success: false, Error: fmt.Sprintf("Failed to execute Claude Code: %v", waitErr), Output: combinedOutput}, cleanup, nil
		}
	}

	if ctxErr == context.Canceled {
		qp.appendTaskLog(task.ID, agentTag, "stderr", "INFO: Claude process completed after idle watchdog cancellation signal")
	}

	elapsed := time.Since(startTime)
	if elapsed <= RateLimitDetectionWindow && isRateLimitError(combinedOutput) {
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
	defaultRetry := DefaultRateLimitRetry
	lowerOutput := strings.ToLower(output)

	// Common time duration patterns and their backoff values
	// Using 30 minutes default for 1 hour pattern to be less disruptive
	hourPatterns := map[string]int{
		"5 hour":        5 * 3600,
		"5-hour":        5 * 3600,
		"every 5 hours": 5 * 3600,
		"4 hour":        4 * 3600,
		"4-hour":        4 * 3600,
		"1 hour":        DefaultRateLimitRetry,
		"1-hour":        DefaultRateLimitRetry,
	}

	// Check for common time patterns
	for pattern, duration := range hourPatterns {
		if strings.Contains(lowerOutput, pattern) {
			return duration
		}
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
					if seconds > MaxRateLimitPause {
						return MaxRateLimitPause
					}
					// Minimum 5 minutes
					if seconds < MinRateLimitPause {
						return MinRateLimitPause
					}
					return seconds
				}
			}
		}
	}

	// If we see "critical" rate limits, use a longer default
	if strings.Contains(lowerOutput, "critical") {
		return CriticalRateLimitPause
	}

	return defaultRetry
}

func (qp *Processor) handleNonZeroExit(waitErr error, combinedOutput string, task tasks.TaskItem, agentTag string, maxTurns int, elapsed time.Duration) (*tasks.ClaudeCodeResponse, bool) {
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

	// More specific rate limit detection to avoid false positives
	// Avoid matching benign phrases like "rate limiting headers" or "rate limit middleware"
	rateLimitDetected := isRateLimitError(combinedOutput) || exitCode == HTTPStatusTooManyRequests

	if rateLimitDetected && elapsed <= RateLimitDetectionWindow {
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

// isRateLimitError detects if the output contains rate limit error patterns
func isRateLimitError(output string) bool {
	patterns := []string{
		"ai usage limit reached",
		"rate/usage limit reached",
		"claude ai usage limit reached",
		"you've reached your claude usage limit",
		"usage limit reached",
		"rate limit reached",
		"rate limit exceeded",
		"too many requests",
		"quota exceeded",
		"rate limits are critical",
		"error 429",
		"usage limit",
		"rate limit",
		"429",
	}
	lower := strings.ToLower(output)
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
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
