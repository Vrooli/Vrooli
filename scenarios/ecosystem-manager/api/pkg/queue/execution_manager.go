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
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/ratelimit"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/steering"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// ExecutionManagerDeps contains dependencies for the ExecutionManager.
type ExecutionManagerDeps struct {
	Storage              tasks.StorageAPI
	Assembler            prompts.AssemblerAPI
	AgentSvc             agentmanager.AgentServiceAPI
	Registry             ExecutionRegistryAPI
	TaskLogger           *TaskLogger
	Broadcast            chan<- any
	AutoSteerIntegration *AutoSteerIntegration
	SteeringRegistry     steering.RegistryAPI
	HistoryManager       HistoryManagerAPI
	TaskLogsDir          string
}

// ExecutionManager handles individual task execution lifecycle.
// It separates the "doing the work" from scheduling decisions.
type ExecutionManager struct {
	storage              tasks.StorageAPI
	assembler            prompts.AssemblerAPI
	agentSvc             agentmanager.AgentServiceAPI
	registry             ExecutionRegistryAPI
	taskLogger           *TaskLogger
	broadcast            chan<- any
	autoSteerIntegration *AutoSteerIntegration
	steeringRegistry     steering.RegistryAPI
	historyManager       HistoryManagerAPI
	taskLogsDir          string

	// Callback for waking the processor after execution completes
	wakeFunc func()

	// Callback for finalizing task status (handles coordinator/lifecycle)
	finalizeFunc func(task *tasks.TaskItem, toStatus string) error
}

// NewExecutionManager creates a new ExecutionManager with the given dependencies.
func NewExecutionManager(deps ExecutionManagerDeps) *ExecutionManager {
	return &ExecutionManager{
		storage:              deps.Storage,
		assembler:            deps.Assembler,
		agentSvc:             deps.AgentSvc,
		registry:             deps.Registry,
		taskLogger:           deps.TaskLogger,
		broadcast:            deps.Broadcast,
		autoSteerIntegration: deps.AutoSteerIntegration,
		steeringRegistry:     deps.SteeringRegistry,
		historyManager:       deps.HistoryManager,
		taskLogsDir:          deps.TaskLogsDir,
	}
}

// SetWakeFunc sets the callback for waking the processor after execution completes.
func (em *ExecutionManager) SetWakeFunc(f func()) {
	em.wakeFunc = f
}

// SetFinalizeFunc sets the callback for finalizing task status.
func (em *ExecutionManager) SetFinalizeFunc(f func(task *tasks.TaskItem, toStatus string) error) {
	em.finalizeFunc = f
}

// SetAutoSteerIntegration updates the Auto Steer integration.
func (em *ExecutionManager) SetAutoSteerIntegration(integration *AutoSteerIntegration) {
	em.autoSteerIntegration = integration
}

// SetSteeringRegistry sets the steering registry for unified steering strategy dispatch.
func (em *ExecutionManager) SetSteeringRegistry(registry steering.RegistryAPI) {
	em.steeringRegistry = registry
}

// GetExecutionDir returns the directory for a specific execution.
// Delegates to HistoryManager when available, falls back to local implementation.
func (em *ExecutionManager) GetExecutionDir(taskID, executionID string) string {
	if em.historyManager != nil {
		return em.historyManager.GetExecutionDir(taskID, executionID)
	}
	return filepath.Join(em.taskLogsDir, taskID, "executions", executionID)
}

// GetExecutionFilePath returns the full path to a file within an execution directory.
// Delegates to HistoryManager when available.
func (em *ExecutionManager) GetExecutionFilePath(taskID, executionID, filename string) string {
	if em.historyManager != nil {
		return em.historyManager.GetExecutionFilePath(taskID, executionID, filename)
	}
	return filepath.Join(em.GetExecutionDir(taskID, executionID), filename)
}

// ExecuteTask executes a single task and handles all execution lifecycle.
func (em *ExecutionManager) ExecuteTask(ctx context.Context, task tasks.TaskItem) (*ExecutionResult, error) {
	log.Printf("ExecutionManager: Executing task %s: %s", task.ID, task.Title)
	systemlog.Infof("ExecutionManager: Executing task %s (%s)", task.ID, task.Title)

	if em.wakeFunc != nil {
		defer em.wakeFunc()
	}

	// Track execution timing
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

	// Get timeout setting
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute
	history.TimeoutAllowed = timeoutDuration.String()

	// Initialize Auto Steer if needed
	autoSteerInitFailed := false
	if em.autoSteerIntegration != nil && task.AutoSteerProfileID != "" {
		scenarioName := GetScenarioNameFromTask(&task)
		if err := em.autoSteerIntegration.InitializeAutoSteer(&task, scenarioName); err != nil {
			log.Printf("Failed to initialize Auto Steer for task %s: %v", task.ID, err)
			systemlog.Errorf("Auto Steer initialization failed for task %s: %v", task.ID, err)
			autoSteerInitFailed = true
		}
	}

	// Surface latest execution output path for prompt templating
	if latest := em.LatestExecutionOutputPath(task.ID); latest != "" {
		task.LatestOutputPath = latest
	}

	// Generate the full prompt
	assembly, err := em.assembler.AssemblePromptForTask(task)
	if err != nil {
		executionTime := time.Since(executionStartTime)
		log.Printf("Failed to assemble prompt for task %s: %v", task.ID, err)
		return em.handleFailure(&task, fmt.Sprintf("Prompt assembly failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil)
	}
	prompt := assembly.Prompt

	// Apply steering
	prompt = em.applySteeringToPrompt(prompt, &task)

	history.PromptSize = len(prompt)
	em.enrichSteeringMetadata(&task, &history)

	// Store the assembled prompt
	promptRelPath, err := em.savePromptToHistory(task.ID, executionID, prompt)
	if err != nil {
		log.Printf("Warning: Failed to save prompt to history for %s: %v", task.ID, err)
	} else {
		history.PromptPath = promptRelPath
	}

	// Update task phase
	task.CurrentPhase = "prompt_assembled"
	em.storage.SaveQueueItemSkipCleanup(task, "in-progress")
	em.broadcastUpdate("task_progress", task)

	// Log prompt size
	promptSizeKB := float64(len(prompt)) / BytesPerKilobyte
	log.Printf("Task %s: Prompt size: %d characters (%.2f KB)", task.ID, len(prompt), promptSizeKB)

	// Execute via agent-manager
	result, cleanup, err := em.callClaudeCode(ctx, prompt, task, executionID, executionStartTime, timeoutDuration, &history)
	executionTime := time.Since(executionStartTime)

	if err != nil {
		log.Printf("Failed to execute task %s with Claude Code: %v", task.ID, err)
		execResult, finalizeErr := em.handleFailure(&task, fmt.Sprintf("Claude Code execution failed: %v", err), "", executionStartTime, executionTime, timeoutDuration, nil)
		if finalizeErr != nil {
			em.cleanupAfterFailure(task.ID, cleanup, "execution failure")
			return execResult, finalizeErr
		}
		em.completeCleanup(task.ID, cleanup)
		return execResult, nil
	}

	// Process the result
	return em.processExecutionResult(result, &task, &history, executionID, executionStartTime, executionTime, timeoutDuration, promptSizeKB, autoSteerInitFailed, cleanup)
}

// applySteeringToPrompt applies steering context to the prompt.
func (em *ExecutionManager) applySteeringToPrompt(prompt string, task *tasks.TaskItem) string {
	// Only apply steering to scenario improver tasks
	if task.Type != "scenario" || task.Operation != "improver" {
		return prompt
	}

	// Use unified steering registry when available
	if em.steeringRegistry != nil {
		provider := em.steeringRegistry.GetProvider(task)
		if provider != nil {
			enhancement, err := provider.EnhancePrompt(task)
			if err != nil {
				log.Printf("Warning: Steering enhancement failed for task %s: %v", task.ID, err)
			} else if enhancement != nil && enhancement.Section != "" {
				log.Printf("Prompt enhanced with steering (%s) for task %s", enhancement.Source, task.ID)
				return autosteer.InjectSteeringSection(prompt, enhancement.Section)
			}
		}
	}

	// Fallback: Use legacy steering logic for backward compatibility
	return em.applySteeringToPromptLegacy(prompt, task)
}

// applySteeringToPromptLegacy is the original steering logic for backward compatibility.
// Used when steeringRegistry is not configured.
func (em *ExecutionManager) applySteeringToPromptLegacy(prompt string, task *tasks.TaskItem) string {
	// Apply default Progress steering when Auto Steer is not configured
	if strings.TrimSpace(task.AutoSteerProfileID) == "" {
		mode := autosteer.SteerMode(strings.ToLower(strings.TrimSpace(task.SteerMode)))
		prompt = em.appendManualSteeringSection(prompt, mode)
	}

	// Enhance prompt with Auto Steer context if active
	if em.autoSteerIntegration != nil && task.AutoSteerProfileID != "" {
		enhancedPrompt, err := em.autoSteerIntegration.EnhancePrompt(task, prompt)
		if err != nil {
			log.Printf("Warning: Failed to enhance prompt with Auto Steer for task %s: %v", task.ID, err)
		} else if enhancedPrompt != prompt {
			prompt = enhancedPrompt
			log.Printf("Prompt enhanced with Auto Steer for task %s", task.ID)
		}
	}

	return prompt
}

// appendManualSteeringSection attaches steering mode guidance.
func (em *ExecutionManager) appendManualSteeringSection(prompt string, steerMode autosteer.SteerMode) string {
	if em.assembler == nil {
		return prompt
	}

	mode := steerMode
	if !mode.IsValid() {
		mode = autosteer.ModeProgress
	}

	if em.autoSteerIntegration != nil {
		if orchestrator := em.autoSteerIntegration.ExecutionOrchestrator(); orchestrator != nil {
			if section := strings.TrimSpace(orchestrator.GenerateModeSection(mode)); section != "" {
				return autosteer.InjectSteeringSection(prompt, section)
			}
		}
	}

	phasesDir := filepath.Join(em.assembler.GetPromptsDir(), "phases")
	section := strings.TrimSpace(autosteer.NewPromptEnhancer(phasesDir).GenerateModeSection(mode))
	return autosteer.InjectSteeringSection(prompt, section)
}

// handleSteeringContinuation determines whether a task should continue after execution.
// Uses the unified steering registry when available, falling back to legacy logic.
func (em *ExecutionManager) handleSteeringContinuation(task *tasks.TaskItem, autoSteerInitFailed bool) {
	// Only apply steering continuation to scenario improver tasks
	if task.Type != "scenario" || task.Operation != "improver" {
		return
	}

	// Use unified steering registry when available
	if em.steeringRegistry != nil {
		provider := em.steeringRegistry.GetProvider(task)
		if provider != nil {
			scenarioName := GetScenarioNameFromTask(task)
			decision, err := provider.AfterExecution(task, scenarioName)
			if err != nil {
				log.Printf("Warning: Steering evaluation failed for task %s: %v", task.ID, err)
				task.ProcessorAutoRequeue = false
				task.Status = tasks.StatusCompletedFinalized
				return
			}

			if decision.Exhausted || !decision.ShouldRequeue {
				log.Printf("Steering: Task %s completed (%s) - moving to finalized", task.ID, decision.Reason)
				task.ProcessorAutoRequeue = false
				task.Status = tasks.StatusCompletedFinalized
			} else if task.ProcessorAutoRequeue {
				log.Printf("Steering: Task %s will continue (%s) - requeuing", task.ID, decision.Reason)
				task.Status = "pending"
			}
			return
		}
	}

	// Fallback: Use legacy Auto Steer logic for backward compatibility
	em.handleSteeringContinuationLegacy(task, autoSteerInitFailed)
}

// handleSteeringContinuationLegacy is the original Auto Steer continuation logic.
// Used when steeringRegistry is not configured.
func (em *ExecutionManager) handleSteeringContinuationLegacy(task *tasks.TaskItem, autoSteerInitFailed bool) {
	if em.autoSteerIntegration != nil && task.AutoSteerProfileID != "" {
		if autoSteerInitFailed {
			log.Printf("Auto Steer: Task %s initialization failed - disabling auto-requeue", task.ID)
			task.ProcessorAutoRequeue = false
			task.Status = tasks.StatusCompletedFinalized
		} else {
			scenarioName := GetScenarioNameFromTask(task)
			shouldContinue, err := em.autoSteerIntegration.ShouldContinueTask(task, scenarioName)
			if err != nil {
				log.Printf("Warning: Auto Steer evaluation failed for task %s: %v", task.ID, err)
				task.ProcessorAutoRequeue = false
				task.Status = tasks.StatusCompletedFinalized
			} else if !shouldContinue {
				log.Printf("Auto Steer: Task %s completed all phases - moving to finalized", task.ID)
				task.ProcessorAutoRequeue = false
				task.Status = tasks.StatusCompletedFinalized
			} else if task.ProcessorAutoRequeue {
				log.Printf("Auto Steer: Task %s will continue - requeuing for next iteration", task.ID)
				task.Status = "pending"
			}
		}
	}
}

// enrichSteeringMetadata captures steering context for execution history.
func (em *ExecutionManager) enrichSteeringMetadata(task *tasks.TaskItem, history *ExecutionHistory) {
	if task == nil || history == nil {
		return
	}

	history.SteeringSource = "none"

	// Use steering registry to determine source when available
	if em.steeringRegistry != nil && task.Type == "scenario" && task.Operation == "improver" {
		strategy := em.steeringRegistry.DetermineStrategy(task)
		switch strategy {
		case steering.StrategyProfile:
			em.enrichSteeringMetadataProfile(task, history)
		case steering.StrategyQueue:
			em.enrichSteeringMetadataQueue(task, history)
		case steering.StrategyManual:
			em.enrichSteeringMetadataManual(task, history)
		default:
			em.enrichSteeringMetadataNone(task, history)
		}
		return
	}

	// Fallback: legacy logic
	em.enrichSteeringMetadataLegacy(task, history)
}

// enrichSteeringMetadataProfile populates history for profile steering.
func (em *ExecutionManager) enrichSteeringMetadataProfile(task *tasks.TaskItem, history *ExecutionHistory) {
	if em.autoSteerIntegration != nil {
		orchestrator := em.autoSteerIntegration.ExecutionOrchestrator()
		if orchestrator != nil {
			if state, err := orchestrator.GetExecutionState(task.ID); err == nil && state != nil {
				history.AutoSteerProfileID = state.ProfileID
				history.AutoSteerIteration = state.AutoSteerIteration
				history.SteerPhaseIndex = state.CurrentPhaseIndex + 1
				history.SteerPhaseIteration = state.CurrentPhaseIteration + 1
				history.SteeringSource = "auto_steer"
			}

			if mode, err := orchestrator.GetCurrentMode(task.ID); err == nil && mode != "" {
				history.SteerMode = string(mode)
				if history.SteeringSource == "none" {
					history.SteeringSource = "auto_steer"
				}
			}
		}
	}
}

// enrichSteeringMetadataQueue populates history for queue steering.
func (em *ExecutionManager) enrichSteeringMetadataQueue(task *tasks.TaskItem, history *ExecutionHistory) {
	provider := em.steeringRegistry.GetProvider(task)
	if provider == nil {
		return
	}

	mode, err := provider.GetCurrentMode(task.ID)
	if err == nil && mode != "" {
		history.SteerMode = string(mode)
	}
	history.SteeringSource = "steering_queue"
	history.SteerPhaseIndex = 1
	history.SteerPhaseIteration = 1
}

// enrichSteeringMetadataManual populates history for manual steering.
func (em *ExecutionManager) enrichSteeringMetadataManual(task *tasks.TaskItem, history *ExecutionHistory) {
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

// enrichSteeringMetadataNone populates history when no steering is configured.
func (em *ExecutionManager) enrichSteeringMetadataNone(task *tasks.TaskItem, history *ExecutionHistory) {
	history.SteerMode = string(autosteer.ModeProgress)
	history.SteeringSource = "default_progress"
	history.SteerPhaseIndex = 1
	history.SteerPhaseIteration = 1
}

// enrichSteeringMetadataLegacy is the original metadata enrichment logic.
func (em *ExecutionManager) enrichSteeringMetadataLegacy(task *tasks.TaskItem, history *ExecutionHistory) {
	if em.autoSteerIntegration != nil && strings.TrimSpace(task.AutoSteerProfileID) != "" {
		orchestrator := em.autoSteerIntegration.ExecutionOrchestrator()
		if orchestrator != nil {
			if state, err := orchestrator.GetExecutionState(task.ID); err == nil && state != nil {
				history.AutoSteerProfileID = state.ProfileID
				history.AutoSteerIteration = state.AutoSteerIteration
				history.SteerPhaseIndex = state.CurrentPhaseIndex + 1
				history.SteerPhaseIteration = state.CurrentPhaseIteration + 1
				history.SteeringSource = "auto_steer"
			}

			if mode, err := orchestrator.GetCurrentMode(task.ID); err == nil && mode != "" {
				history.SteerMode = string(mode)
				if history.SteeringSource == "none" {
					history.SteeringSource = "auto_steer"
				}
			}
		}
	}

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

// callClaudeCode executes via agent-manager.
func (em *ExecutionManager) callClaudeCode(ctx context.Context, prompt string, task tasks.TaskItem, executionID string, startTime time.Time, timeoutDuration time.Duration, history *ExecutionHistory) (*tasks.ClaudeCodeResponse, func(), error) {
	cleanup := func() {}
	agentTag := makeAgentTag(task.ID)

	log.Printf("Executing task %s via agent-manager (prompt length: %d characters, timeout: %v)", task.ID, len(prompt), timeoutDuration)

	if !em.agentSvc.IsAvailable(ctx) {
		return &tasks.ClaudeCodeResponse{
			Success: false,
			Error:   "agent-manager is not available",
		}, cleanup, fmt.Errorf("agent-manager not available")
	}

	execDir := em.GetExecutionDir(task.ID, executionID)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		log.Printf("Warning: failed to ensure execution directory %s: %v", execDir, err)
	}

	em.broadcastUpdate("task_started", map[string]any{
		"task_id":    task.ID,
		"agent_id":   agentTag,
		"start_time": timeutil.NowRFC3339(),
	})

	task.CurrentPhase = "executing_claude"
	task.StartedAt = startTime.Format(time.RFC3339)
	if err := em.storage.SaveQueueItemSkipCleanup(task, "in-progress"); err != nil {
		log.Printf("Warning: failed to persist in-progress task %s: %v", task.ID, err)
	}
	em.broadcastUpdate("task_executing", task)

	runID, err := em.agentSvc.ExecuteTaskAsync(ctx, agentmanager.ExecuteRequest{
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

	em.registry.ReserveExecution(task.ID, agentTag, startTime)
	em.registry.RegisterRunID(task.ID, runID)
	em.initTaskLogBuffer(task.ID, agentTag, 0)

	cleanup = func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := em.agentSvc.StopRun(stopCtx, runID); err != nil {
			log.Printf("Warning: failed to stop agent-manager run %s: %v", runID, err)
		}
		em.registry.UnregisterExecution(task.ID)
	}

	var outputBuilder strings.Builder
	eventsDone := make(chan struct{})
	go em.streamAgentManagerEvents(ctx, runID, task.ID, agentTag, &outputBuilder, eventsDone)

	waitCtx, waitCancel := context.WithTimeout(ctx, timeoutDuration+30*time.Second)
	defer waitCancel()

	completedRun, err := em.agentSvc.WaitForRun(waitCtx, runID)
	close(eventsDone)

	if err != nil {
		if waitCtx.Err() == context.DeadlineExceeded {
			msg := fmt.Sprintf("TIMEOUT: Task execution exceeded %v limit", timeoutDuration)
			em.appendTaskLog(task.ID, agentTag, "stderr", msg)
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

	response := em.mapAgentManagerResult(completedRun, task, agentTag, outputBuilder.String(), history)
	em.finalizeTaskLogs(task.ID, response.Success)

	if response.Success {
		task.CurrentPhase = "claude_completed"
		em.broadcastUpdate("claude_execution_complete", task)
	}

	return response, cleanup, nil
}

// streamAgentManagerEvents polls for events and forwards them to the log system.
func (em *ExecutionManager) streamAgentManagerEvents(ctx context.Context, runID, taskID, agentTag string, outputBuilder *strings.Builder, done <-chan struct{}) {
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
			events, err := em.agentSvc.GetRunEvents(ctx, runID, lastSeq)
			if err != nil {
				continue
			}

			for _, evt := range events {
				em.handleAgentManagerEvent(taskID, agentTag, evt, outputBuilder)
				if evt.Sequence > lastSeq {
					lastSeq = evt.Sequence
				}
			}
		}
	}
}

// handleAgentManagerEvent processes a single agent-manager event.
func (em *ExecutionManager) handleAgentManagerEvent(taskID, agentTag string, evt *domainpb.RunEvent, outputBuilder *strings.Builder) {
	switch evt.EventType {
	case domainpb.RunEventType_RUN_EVENT_TYPE_LOG:
		if evt.GetLog() != nil {
			msg := evt.GetLog().Message
			em.appendTaskLog(taskID, agentTag, "stdout", msg)
			outputBuilder.WriteString(msg)
			outputBuilder.WriteByte('\n')
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
		if evt.GetMessage() != nil {
			msg := evt.GetMessage().Content
			em.appendTaskLog(taskID, agentTag, "stdout", msg)
			outputBuilder.WriteString(msg)
			outputBuilder.WriteByte('\n')
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
		if evt.GetToolCall() != nil {
			tc := evt.GetToolCall()
			msg := fmt.Sprintf("[Tool: %s]", tc.ToolName)
			em.appendTaskLog(taskID, agentTag, "stdout", msg)
			em.broadcastUpdate("tool_call", map[string]any{
				"task_id": taskID,
				"tool":    tc.ToolName,
			})
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
		if evt.GetToolResult() != nil {
			tr := evt.GetToolResult()
			if !tr.Success && tr.Error != "" {
				em.appendTaskLog(taskID, agentTag, "stderr", tr.Error)
			}
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_ERROR:
		if evt.GetError() != nil {
			em.appendTaskLog(taskID, agentTag, "stderr", evt.GetError().Message)
		}

	case domainpb.RunEventType_RUN_EVENT_TYPE_STATUS:
		if evt.GetStatus() != nil {
			s := evt.GetStatus()
			msg := fmt.Sprintf("Status: %s -> %s", s.OldStatus, s.NewStatus)
			em.appendTaskLog(taskID, agentTag, "stdout", msg)
		}
	}
}

// mapAgentManagerResult converts agent-manager Run result to ClaudeCodeResponse.
func (em *ExecutionManager) mapAgentManagerResult(run *domainpb.Run, task tasks.TaskItem, agentTag, output string, history *ExecutionHistory) *tasks.ClaudeCodeResponse {
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

		if strings.Contains(lowerErr, "rate limit") || strings.Contains(lowerErr, "429") {
			response.RateLimited = true
			response.RetryAfter = DefaultRateLimitRetry
		}

		if strings.Contains(lowerErr, "timeout") {
			response.Error = fmt.Sprintf("TIMEOUT: Task execution exceeded limit. %s", run.ErrorMsg)
		}
	}

	if detectMaxTurnsExceeded(run.ErrorMsg) || detectMaxTurnsExceeded(output) {
		response.MaxTurnsExceeded = true
		response.Success = false
		response.Error = fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", currentSettings.MaxTurns)
		em.appendTaskLog(task.ID, agentTag, "stderr", response.Error)
	}

	if history != nil {
		if run.StartedAt != nil && run.EndedAt != nil {
			duration := run.EndedAt.AsTime().Sub(run.StartedAt.AsTime())
			history.Duration = duration.String()
		}
	}

	return response
}

// handleNonZeroExit handles non-zero exit codes from Claude Code.
func (em *ExecutionManager) handleNonZeroExit(waitErr error, combinedOutput string, task tasks.TaskItem, agentTag string, maxTurns int, elapsed time.Duration) (*tasks.ClaudeCodeResponse, bool) {
	if detectMaxTurnsExceeded(combinedOutput) {
		msg := fmt.Sprintf("Claude reached the configured MAX_TURNS limit (%d). Consider simplifying the task or increasing the limit in Settings.", maxTurns)
		em.appendTaskLog(task.ID, agentTag, "stderr", msg)
		return &tasks.ClaudeCodeResponse{
			Success:          false,
			Error:            msg,
			Output:           combinedOutput,
			MaxTurnsExceeded: true,
		}, true
	}

	detection := ratelimit.DetectFromError(waitErr, combinedOutput, elapsed)
	if detection.IsRateLimited {
		if !detection.CheckWindow || elapsed <= ratelimit.DetectionWindow {
			em.appendTaskLog(task.ID, agentTag, "stderr", fmt.Sprintf("🚫 Rate limit hit. Pausing for %d seconds", detection.RetryAfter))
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

// processExecutionResult handles the result of task execution.
func (em *ExecutionManager) processExecutionResult(result *tasks.ClaudeCodeResponse, task *tasks.TaskItem, history *ExecutionHistory, executionID string, executionStartTime time.Time, executionTime time.Duration, timeoutDuration time.Duration, promptSizeKB float64, autoSteerInitFailed bool, cleanup func()) (*ExecutionResult, error) {
	if result.Success {
		return em.handleSuccessfulExecution(result, task, history, executionID, executionStartTime, executionTime, timeoutDuration, promptSizeKB, autoSteerInitFailed, cleanup)
	}

	if result.RateLimited {
		return em.handleRateLimitedExecution(result, task, history, executionID, executionTime, cleanup)
	}

	return em.handleFailedExecution(result, task, history, executionID, executionStartTime, executionTime, timeoutDuration, cleanup)
}

// handleSuccessfulExecution handles a successful task execution.
func (em *ExecutionManager) handleSuccessfulExecution(result *tasks.ClaudeCodeResponse, task *tasks.TaskItem, history *ExecutionHistory, executionID string, executionStartTime time.Time, executionTime time.Duration, timeoutDuration time.Duration, promptSizeKB float64, autoSteerInitFailed bool, cleanup func()) (*ExecutionResult, error) {
	summary := fmt.Sprintf("Task %s completed successfully in %v (timeout %v)", task.ID, executionTime.Round(time.Second), timeoutDuration)
	log.Println(summary)
	systemlog.Info(summary)

	completedAt := timeutil.NowRFC3339()
	taskResults := tasks.NewSuccessResults(
		result.Message,
		result.Output,
		executionTime.Round(time.Second).String(),
		timeoutDuration.String(),
		executionStartTime.Format(time.RFC3339),
		completedAt,
		fmt.Sprintf("%d chars (%.2f KB)", int(promptSizeKB*KilobytesPerMegabyte), promptSizeKB),
	)
	task.Results = taskResults.ToMap()
	task.CurrentPhase = "completed"
	task.Status = "completed"
	task.CompletedAt = completedAt
	task.CompletionCount++
	task.LastCompletedAt = task.CompletedAt

	// Handle steering continuation logic
	em.handleSteeringContinuation(task, autoSteerInitFailed)

	// Apply cooldown for auto-requeue tasks
	if task.Status == "completed" && task.ProcessorAutoRequeue {
		cooldownSeconds := settings.GetSettings().CooldownSeconds
		if cooldownSeconds > 0 {
			task.CooldownUntil = time.Now().Add(time.Duration(cooldownSeconds) * time.Second).Format(time.RFC3339)
		}
	} else {
		task.CooldownUntil = ""
	}

	history.EndTime = time.Now()
	history.Duration = executionTime.Round(time.Second).String()
	history.Success = true
	history.ExitReason = "completed"

	finalizeStatus := task.Status
	if em.finalizeFunc != nil {
		if err := em.finalizeFunc(task, finalizeStatus); err != nil {
			log.Printf("CRITICAL: Failed to finalize completed task %s: %v", task.ID, err)
			em.cleanupAfterFailure(task.ID, cleanup, "completed task finalization failure")
			return &ExecutionResult{Success: false, Error: err.Error()}, err
		}
	}

	em.saveExecutionArtifacts(task.ID, executionID, history)
	em.broadcastUpdate("task_completed", *task)
	em.completeCleanup(task.ID, cleanup)

	return &ExecutionResult{
		Success: true,
		Output:  result.Output,
		Message: result.Message,
	}, nil
}

// handleRateLimitedExecution handles a rate-limited execution.
func (em *ExecutionManager) handleRateLimitedExecution(result *tasks.ClaudeCodeResponse, task *tasks.TaskItem, history *ExecutionHistory, executionID string, executionTime time.Duration, cleanup func()) (*ExecutionResult, error) {
	log.Printf("🚫 Task %s hit rate limit. Pausing queue for %d seconds", task.ID, result.RetryAfter)

	task.CurrentPhase = "rate_limited"
	task.Status = "pending"
	task.CooldownUntil = ""

	hitAt := timeutil.NowRFC3339()
	taskResults := tasks.NewRateLimitResults(hitAt, result.RetryAfter)
	task.Results = taskResults.ToMap()

	history.EndTime = time.Now()
	history.Duration = executionTime.Round(time.Second).String()
	history.Success = false
	history.ExitReason = "rate_limited"
	history.RateLimited = true
	history.RetryAfter = result.RetryAfter

	if em.finalizeFunc != nil {
		if err := em.finalizeFunc(task, "pending"); err != nil {
			log.Printf("CRITICAL: Failed to finalize rate-limited task %s: %v", task.ID, err)
			em.cleanupAfterFailure(task.ID, cleanup, "rate-limited task finalization failure")
			return &ExecutionResult{Success: false, RateLimited: true, RetryAfter: result.RetryAfter, Error: err.Error()}, err
		}
	}

	em.saveExecutionArtifacts(task.ID, executionID, history)
	em.completeCleanup(task.ID, cleanup)

	em.broadcastUpdate("rate_limit_hit", map[string]any{
		"task_id":     task.ID,
		"retry_after": result.RetryAfter,
		"pause_until": time.Now().Add(time.Duration(result.RetryAfter) * time.Second).Format(time.RFC3339),
	})

	return &ExecutionResult{
		Success:     false,
		RateLimited: true,
		RetryAfter:  result.RetryAfter,
		Output:      result.Output,
	}, nil
}

// handleFailedExecution handles a failed execution.
func (em *ExecutionManager) handleFailedExecution(result *tasks.ClaudeCodeResponse, task *tasks.TaskItem, history *ExecutionHistory, executionID string, executionStartTime time.Time, executionTime time.Duration, timeoutDuration time.Duration, cleanup func()) (*ExecutionResult, error) {
	log.Printf("Task %s failed after %v: %s", task.ID, executionTime.Round(time.Second), result.Error)
	systemlog.Warnf("Task %s failed: %s", task.ID, result.Error)

	var extras map[string]any
	if result.MaxTurnsExceeded {
		extras = map[string]any{"max_turns_exceeded": true}
		log.Printf("⚠️  MAX_TURNS exceeded for task %s - using enhanced cleanup verification", task.ID)
	}

	failResult, finalizeErr := em.handleFailure(task, result.Error, result.Output, executionStartTime, executionTime, timeoutDuration, extras)
	if finalizeErr != nil {
		ctx := "failed task finalization failure"
		if result.MaxTurnsExceeded {
			ctx = "MAX_TURNS finalization failure"
			time.Sleep(scaleDuration(MaxTurnsCleanupDelay))
		}
		em.cleanupAfterFailure(task.ID, cleanup, ctx)
		return failResult, finalizeErr
	}

	em.saveExecutionArtifacts(task.ID, executionID, history)

	if result.MaxTurnsExceeded {
		time.Sleep(scaleDuration(MaxTurnsCleanupDelay))
	}
	em.completeCleanup(task.ID, cleanup)

	return failResult, nil
}

// handleFailure handles task failure with timing information.
func (em *ExecutionManager) handleFailure(task *tasks.TaskItem, errorMsg, output string, startTime time.Time, executionTime, timeoutAllowed time.Duration, extras map[string]any) (*ExecutionResult, error) {
	isTimeout := strings.Contains(errorMsg, "timed out") || strings.Contains(errorMsg, "timeout")

	assembly, _ := em.assembler.AssemblePromptForTask(*task)
	prompt := assembly.Prompt
	promptSizeKB := float64(len(prompt)) / BytesPerKilobyte

	failedAt := timeutil.NowRFC3339()
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
		}
	} else {
		task.CooldownUntil = ""
	}

	if em.finalizeFunc != nil {
		if err := em.finalizeFunc(task, "failed"); err != nil {
			log.Printf("CRITICAL: Failed to finalize failed task %s: %v", task.ID, err)
			return &ExecutionResult{Success: false, Error: errorMsg}, fmt.Errorf("failed to finalize task status: %w", err)
		}
	}

	em.broadcastUpdate("task_failed", *task)
	return &ExecutionResult{Success: false, Error: errorMsg, Output: output}, nil
}

// saveExecutionArtifacts saves execution history artifacts.
func (em *ExecutionManager) saveExecutionArtifacts(taskID, executionID string, history *ExecutionHistory) {
	outputRelPath, err := em.saveOutputToHistory(taskID, executionID)
	if err != nil {
		log.Printf("Warning: Failed to save output to history for task %s execution %s: %v", taskID, executionID, err)
	}
	history.OutputPath = outputRelPath

	if err := em.saveExecutionMetadata(*history); err != nil {
		log.Printf("Warning: Failed to save execution metadata for task %s execution %s: %v", taskID, executionID, err)
	}
}

// completeCleanup handles cleanup after successful finalization.
func (em *ExecutionManager) completeCleanup(taskID string, cleanup func()) {
	cleanup()
	em.registry.UnregisterExecution(taskID)
}

// cleanupAfterFailure handles cleanup when finalization fails.
func (em *ExecutionManager) cleanupAfterFailure(taskID string, cleanup func(), ctx string) {
	if ctx != "" {
		log.Printf("Task %s cleanup (%s)", taskID, ctx)
	}
	cleanup()
	em.registry.UnregisterExecution(taskID)
}

// broadcastUpdate sends updates to WebSocket clients.
func (em *ExecutionManager) broadcastUpdate(updateType string, data any) {
	if em.broadcast == nil {
		return
	}

	select {
	case em.broadcast <- map[string]any{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}:
		log.Printf("Broadcast %s update for task", updateType)
	default:
		log.Printf("Warning: WebSocket broadcast channel full, dropping update")
	}
}

// Task logging methods

func (em *ExecutionManager) initTaskLogBuffer(taskID, agentID string, pid int) {
	if em.taskLogger != nil {
		em.taskLogger.InitBuffer(taskID, agentID, pid)
	}
}

func (em *ExecutionManager) appendTaskLog(taskID, agentID, stream, message string) {
	if em.taskLogger != nil {
		em.taskLogger.Append(taskID, agentID, stream, message)
	}
}

func (em *ExecutionManager) finalizeTaskLogs(taskID string, success bool) {
	if em.taskLogger != nil {
		em.taskLogger.Finalize(taskID, success)
	}
}

// Execution history methods - delegate to HistoryManager when available

func (em *ExecutionManager) savePromptToHistory(taskID, executionID, prompt string) (string, error) {
	if em.historyManager != nil {
		return em.historyManager.SavePromptToHistory(taskID, executionID, prompt)
	}
	// Fallback for when HistoryManager is not available
	execDir := em.GetExecutionDir(taskID, executionID)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return "", fmt.Errorf("create execution directory: %w", err)
	}
	promptPath := filepath.Join(execDir, "prompt.txt")
	if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
		return "", fmt.Errorf("write prompt file: %w", err)
	}
	return filepath.Join(taskID, "executions", executionID, "prompt.txt"), nil
}

func (em *ExecutionManager) saveExecutionMetadata(history ExecutionHistory) error {
	if em.historyManager != nil {
		return em.historyManager.SaveExecutionMetadata(history)
	}
	// Fallback for when HistoryManager is not available
	execDir := em.GetExecutionDir(history.TaskID, history.ExecutionID)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return fmt.Errorf("create execution directory: %w", err)
	}
	metadataPath := filepath.Join(execDir, "metadata.json")
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	return os.WriteFile(metadataPath, data, 0o644)
}

func (em *ExecutionManager) saveOutputToHistory(taskID, executionID string) (string, error) {
	if em.historyManager != nil {
		return em.historyManager.SaveOutputToHistory(taskID, executionID)
	}
	// Fallback for when HistoryManager is not available
	execDir := em.GetExecutionDir(taskID, executionID)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return "", fmt.Errorf("create execution directory: %w", err)
	}
	oldLogPath := filepath.Join(em.taskLogsDir, fmt.Sprintf("%s.log", taskID))
	newLogPath := filepath.Join(execDir, "output.log")
	if _, err := os.Stat(oldLogPath); err == nil {
		data, readErr := os.ReadFile(oldLogPath)
		if readErr == nil {
			if writeErr := os.WriteFile(newLogPath, data, 0o644); writeErr != nil {
				return "", fmt.Errorf("write output to history: %w", writeErr)
			}
			os.Remove(oldLogPath)
		}
	}
	return filepath.Join(taskID, "executions", executionID, "output.log"), nil
}

// LatestExecutionOutputPath returns the path to the most recent execution output.
func (em *ExecutionManager) LatestExecutionOutputPath(taskID string) string {
	if em.historyManager != nil {
		return em.historyManager.LatestExecutionOutputPath(taskID)
	}
	// Fallback for when HistoryManager is not available
	history, err := em.loadExecutionHistory(taskID)
	if err != nil || len(history) == 0 {
		return ""
	}
	latest := history[0]
	if strings.TrimSpace(latest.CleanOutputPath) != "" {
		return filepath.Join(em.taskLogsDir, latest.CleanOutputPath)
	}
	if strings.TrimSpace(latest.OutputPath) != "" {
		return filepath.Join(em.taskLogsDir, latest.OutputPath)
	}
	return ""
}

func (em *ExecutionManager) loadExecutionHistory(taskID string) ([]ExecutionHistory, error) {
	if em.historyManager != nil {
		return em.historyManager.LoadExecutionHistory(taskID)
	}
	// Fallback for when HistoryManager is not available
	historyDir := filepath.Join(em.taskLogsDir, taskID, "executions")
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ExecutionHistory{}, nil
		}
		return nil, fmt.Errorf("read history directory: %w", err)
	}
	var history []ExecutionHistory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metadataPath := filepath.Join(historyDir, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			continue
		}
		var exec ExecutionHistory
		if err := json.Unmarshal(data, &exec); err != nil {
			continue
		}
		history = append(history, exec)
	}
	// Sort by start time (newest first)
	for i := 0; i < len(history)-1; i++ {
		for j := i + 1; j < len(history); j++ {
			if history[j].StartTime.After(history[i].StartTime) {
				history[i], history[j] = history[j], history[i]
			}
		}
	}
	return history, nil
}

// Helper methods for clean outputs

func (em *ExecutionManager) saveCleanOutputs(taskID, executionID, execDir, prompt, combinedOutput, finalMessage, transcriptFile, lastMessageFile string) (string, string, string) {
	makeRel := func(filename string) string {
		return filepath.Join(taskID, "executions", executionID, filename)
	}

	if err := os.MkdirAll(execDir, 0o755); err != nil {
		log.Printf("Warning: unable to ensure execution directory %s: %v", execDir, err)
	}

	var cleanRel, lastRel, transcriptRel string

	if strings.TrimSpace(combinedOutput) != "" {
		cleanPath := filepath.Join(execDir, "clean_output.txt")
		if err := os.WriteFile(cleanPath, []byte(combinedOutput), 0o644); err != nil {
			log.Printf("Warning: failed to persist clean output for %s/%s: %v", taskID, executionID, err)
		} else {
			cleanRel = makeRel("clean_output.txt")
		}
	}

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

	transcriptPath := transcriptFile
	if strings.TrimSpace(transcriptPath) == "" {
		transcriptPath = filepath.Join(execDir, "conversation.jsonl")
	}
	if fileExistsAndNotEmpty(transcriptPath) {
		transcriptRel = makeRel(filepath.Base(transcriptPath))
	} else if err := em.writeFallbackTranscript(transcriptPath, prompt, finalMessage, combinedOutput); err != nil {
		log.Printf("Warning: failed to persist transcript for %s/%s: %v", taskID, executionID, err)
	} else {
		transcriptRel = makeRel(filepath.Base(transcriptPath))
	}

	return cleanRel, lastRel, transcriptRel
}

func (em *ExecutionManager) writeFallbackTranscript(path, prompt, finalMessage, combinedOutput string) error {
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

// Claude config validation

func (em *ExecutionManager) ensureValidClaudeConfig() error {
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

		if err := em.resetClaudeConfig(path, data); err != nil {
			return err
		}
		log.Printf("Detected invalid Claude config at %s; reset to defaults", path)
		validFound = true
	}

	if !validFound {
		path := configPaths[0]
		if err := em.resetClaudeConfig(path, nil); err != nil {
			return err
		}
		log.Printf("Claude config not found; created default config at %s", path)
	}

	return nil
}

func (em *ExecutionManager) resetClaudeConfig(path string, original []byte) error {
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

// Verify interface implementation at compile time
var _ ExecutionManagerAPI = (*ExecutionManager)(nil)
