package toolexecution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
)

// ServerExecutorConfig holds dependencies for the ServerExecutor.
type ServerExecutorConfig struct {
	Orchestrator orchestration.Service
}

// ServerExecutor implements tool execution using the orchestration service.
type ServerExecutor struct {
	orchestrator orchestration.Service
}

// NewServerExecutor creates a new ServerExecutor with the given configuration.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	return &ServerExecutor{
		orchestrator: cfg.Orchestrator,
	}
}

// Execute dispatches tool execution to the appropriate handler.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	switch toolName {
	case "spawn_coding_agent":
		return e.spawnCodingAgent(ctx, args)
	case "check_agent_status":
		return e.checkAgentStatus(ctx, args)
	case "stop_agent":
		return e.stopAgent(ctx, args)
	case "list_active_agents":
		return e.listActiveAgents(ctx, args)
	case "get_agent_diff":
		return e.getAgentDiff(ctx, args)
	case "approve_agent_changes":
		return e.approveAgentChanges(ctx, args)
	default:
		return ErrorResult("unknown tool: "+toolName, CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Tool Implementations
// -----------------------------------------------------------------------------

// spawnCodingAgent creates a new coding agent run.
// This is the primary tool for spawning agents - it creates a Task and then a Run.
func (e *ServerExecutor) spawnCodingAgent(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	// Parse required arguments
	taskDescription := getStringArg(args, "task", "")
	if taskDescription == "" {
		return ErrorResult("task is required", CodeInvalidArgs), nil
	}

	// Parse optional arguments with defaults
	runnerType := getStringArg(args, "runner_type", "claude-code")
	workspacePath := getStringArg(args, "workspace_path", "")
	if workspacePath == "" {
		workspacePath = os.Getenv("VROOLI_ROOT")
		if workspacePath == "" {
			workspacePath = os.Getenv("HOME") + "/Vrooli"
		}
	}
	timeoutMinutes := getIntArg(args, "timeout_minutes", 30)

	// Extract context attachments (skill injection from agent-inbox)
	var contextAttachments []domain.ContextAttachment
	if attachments, ok := args["_context_attachments"].([]interface{}); ok {
		for _, a := range attachments {
			if attMap, ok := a.(map[string]interface{}); ok {
				attachment := domain.ContextAttachment{
					Type:    getStringArg(attMap, "type", "note"),
					Key:     getStringArg(attMap, "key", ""),
					Label:   getStringArg(attMap, "label", ""),
					Content: getStringArg(attMap, "content", ""),
					Format:  getStringArg(attMap, "format", ""),
				}
				// Extract tags if present
				if tags, ok := attMap["tags"].([]interface{}); ok {
					for _, t := range tags {
						if tag, ok := t.(string); ok {
							attachment.Tags = append(attachment.Tags, tag)
						}
					}
				}
				contextAttachments = append(contextAttachments, attachment)
			}
		}
	}

	// Step 1: Create the Task
	task := &domain.Task{
		Title:              "Chat-initiated task",
		Description:        taskDescription,
		ScopePath:          "agent-inbox/chat-tasks",
		ProjectRoot:        workspacePath,
		Status:             domain.TaskStatusQueued,
		ContextAttachments: contextAttachments,
	}

	createdTask, err := e.orchestrator.CreateTask(ctx, task)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create task: %v", err), CodeInternalError), nil
	}

	// Step 2: Create the Run
	rt := mapRunnerType(runnerType)
	timeout := time.Duration(timeoutMinutes) * time.Minute
	requiresApproval := true

	runReq := orchestration.CreateRunRequest{
		TaskID:           createdTask.ID,
		RunnerType:       &rt,
		Timeout:          &timeout,
		RequiresApproval: &requiresApproval,
	}

	run, err := e.orchestrator.CreateRun(ctx, runReq)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create run: %v", err), CodeInternalError), nil
	}

	// Return async result with run_id for polling
	return AsyncResult(map[string]interface{}{
		"success": true,
		"run_id":  run.ID.String(),
		"task_id": createdTask.ID.String(),
		"status":  string(run.Status),
		"message": fmt.Sprintf("Coding agent spawned successfully. Run ID: %s, Task ID: %s", run.ID, createdTask.ID),
	}, run.ID.String()), nil
}

// checkAgentStatus gets the status of a coding agent run.
func (e *ServerExecutor) checkAgentStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	runIDStr := getStringArg(args, "run_id", "")
	if runIDStr == "" {
		return ErrorResult("run_id is required", CodeInvalidArgs), nil
	}

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid run_id: %v", err), CodeInvalidArgs), nil
	}

	run, err := e.orchestrator.GetRun(ctx, runID)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult(fmt.Sprintf("run not found: %s", runIDStr), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get run: %v", err), CodeInternalError), nil
	}

	// Build run data response
	runData := map[string]interface{}{
		"id":         run.ID.String(),
		"task_id":    run.TaskID.String(),
		"status":     string(run.Status),
		"phase":      string(run.Phase),
		"created_at": run.CreatedAt,
		"updated_at": run.UpdatedAt,
	}
	// Add runner type if resolved config is available
	if run.ResolvedConfig != nil {
		runData["runner_type"] = string(run.ResolvedConfig.RunnerType)
	}

	return SuccessResult(map[string]interface{}{
		"run": runData,
	}), nil
}

// stopAgent stops a running coding agent.
func (e *ServerExecutor) stopAgent(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	runIDStr := getStringArg(args, "run_id", "")
	if runIDStr == "" {
		return ErrorResult("run_id is required", CodeInvalidArgs), nil
	}

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid run_id: %v", err), CodeInvalidArgs), nil
	}

	err = e.orchestrator.StopRun(ctx, runID)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult(fmt.Sprintf("run not found: %s", runIDStr), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to stop run: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent run %s stopped", runIDStr),
	}), nil
}

// listActiveAgents returns a list of active (running) agent runs.
func (e *ServerExecutor) listActiveAgents(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	status := domain.RunStatusRunning
	runs, err := e.orchestrator.ListRuns(ctx, orchestration.RunListOptions{
		Status: &status,
	})
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list runs: %v", err), CodeInternalError), nil
	}

	// Convert runs to response format
	runList := make([]map[string]interface{}, 0, len(runs))
	for _, run := range runs {
		runData := map[string]interface{}{
			"id":         run.ID.String(),
			"task_id":    run.TaskID.String(),
			"status":     string(run.Status),
			"phase":      string(run.Phase),
			"created_at": run.CreatedAt,
		}
		if run.ResolvedConfig != nil {
			runData["runner_type"] = string(run.ResolvedConfig.RunnerType)
		}
		runList = append(runList, runData)
	}

	return SuccessResult(map[string]interface{}{
		"runs": runList,
	}), nil
}

// getAgentDiff gets the code changes from an agent run.
func (e *ServerExecutor) getAgentDiff(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	runIDStr := getStringArg(args, "run_id", "")
	if runIDStr == "" {
		return ErrorResult("run_id is required", CodeInvalidArgs), nil
	}

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid run_id: %v", err), CodeInvalidArgs), nil
	}

	diff, err := e.orchestrator.GetRunDiff(ctx, runID)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult(fmt.Sprintf("run not found: %s", runIDStr), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get diff: %v", err), CodeInternalError), nil
	}

	// Convert diff to response format
	result := map[string]interface{}{
		"run_id": runIDStr,
	}
	if diff != nil {
		result["unified_diff"] = diff.UnifiedDiff
		result["files"] = diff.Files
		result["stats"] = diff.Stats
		result["sandbox_id"] = diff.SandboxID.String()
		result["generated"] = diff.Generated
	}

	return SuccessResult(result), nil
}

// approveAgentChanges approves and applies changes from an agent run.
func (e *ServerExecutor) approveAgentChanges(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	runIDStr := getStringArg(args, "run_id", "")
	if runIDStr == "" {
		return ErrorResult("run_id is required", CodeInvalidArgs), nil
	}

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid run_id: %v", err), CodeInvalidArgs), nil
	}

	approveResult, err := e.orchestrator.ApproveRun(ctx, orchestration.ApproveRequest{
		RunID: runID,
		Actor: "agent-inbox",
	})
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult(fmt.Sprintf("run not found: %s", runIDStr), CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to approve: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"success":     approveResult.Success,
		"message":     fmt.Sprintf("Changes from run %s approved and applied", runIDStr),
		"applied":     approveResult.Applied,
		"remaining":   approveResult.Remaining,
		"is_partial":  approveResult.IsPartial,
		"commit_hash": approveResult.CommitHash,
	}), nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// getStringArg extracts a string argument with a default value.
func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultValue
}

// getIntArg extracts an int argument with a default value.
// Handles both int and float64 (JSON numbers decode as float64).
func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	if v, ok := args[key].(int); ok {
		return v
	}
	return defaultValue
}

// mapRunnerType converts string runner type to domain.RunnerType.
func mapRunnerType(rt string) domain.RunnerType {
	switch rt {
	case "claude-code":
		return domain.RunnerTypeClaudeCode
	case "codex":
		return domain.RunnerTypeCodex
	case "opencode":
		return domain.RunnerTypeOpenCode
	default:
		return domain.RunnerTypeClaudeCode
	}
}

// isNotFoundError checks if an error is a not-found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for domain NotFoundError
	var notFoundErr *domain.NotFoundError
	if errors.As(err, &notFoundErr) {
		return true
	}
	// Also check error code
	errCode := domain.GetErrorCode(err)
	return errCode.Category() == "NOT_FOUND"
}
