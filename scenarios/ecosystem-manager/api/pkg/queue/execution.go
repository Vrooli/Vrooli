package queue

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/agentmanager"
	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/ratelimit"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// cleanupTaskWithVerifiedAgentRemoval is the unified cleanup handler used after task finalization.
// With agent-manager integration, cleanup is simplified: the cleanup function calls agentSvc.StopRun()
// which handles all termination logic. We just call cleanup and unregister.
func (qp *Processor) cleanupTaskWithVerifiedAgentRemoval(taskID string, cleanupFunc func(), ctx string, keepTracking bool) {
	if ctx != "" {
		log.Printf("Task %s cleanup (%s)", taskID, ctx)
	}

	// Call the cleanup function (calls agentSvc.StopRun via closure)
	cleanupFunc()

	// Unregister from execution tracking
	qp.unregisterExecution(taskID)
}

// completeTaskCleanupExplicit handles cleanup after successful finalization
func (qp *Processor) completeTaskCleanupExplicit(taskID string, cleanupFunc func()) {
	qp.cleanupTaskWithVerifiedAgentRemoval(taskID, cleanupFunc, "task finalization success", true)
}

// cleanupAgentAfterFinalizationFailure handles cleanup when finalization fails
func (qp *Processor) cleanupAgentAfterFinalizationFailure(taskID string, cleanupFunc func(), context string) {
	qp.cleanupTaskWithVerifiedAgentRemoval(taskID, cleanupFunc, context, true)
}

// appendManualSteeringSection attaches a manual steering mode when provided, otherwise falls back to Progress.
func (qp *Processor) appendManualSteeringSection(prompt string, steerMode autosteer.SteerMode) string {
	if qp.assembler == nil {
		return prompt
	}

	mode := steerMode
	if !mode.IsValid() {
		mode = autosteer.ModeProgress
	}

	// Prefer the orchestrator's cached prompt enhancer when available.
	if qp.autoSteerIntegration != nil {
		if orchestrator := qp.autoSteerIntegration.ExecutionOrchestrator(); orchestrator != nil {
			if section := strings.TrimSpace(orchestrator.GenerateModeSection(mode)); section != "" {
				return autosteer.InjectSteeringSection(prompt, section)
			}
		}
	}

	phasesDir := filepath.Join(qp.assembler.PromptsDir, "phases")
	section := strings.TrimSpace(autosteer.NewPromptEnhancer(phasesDir).GenerateModeSection(mode))
	if section == "" {
		return autosteer.InjectSteeringSection(prompt, "")
	}
	return autosteer.InjectSteeringSection(prompt, section)
}

// enrichSteeringMetadata captures the steering context used for an execution so we can analyze prompts later.
func (qp *Processor) enrichSteeringMetadata(task *tasks.TaskItem, history *ExecutionHistory) {
	if task == nil || history == nil {
		return
	}

	history.SteeringSource = "none"

	// Capture Auto Steer state when active.
	if qp.autoSteerIntegration != nil && strings.TrimSpace(task.AutoSteerProfileID) != "" {
		orchestrator := qp.autoSteerIntegration.ExecutionOrchestrator()
		if orchestrator != nil {
			if state, err := orchestrator.GetExecutionState(task.ID); err == nil && state != nil {
				history.AutoSteerProfileID = state.ProfileID
				history.AutoSteerIteration = state.AutoSteerIteration
				history.SteerPhaseIndex = state.CurrentPhaseIndex + 1
				history.SteerPhaseIteration = state.CurrentPhaseIteration + 1
				history.SteeringSource = "auto_steer"
			} else if err != nil {
				log.Printf("Warning: Failed to capture Auto Steer state for task %s: %v", task.ID, err)
			}

			if mode, err := orchestrator.GetCurrentMode(task.ID); err == nil {
				if mode != "" {
					history.SteerMode = string(mode)
					if history.SteeringSource == "none" {
						history.SteeringSource = "auto_steer"
					}
				}
			} else {
				log.Printf("Warning: Failed to capture Auto Steer mode for task %s: %v", task.ID, err)
			}
		}
	}

	// For improver tasks without Auto Steer we still inject steering guidance.
	if history.SteeringSource == "none" && task.Type == "scenario" && task.Operation == "improver" {
		mode := autosteer.SteerMode(strings.ToLower(strings.TrimSpace(task.SteerMode)))
		if mode.IsValid() {
			history.SteerMode = string(mode)
			history.SteeringSource = "manual_mode"
		} else {
			history.SteerMode = string(autosteer.ModeProgress)
			history.SteeringSource = "default_progress"
		}
		history.SteerPhaseIndex = 1
		history.SteerPhaseIteration = 1
	}
}

// executeTask executes a single task
func (qp *Processor) executeTask(task tasks.TaskItem) {
	log.Printf("Executing task %s: %s", task.ID, task.Title)
	systemlog.Infof("Executing task %s (%s)", task.ID, task.Title)
	defer qp.Wake()

	// Track execution timing and create execution ID
	executionStartTime := time.Now()
	executionID := createExecutionID()

	// Initialize execution history metadata
	history := ExecutionHistory{
		TaskID:        task.ID,
		TaskTitle:     task.Title,
		TaskType:      task.Type,
		TaskOperation: task.Operation,
		ExecutionID:   executionID,
		StartTime:     executionStartTime,
	}

	// Get timeout setting for timing info
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute
	history.TimeoutAllowed = timeoutDuration.String()

	// Initialize Auto Steer if needed (first time executing with Auto Steer profile)
	autoSteerInitFailed := false
	if qp.autoSteerIntegration != nil && task.AutoSteerProfileID != "" {
		scenarioName := getScenarioNameFromTask(&task)
		if err := qp.autoSteerIntegration.InitializeAutoSteer(&task, scenarioName); err != nil {
			log.Printf("Failed to initialize Auto Steer for task %s: %v", task.ID, err)
			systemlog.Errorf("Auto Steer initialization failed for task %s: %v", task.ID, err)
			autoSteerInitFailed = true
			// Continue without Auto Steer rather than failing the task
		}
	}

	// Surface latest execution output path for prompt templating (if available)
	if latest := qp.LatestExecutionOutputPath(task.ID); latest != "" {
		task.LatestOutputPath = latest
	}

	// Generate the full prompt for the task
	assembly, err := qp.assembler.AssemblePromptForTask(task)
	if err != nil {
		executionTime := time.Since(executionStartTime)
		log.Printf("Failed to assemble prompt for task %s: %v", task.ID, err)
		if finalizeErr := qp.handleTaskFailureWithTiming(&task, fmt.Sprintf("Prompt assembly failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil); finalizeErr != nil {
			log.Printf("CRITICAL: Failed to finalize task %s after prompt assembly failure: %v", task.ID, finalizeErr)
			systemlog.Errorf("Failed to finalize task %s after prompt assembly failure: %v - task will remain in executions", task.ID, finalizeErr)
			// No agent process exists yet at this point, no cleanup needed
			qp.unregisterExecution(task.ID)
			return
		}
		// Finalization succeeded - unregister execution
		qp.unregisterExecution(task.ID)
		return
	}
	prompt := assembly.Prompt

	// Apply default Progress steering when Auto Steer is not configured
	if strings.TrimSpace(task.AutoSteerProfileID) == "" && task.Type == "scenario" && task.Operation == "improver" {
		mode := autosteer.SteerMode(strings.ToLower(strings.TrimSpace(task.SteerMode)))
		prompt = qp.appendManualSteeringSection(prompt, mode)
	}

	// Enhance prompt with Auto Steer context if active
	if qp.autoSteerIntegration != nil && task.AutoSteerProfileID != "" {
		enhancedPrompt, err := qp.autoSteerIntegration.EnhancePrompt(&task, prompt)
		if err != nil {
			log.Printf("Warning: Failed to enhance prompt with Auto Steer for task %s: %v", task.ID, err)
			// Continue with base prompt
		} else if enhancedPrompt != prompt {
			prompt = enhancedPrompt
			log.Printf("Prompt enhanced with Auto Steer for task %s", task.ID)
		}
	}

	history.PromptSize = len(prompt)
	qp.enrichSteeringMetadata(&task, &history)

	// Store the assembled prompt in execution history
	promptRelPath, err := qp.savePromptToHistory(task.ID, executionID, prompt)
	if err != nil {
		log.Printf("Warning: Failed to save prompt to history for %s: %v", task.ID, err)
		// Don't fail the task, just log the warning
	} else {
		history.PromptPath = promptRelPath
		log.Printf("Saved assembled prompt to execution history: %s", promptRelPath)
	}

	// Update task phase before execution starts (use SkipCleanup for performance)
	task.CurrentPhase = "prompt_assembled"
	qp.storage.SaveQueueItemSkipCleanup(task, "in-progress")
	qp.broadcastUpdate("task_progress", task)

	// Log prompt size for debugging
	promptSizeKB := float64(len(prompt)) / BytesPerKilobyte
	promptSizeMB := promptSizeKB / KilobytesPerMegabyte
	log.Printf("Task %s: Prompt size: %d characters (%.2f KB / %.2f MB)", task.ID, len(prompt), promptSizeKB, promptSizeMB)

	// Call Claude Code resource
	result, cleanup, err := qp.callClaudeCode(prompt, task, executionID, executionStartTime, timeoutDuration, &history)
	executionTime := time.Since(executionStartTime)

	// Handle Claude Code execution errors
	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		if finalizeErr := qp.handleTaskFailureWithTiming(&task, fmt.Sprintf("Claude Code execution failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil); finalizeErr != nil {
			log.Printf("CRITICAL: Failed to finalize task %s after execution failure: %v", task.ID, finalizeErr)
			systemlog.Errorf("Failed to finalize task %s after execution failure: %v", task.ID, finalizeErr)
			// CRITICAL: Use verified cleanup to prevent agent remaining in registry
			qp.cleanupAgentAfterFinalizationFailure(task.ID, cleanup, "execution failure")
			return
		}
		// Finalization succeeded - cleanup agent and unregister
		qp.completeTaskCleanupExplicit(task.ID, cleanup)
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

		// Update task with structured results
		completedAt := timeutil.NowRFC3339()
		taskResults := tasks.NewSuccessResults(
			result.Message,
			result.Output,
			executionTime.Round(time.Second).String(),
			timeoutDuration.String(),
			executionStartTime.Format(time.RFC3339),
			completedAt,
			fmt.Sprintf("%d chars (%.2f KB)", len(prompt), promptSizeKB),
		)
		task.Results = taskResults.ToMap()
		task.CurrentPhase = "completed"
		task.Status = "completed"
		task.CompletedAt = completedAt
		task.CompletionCount++
		task.LastCompletedAt = task.CompletedAt

		// Auto Steer: Evaluate iteration and determine if task should continue
		if qp.autoSteerIntegration != nil && task.AutoSteerProfileID != "" {
			// If initialization failed earlier, disable auto-requeue to prevent infinite loop
			if autoSteerInitFailed {
				log.Printf("Auto Steer: Task %s initialization failed - disabling auto-requeue", task.ID)
				systemlog.Warnf("Auto Steer: Task %s disabled due to initialization failure", task.ID)
				task.ProcessorAutoRequeue = false
				task.Status = tasks.StatusCompletedFinalized
			} else {
				scenarioName := getScenarioNameFromTask(&task)
				shouldContinue, err := qp.autoSteerIntegration.ShouldContinueTask(&task, scenarioName)
				if err != nil {
					log.Printf("Warning: Auto Steer evaluation failed for task %s: %v", task.ID, err)
					systemlog.Warnf("Auto Steer evaluation error for %s: %v", task.ID, err)
					// Disable auto-requeue to prevent infinite loop when evaluation fails
					task.ProcessorAutoRequeue = false
					task.Status = tasks.StatusCompletedFinalized
				} else {
					if !shouldContinue {
						log.Printf("Auto Steer: Task %s completed all phases - moving to finalized", task.ID)
						systemlog.Infof("Auto Steer: Task %s fully complete - finalized", task.ID)
						// Use completed-finalized to prevent profile reset on manual moves
						task.ProcessorAutoRequeue = false
						task.Status = tasks.StatusCompletedFinalized
					} else {
						if task.ProcessorAutoRequeue {
							log.Printf("Auto Steer: Task %s will continue - requeuing for next iteration", task.ID)
							task.Status = "pending"
						} else {
							log.Printf("Auto Steer: Task %s would continue, but auto-enqueue is disabled; leaving completed", task.ID)
							task.Status = "completed"
						}
					}
				}
			}
		}

		if task.Status == "completed" && task.ProcessorAutoRequeue {
			cooldownSeconds := settings.GetSettings().CooldownSeconds
			if cooldownSeconds > 0 {
				task.CooldownUntil = time.Now().Add(time.Duration(cooldownSeconds) * time.Second).Format(time.RFC3339)
			} else {
				task.CooldownUntil = ""
			}
		} else {
			task.CooldownUntil = ""
		}

		// Update execution history
		history.EndTime = time.Now()
		history.Duration = executionTime.Round(time.Second).String()
		history.Success = true
		history.ExitReason = "completed"

		// CRITICAL: Finalize status to disk BEFORE broadcasting to prevent UI/disk state divergence
		// This also prevents reconciliation race condition where task appears orphaned
		// Use the task's actual status (which may be "pending" if Auto Steer wants to continue)
		finalizeStatus := task.Status
		if finalizeErr := qp.finalizeTaskStatus(&task, finalizeStatus); finalizeErr != nil {
			log.Printf("CRITICAL: Failed to finalize completed task %s: %v", task.ID, finalizeErr)
			systemlog.Errorf("Failed to finalize completed task %s: %v - task will remain in executions for reconciler", task.ID, finalizeErr)
			// CRITICAL: Use verified cleanup to prevent agent remaining in registry
			// Agent should have exited naturally, but cleanup() safely handles already-terminated processes
			qp.cleanupAgentAfterFinalizationFailure(task.ID, cleanup, "completed task finalization failure")
			return
		}

		// Save execution history and output
		outputRelPath, err := qp.saveOutputToHistory(task.ID, executionID)
		if err != nil {
			log.Printf("Warning: Failed to save output to history for task %s execution %s: %v", task.ID, executionID, err)
			systemlog.Warnf("Output save failed for execution %s/%s: %v", task.ID, executionID, err)
		}
		history.OutputPath = outputRelPath
		if err := qp.saveExecutionMetadata(history); err != nil {
			log.Printf("Warning: Failed to save execution metadata for task %s execution %s: %v", task.ID, executionID, err)
			systemlog.Warnf("Metadata save failed for execution %s/%s: %v", task.ID, executionID, err)
		}

		// Only broadcast after successful finalization
		qp.broadcastUpdate("task_completed", task)
		if qp.recycler != nil && task.ProcessorAutoRequeue {
			qp.recycler.Enqueue(task.ID)
		}

		// Finalization succeeded, safe to unregister and cleanup
		// Agent should have exited naturally, but cleanup() handles already-terminated processes safely
		qp.completeTaskCleanupExplicit(task.ID, cleanup)
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
			task.CooldownUntil = ""

			// Store rate limit info using structured results
			hitAt := timeutil.NowRFC3339()
			taskResults := tasks.NewRateLimitResults(hitAt, result.RetryAfter)
			task.Results = taskResults.ToMap()

			// Update execution history
			history.EndTime = time.Now()
			history.Duration = executionTime.Round(time.Second).String()
			history.Success = false
			history.ExitReason = "rate_limited"
			history.RateLimited = true
			history.RetryAfter = result.RetryAfter

			// Move back to pending queue for retry
			if finalizeErr := qp.finalizeTaskStatus(&task, "pending"); finalizeErr != nil {
				log.Printf("CRITICAL: Failed to finalize rate-limited task %s: %v", task.ID, finalizeErr)
				systemlog.Errorf("Failed to move rate-limited task %s to pending: %v - task will remain in executions", task.ID, finalizeErr)
				// CRITICAL: Use verified cleanup to prevent agent remaining in registry
				qp.cleanupAgentAfterFinalizationFailure(task.ID, cleanup, "rate-limited task finalization failure")
				return
			}

			// Save execution history
			outputRelPath, err := qp.saveOutputToHistory(task.ID, executionID)
			if err != nil {
				log.Printf("Warning: Failed to save output to history for task %s execution %s: %v", task.ID, executionID, err)
				systemlog.Warnf("Output save failed for execution %s/%s: %v", task.ID, executionID, err)
			}
			history.OutputPath = outputRelPath
			if err := qp.saveExecutionMetadata(history); err != nil {
				log.Printf("Warning: Failed to save execution metadata for task %s execution %s: %v", task.ID, executionID, err)
				systemlog.Warnf("Metadata save failed for execution %s/%s: %v", task.ID, executionID, err)
			}

			// Finalization succeeded, cleanup agent and unregister
			qp.completeTaskCleanupExplicit(task.ID, cleanup)

			// Trigger a pause of the queue processor
			qp.handleRateLimitPause(result.RetryAfter)

			// Broadcast rate limit event
			qp.broadcastUpdate("rate_limit_hit", map[string]any{
				"task_id":     task.ID,
				"retry_after": result.RetryAfter,
				"pause_until": time.Now().Add(time.Duration(result.RetryAfter) * time.Second).Format(time.RFC3339),
			})
		} else {
			log.Printf("Task %s failed after %v: %s", task.ID, executionTime.Round(time.Second), result.Error)
			systemlog.Warnf("Task %s failed: %s", task.ID, result.Error)
			var extras map[string]any
			if result.MaxTurnsExceeded {
				extras = map[string]any{"max_turns_exceeded": true}
				// MAX_TURNS is particularly prone to zombie agents because the agent
				// may not have fully exited yet when we start cleanup
				log.Printf("⚠️  MAX_TURNS exceeded for task %s - using enhanced cleanup verification", task.ID)
				systemlog.Warnf("MAX_TURNS exceeded for task %s - enhanced cleanup will be applied", task.ID)
			}
			if result.IdleTimeout {
				if extras == nil {
					extras = make(map[string]any)
				}
				extras["idle_timeout"] = true
			}
			if finalizeErr := qp.handleTaskFailureWithTiming(&task, result.Error, result.Output, executionStartTime, executionTime, timeoutDuration, extras); finalizeErr != nil {
				log.Printf("CRITICAL: Failed to finalize failed task %s: %v", task.ID, finalizeErr)
				systemlog.Errorf("Failed to finalize failed task %s: %v - task will remain in executions", task.ID, finalizeErr)
				// CRITICAL: Use verified cleanup to prevent agent remaining in registry
				// This is especially important for MAX_TURNS cases where agent may still be running
				context := "failed task finalization failure"
				if result.MaxTurnsExceeded {
					context = "MAX_TURNS finalization failure"
					// Give agent extra time to fully exit after MAX_TURNS before cleanup
					time.Sleep(scaleDuration(MaxTurnsCleanupDelay))
				}
				qp.cleanupAgentAfterFinalizationFailure(task.ID, cleanup, context)
				return
			}
			// Finalization succeeded (handleTaskFailureWithTiming calls finalizeTaskStatus)

			// Save execution history for failed tasks
			outputRelPath, err := qp.saveOutputToHistory(task.ID, executionID)
			if err != nil {
				log.Printf("Warning: Failed to save output to history for task %s execution %s: %v", task.ID, executionID, err)
				systemlog.Warnf("Output save failed for execution %s/%s: %v", task.ID, executionID, err)
			}
			history.OutputPath = outputRelPath
			history.EndTime = time.Now()
			history.Duration = executionTime.Round(time.Second).String()
			history.Success = false
			if result.MaxTurnsExceeded {
				history.ExitReason = "max_turns_exceeded"
			} else {
				history.ExitReason = "failed"
			}
			if err := qp.saveExecutionMetadata(history); err != nil {
				log.Printf("Warning: Failed to save execution metadata for task %s execution %s: %v", task.ID, executionID, err)
				systemlog.Warnf("Metadata save failed for execution %s/%s: %v", task.ID, executionID, err)
			}

			// For MAX_TURNS, give agent extra time to fully exit
			if result.MaxTurnsExceeded {
				time.Sleep(scaleDuration(MaxTurnsCleanupDelay))
			}
			// Cleanup via agent-manager (cleanup func calls agentSvc.StopRun)
			qp.completeTaskCleanupExplicit(task.ID, cleanup)
		}
	}
}

// handleTaskFailureWithTiming handles a failed task with execution timing information
func (qp *Processor) handleTaskFailureWithTiming(task *tasks.TaskItem, errorMsg string, output string, startTime time.Time, executionTime time.Duration, timeoutAllowed time.Duration, extras map[string]any) error {
	// Determine if this was a timeout failure
	isTimeout := strings.Contains(errorMsg, "timed out") || strings.Contains(errorMsg, "timeout")

	// Get the prompt for size calculation (if available)
	assembly, _ := qp.assembler.AssemblePromptForTask(*task)
	prompt := assembly.Prompt
	promptSizeKB := float64(len(prompt)) / BytesPerKilobyte

	failedAt := timeutil.NowRFC3339()

	// Use structured results
	taskResults := tasks.NewFailureResults(
		errorMsg,
		output,
		executionTime.Round(time.Second).String(),
		timeoutAllowed.String(),
		startTime.Format(time.RFC3339),
		failedAt,
		fmt.Sprintf("%d chars (%.2f KB)", len(prompt), promptSizeKB),
		isTimeout,
		extras,
	)
	task.Results = taskResults.ToMap()

	task.CurrentPhase = "failed"
	task.Status = "failed"
	task.CompletedAt = timeutil.NowRFC3339()

	if task.ProcessorAutoRequeue {
		cooldownSeconds := settings.GetSettings().CooldownSeconds
		if cooldownSeconds > 0 {
			task.CooldownUntil = time.Now().Add(time.Duration(cooldownSeconds) * time.Second).Format(time.RFC3339)
		} else {
			task.CooldownUntil = ""
		}
	} else {
		task.CooldownUntil = ""
	}

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

// callClaudeCode executes Claude Code via agent-manager for centralized agent orchestration.
// This function delegates to callClaudeCodeViaAgentManager which handles all agent lifecycle management.
func (qp *Processor) callClaudeCode(prompt string, task tasks.TaskItem, executionID string, startTime time.Time, timeoutDuration time.Duration, history *ExecutionHistory) (*tasks.ClaudeCodeResponse, func(), error) {
	// Delegate to agent-manager for execution
	return qp.callClaudeCodeViaAgentManager(prompt, task, executionID, startTime, timeoutDuration, history)
}

func (qp *Processor) handleNonZeroExit(waitErr error, combinedOutput string, task tasks.TaskItem, agentTag string, maxTurns int, elapsed time.Duration) (*tasks.ClaudeCodeResponse, bool) {
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

	// Use consolidated rate limit detection
	detection := ratelimit.DetectFromError(waitErr, combinedOutput, elapsed)
	if detection.IsRateLimited {
		// Only trigger rate limit handling if within detection window OR exit code was 429
		if !detection.CheckWindow || elapsed <= ratelimit.DetectionWindow {
			qp.appendTaskLog(task.ID, agentTag, "stderr", fmt.Sprintf("🚫 Rate limit hit. Pausing for %d seconds", detection.RetryAfter))
			return &tasks.ClaudeCodeResponse{
				Success:     false,
				Error:       "RATE_LIMIT: API rate limit reached",
				RateLimited: true,
				RetryAfter:  detection.RetryAfter,
				Output:      combinedOutput,
			}, true
		}
	}

	log.Printf("Claude Code exited with non-zero status for task %s", task.ID)
	return &tasks.ClaudeCodeResponse{
		Success: false,
		Error:   fmt.Sprintf("Claude Code execution failed: %s", combinedOutput),
		Output:  combinedOutput,
	}, true
}

func detectMaxTurnsExceeded(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "max turns") && strings.Contains(lower, "reached")
}

// callClaudeCodeViaAgentManager executes a task using the agent-manager service.
// This is the preferred execution path that delegates agent lifecycle management to agent-manager.
func (qp *Processor) callClaudeCodeViaAgentManager(prompt string, task tasks.TaskItem, executionID string, startTime time.Time, timeoutDuration time.Duration, history *ExecutionHistory) (*tasks.ClaudeCodeResponse, func(), error) {
	cleanup := func() {}
	agentTag := makeAgentTag(task.ID)

	log.Printf("Executing task %s via agent-manager (prompt length: %d characters, timeout: %v)", task.ID, len(prompt), timeoutDuration)

	// Check agent-manager availability
	ctx := context.Background()
	if !qp.agentSvc.IsAvailable(ctx) {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   "agent-manager is not available",
		}, cleanup, fmt.Errorf("agent-manager not available")
	}

	// Create execution directory for artifacts
	execDir := qp.getExecutionDir(task.ID, executionID)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		log.Printf("Warning: failed to ensure execution directory %s: %v", execDir, err)
	}

	// Broadcast task started
	qp.broadcastUpdate("task_started", map[string]any{
		"task_id":    task.ID,
		"agent_id":   agentTag,
		"start_time": timeutil.NowRFC3339(),
	})

	// Update task phase
	task.CurrentPhase = "executing_claude"
	task.StartedAt = startTime.Format(time.RFC3339)
	if err := qp.storage.SaveQueueItemSkipCleanup(task, "in-progress"); err != nil {
		log.Printf("Warning: failed to persist in-progress task %s: %v", task.ID, err)
	}
	qp.broadcastUpdate("task_executing", task)

	// Start async execution via agent-manager
	runID, err := qp.agentSvc.ExecuteTaskAsync(ctx, agentmanager.ExecuteRequest{
		Task:    task,
		Prompt:  prompt,
		Timeout: timeoutDuration,
		Tag:     agentTag,
	})
	if err != nil {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to start agent-manager run: %v", err),
		}, cleanup, nil
	}

	// Register execution for tracking
	qp.registerExecutionByRunID(task.ID, agentTag, runID, startTime)
	qp.initTaskLogBuffer(task.ID, agentTag, 0)

	// Cleanup function stops the run via agent-manager
	cleanup = func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := qp.agentSvc.StopRun(stopCtx, runID); err != nil {
			log.Printf("Warning: failed to stop agent-manager run %s: %v", runID, err)
		}
		qp.unregisterExecution(task.ID)
	}

	// Stream events in background
	var outputBuilder strings.Builder
	eventsDone := make(chan struct{})
	go qp.streamAgentManagerEvents(ctx, runID, task.ID, agentTag, &outputBuilder, eventsDone)

	// Wait for run to complete
	waitCtx, waitCancel := context.WithTimeout(ctx, timeoutDuration+30*time.Second)
	defer waitCancel()

	completedRun, err := qp.agentSvc.WaitForRun(waitCtx, runID)
	close(eventsDone) // Signal event streaming to stop

	if err != nil {
		if waitCtx.Err() == context.DeadlineExceeded {
			msg := fmt.Sprintf("TIMEOUT: Task execution exceeded %v limit", timeoutDuration)
			qp.appendTaskLog(task.ID, agentTag, "stderr", msg)
			return &tasks.ClaudeCodeResponse{
				Success: false,
				Error:   msg,
				Output:  outputBuilder.String(),
			}, cleanup, nil
		}
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed waiting for agent-manager run: %v", err),
			Output:  outputBuilder.String(),
		}, cleanup, nil
	}

	// Map agent-manager result to ClaudeCodeResponse
	response := qp.mapAgentManagerResult(completedRun, task, agentTag, outputBuilder.String(), history)

	// Finalize logs
	qp.finalizeTaskLogs(task.ID, response.Success)

	// Update task phase
	if response.Success {
		task.CurrentPhase = "claude_completed"
		qp.broadcastUpdate("claude_execution_complete", task)
	}

	return response, cleanup, nil
}

// registerExecutionByRunID registers a task execution using agent-manager run ID instead of exec.Cmd.
func (qp *Processor) registerExecutionByRunID(taskID, agentTag, runID string, started time.Time) {
	qp.registry.ReserveExecution(taskID, agentTag, started)
	qp.registry.RegisterRunID(taskID, runID)
}

// streamAgentManagerEvents polls for agent-manager events and forwards them to the log system.
func (qp *Processor) streamAgentManagerEvents(ctx context.Context, runID, taskID, agentTag string, outputBuilder *strings.Builder, done <-chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastSeq int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			events, err := qp.agentSvc.GetRunEvents(ctx, runID, lastSeq)
			if err != nil {
				continue
			}

			for _, evt := range events {
				qp.handleAgentManagerEvent(taskID, agentTag, evt, outputBuilder)
				if evt.Sequence > lastSeq {
					lastSeq = evt.Sequence
				}
			}
		}
	}
}

// handleAgentManagerEvent processes a single agent-manager event.
func (qp *Processor) handleAgentManagerEvent(taskID, agentTag string, evt *domainpb.RunEvent, outputBuilder *strings.Builder) {
	switch evt.EventType {
	case domainpb.RunEventType_RUN_EVENT_TYPE_LOG:
		if evt.GetLog() != nil {
			msg := evt.GetLog().Message
			qp.appendTaskLog(taskID, agentTag, "stdout", msg)
			outputBuilder.WriteString(msg)
			outputBuilder.WriteByte('\n')
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
		if evt.GetMessage() != nil {
			msg := evt.GetMessage().Content
			qp.appendTaskLog(taskID, agentTag, "stdout", msg)
			outputBuilder.WriteString(msg)
			outputBuilder.WriteByte('\n')
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
		if evt.GetToolCall() != nil {
			tc := evt.GetToolCall()
			msg := fmt.Sprintf("[Tool: %s]", tc.ToolName)
			qp.appendTaskLog(taskID, agentTag, "stdout", msg)
			qp.broadcastUpdate("tool_call", map[string]any{
				"task_id": taskID,
				"tool":    tc.ToolName,
			})
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
		if evt.GetToolResult() != nil {
			tr := evt.GetToolResult()
			if !tr.Success && tr.Error != "" {
				qp.appendTaskLog(taskID, agentTag, "stderr", tr.Error)
			}
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_ERROR:
		if evt.GetError() != nil {
			qp.appendTaskLog(taskID, agentTag, "stderr", evt.GetError().Message)
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_STATUS:
		if evt.GetStatus() != nil {
			s := evt.GetStatus()
			msg := fmt.Sprintf("Status: %s -> %s", s.OldStatus, s.NewStatus)
			qp.appendTaskLog(taskID, agentTag, "stdout", msg)
		}
	}
}

// mapAgentManagerResult converts agent-manager Run result to ClaudeCodeResponse.
func (qp *Processor) mapAgentManagerResult(run *domainpb.Run, task tasks.TaskItem, agentTag, output string, history *ExecutionHistory) *tasks.ClaudeCodeResponse {
	currentSettings := settings.GetSettings()

	response := &tasks.ClaudeCodeResponse{
		Success: run.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE,
		Output:  output,
	}

	if run.Summary != nil {
		response.Message = run.Summary.Description
		if run.Summary.Description != "" {
			response.FinalMessage = run.Summary.Description
		}
	}

	if run.ErrorMsg != "" {
		response.Error = run.ErrorMsg
		lowerErr := strings.ToLower(run.ErrorMsg)

		// Detect rate limiting
		if strings.Contains(lowerErr, "rate limit") || strings.Contains(lowerErr, "429") {
			response.RateLimited = true
			response.RetryAfter = DefaultRateLimitRetry
		}

		// Detect timeout
		if strings.Contains(lowerErr, "timeout") {
			response.Error = fmt.Sprintf("TIMEOUT: Task execution exceeded limit. %s", run.ErrorMsg)
		}
	}

	// Detect MAX_TURNS from error message or output
	if detectMaxTurnsExceeded(run.ErrorMsg) || detectMaxTurnsExceeded(output) {
		response.MaxTurnsExceeded = true
		response.Success = false
		response.Error = fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", currentSettings.MaxTurns)
		qp.appendTaskLog(task.ID, agentTag, "stderr", response.Error)
	}

	// Update history if provided
	if history != nil {
		if run.StartedAt != nil && run.EndedAt != nil {
			duration := run.EndedAt.AsTime().Sub(run.StartedAt.AsTime())
			history.Duration = duration.String()
		}
	}

	return response
}

func createTranscriptPaths(execDir, agentTag string, now func() time.Time) (string, string) {
	if now == nil {
		now = time.Now
	}
	_ = os.MkdirAll(execDir, 0o755)
	base := fmt.Sprintf("%s-%d", sanitizeForFilename(agentTag), now().UnixNano())
	return filepath.Join(execDir, fmt.Sprintf("%s-conversation.jsonl", base)),
		filepath.Join(execDir, fmt.Sprintf("%s-last.txt", base))
}

func sanitizeForFilename(input string) string {
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", "|", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "%", "-", "$", "-", "@", "-", "!", "-", "#", "-")
	trimmed := strings.TrimSpace(input)
	sanitized := replacer.Replace(trimmed)
	if sanitized == "" {
		return "agent"
	}
	return sanitized
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

func fileExistsAndNotEmpty(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// extractFinalAgentMessage attempts to pull the final agent-authored content from combined output.
// This mirrors the app-issue-tracker approach and stays robust even if new agents are added.
func extractFinalAgentMessage(output string) string {
	cleaned := strings.TrimSpace(output)
	if cleaned == "" {
		return ""
	}

	lines := strings.Split(cleaned, "\n")
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" || strings.HasPrefix(last, "[20") || strings.HasPrefix(strings.ToUpper(last), "[SUCCESS") {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}
	cleaned = strings.TrimSpace(strings.Join(lines, "\n"))
	if cleaned == "" {
		return ""
	}

	lower := strings.ToLower(cleaned)
	if idx := strings.LastIndex(lower, "**summary**"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[idx:])
	} else if idx := strings.LastIndex(lower, "summary:"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[idx:])
	}

	return cleaned
}

func readLastMessageFile(path string) string {
	if !fileExistsAndNotEmpty(path) {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeFallbackTranscript(path, prompt, finalMessage, combinedOutput string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if strings.TrimSpace(prompt) == "" && strings.TrimSpace(finalMessage) == "" && strings.TrimSpace(combinedOutput) == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	entries := []map[string]any{
		{"sandbox": map[string]any{"provider": "claude-code"}},
		{"prompt": prompt},
	}

	if msg := strings.TrimSpace(finalMessage); msg != "" {
		entries = append(entries, map[string]any{
			"msg": map[string]any{
				"role":    "assistant",
				"content": []map[string]string{{"type": "text", "text": msg}},
			},
		})
	} else if trimmed := strings.TrimSpace(combinedOutput); trimmed != "" {
		entries = append(entries, map[string]any{
			"msg": map[string]any{
				"role":    "assistant",
				"content": []map[string]string{{"type": "text", "text": trimmed}},
			},
		})
	}

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
	}

	return nil
}

// saveCleanOutputs persists clean agent-facing artifacts (clean output, last message, transcript)
// alongside the legacy timestamped output.log. Returns relative paths for metadata/bookkeeping.
func (qp *Processor) saveCleanOutputs(taskID, executionID, execDir, prompt, combinedOutput, finalMessage, transcriptFile, lastMessageFile string) (string, string, string) {
	makeRel := func(filename string) string {
		return filepath.Join(taskID, "executions", executionID, filename)
	}

	// Ensure execution directory exists
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		log.Printf("Warning: unable to ensure execution directory %s: %v", execDir, err)
	}

	var cleanRel, lastRel, transcriptRel string

	// Clean output (no timestamp prefixes)
	if strings.TrimSpace(combinedOutput) != "" {
		cleanPath := filepath.Join(execDir, "clean_output.txt")
		if err := os.WriteFile(cleanPath, []byte(combinedOutput), 0o644); err != nil {
			log.Printf("Warning: failed to persist clean output for %s/%s: %v", taskID, executionID, err)
		} else {
			cleanRel = makeRel("clean_output.txt")
		}
	}

	// Last message (prefer agent-written file, otherwise write fallback)
	lastPath := lastMessageFile
	if strings.TrimSpace(lastPath) == "" {
		lastPath = filepath.Join(execDir, "last_message.txt")
	}
	if fileExistsAndNotEmpty(lastPath) {
		lastRel = makeRel(filepath.Base(lastPath))
	} else if strings.TrimSpace(finalMessage) != "" {
		if err := os.WriteFile(lastPath, []byte(finalMessage), 0o644); err != nil {
			log.Printf("Warning: failed to persist last message for %s/%s: %v", taskID, executionID, err)
		} else {
			lastRel = makeRel(filepath.Base(lastPath))
		}
	}

	// Transcript (prefer agent-written file, otherwise create fallback)
	transcriptPath := transcriptFile
	if strings.TrimSpace(transcriptPath) == "" {
		transcriptPath = filepath.Join(execDir, "conversation.jsonl")
	}
	if fileExistsAndNotEmpty(transcriptPath) {
		transcriptRel = makeRel(filepath.Base(transcriptPath))
	} else if err := writeFallbackTranscript(transcriptPath, prompt, finalMessage, combinedOutput); err != nil {
		log.Printf("Warning: failed to persist transcript for %s/%s: %v", taskID, executionID, err)
	} else {
		transcriptRel = makeRel(filepath.Base(transcriptPath))
	}

	return cleanRel, lastRel, transcriptRel
}

func isValidClaudeConfig(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}

	var parsed map[string]any
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return false
	}

	return true
}

// broadcastUpdate sends updates to all connected WebSocket clients
func (qp *Processor) broadcastUpdate(updateType string, data any) {
	// Send the typed update directly, not wrapped in another object
	// The WebSocket manager will wrap it properly
	select {
	case qp.broadcast <- map[string]any{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}:
		log.Printf("Broadcast %s update for task", updateType)
	default:
		log.Printf("Warning: WebSocket broadcast channel full, dropping update")
	}
}

// getScenarioNameFromTask extracts the scenario name from a task
// For scenario-related tasks, returns the target scenario name
func getScenarioNameFromTask(task *tasks.TaskItem) string {
	if task.Type == "scenario" && task.Target != "" {
		return task.Target
	}
	// For multiple targets, use the first one
	if len(task.Targets) > 0 {
		return task.Targets[0]
	}
	// Fallback: try to extract from title or use task ID
	return task.Target
}

// GetScenarioNameFromTask is an exported helper to ensure consistent scenario name derivation.
func GetScenarioNameFromTask(task *tasks.TaskItem) string {
	return getScenarioNameFromTask(task)
}
