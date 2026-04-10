// Package toolexecution implements the Tool Execution Protocol for app-issue-tracker.
package toolexecution

import (
	"context"
	"fmt"
	"time"
)

// ToolExecutor executes tools from the Tool Execution Protocol.
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error)
}

// IssueOperations defines the interface for issue-related operations.
type IssueOperations interface {
	// Issue Discovery
	ListIssues(status, priority, issueType, targetID, targetType string, limit int) ([]interface{}, error)
	GetIssue(issueID string) (interface{}, error)
	SearchIssues(query string, limit int) ([]interface{}, error)
	GetStatistics() (map[string]interface{}, error)
	ListApplications() ([]interface{}, error)

	// Issue Management
	CreateIssue(req interface{}) (interface{}, string, error)
	UpdateIssue(issueID string, updates map[string]interface{}) (interface{}, error)
	TransitionIssue(issueID, status, reason string) (interface{}, error)
	AddArtifacts(issueID string, artifacts []interface{}) error

	// Investigation
	TriggerInvestigation(issueID, agentID string, autoResolve bool) (string, error)
	GetInvestigationStatus(issueID string) (map[string]interface{}, error)
	PreviewInvestigationPrompt(issueID, agentID string) (string, error)
	GetAgentConversation(issueID string, maxEvents int) (interface{}, error)
	ListRunningInvestigations() ([]interface{}, error)
	StopInvestigation(issueID string) (bool, error)
	IsIssueRunning(issueID string) bool

	// Agent/Automation
	ListAgents() ([]interface{}, error)
	GetAgentSettings() (map[string]interface{}, error)
	UpdateAgentSettings(settings map[string]interface{}) error
	GetProcessorState() (map[string]interface{}, error)

	// Reporting
	ExportIssues(format, status, priority, issueType, targetID string) (interface{}, error)
	AnalyzeFailures(since *time.Time, targetID string) (map[string]interface{}, error)
}

// ServerExecutor implements ToolExecutor using the Server's operations.
type ServerExecutor struct {
	ops    IssueOperations
	logger func(msg string, fields map[string]interface{})
}

// NewServerExecutor creates a new ServerExecutor.
func NewServerExecutor(ops IssueOperations, logger func(msg string, fields map[string]interface{})) *ServerExecutor {
	if logger == nil {
		logger = func(msg string, fields map[string]interface{}) {}
	}
	return &ServerExecutor{
		ops:    ops,
		logger: logger,
	}
}

// Execute dispatches tool execution to the appropriate handler method.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	switch toolName {
	// Issue Discovery (5 tools)
	case "list_issues":
		return e.listIssues(ctx, args)
	case "get_issue":
		return e.getIssue(ctx, args)
	case "search_issues":
		return e.searchIssues(ctx, args)
	case "get_issue_statistics":
		return e.getIssueStatistics(ctx, args)
	case "list_applications":
		return e.listApplications(ctx, args)

	// Issue Management (4 tools)
	case "create_issue":
		return e.createIssue(ctx, args)
	case "update_issue":
		return e.updateIssue(ctx, args)
	case "transition_issue":
		return e.transitionIssue(ctx, args)
	case "add_issue_artifacts":
		return e.addIssueArtifacts(ctx, args)

	// Investigation (6 tools)
	case "investigate_issue":
		return e.investigateIssue(ctx, args)
	case "get_investigation_status":
		return e.getInvestigationStatus(ctx, args)
	case "preview_investigation_prompt":
		return e.previewInvestigationPrompt(ctx, args)
	case "get_agent_conversation":
		return e.getAgentConversation(ctx, args)
	case "list_running_investigations":
		return e.listRunningInvestigations(ctx, args)
	case "stop_investigation":
		return e.stopInvestigation(ctx, args)

	// Agent/Automation (4 tools)
	case "list_agents":
		return e.listAgents(ctx, args)
	case "get_agent_settings":
		return e.getAgentSettings(ctx, args)
	case "update_agent_settings":
		return e.updateAgentSettings(ctx, args)
	case "get_processor_state":
		return e.getProcessorState(ctx, args)

	// Reporting (2 tools)
	case "export_issues":
		return e.exportIssues(ctx, args)
	case "analyze_failures":
		return e.analyzeFailures(ctx, args)

	default:
		return ErrorResult(fmt.Sprintf("unknown tool: %s", toolName), CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Issue Discovery Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) listIssues(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	status := getStringArg(args, "status", "")
	priority := getStringArg(args, "priority", "")
	issueType := getStringArg(args, "type", "")
	targetID := getStringArg(args, "target_id", "")
	targetType := getStringArg(args, "target_type", "")
	limit := getIntArg(args, "limit", 50)

	issues, err := e.ops.ListIssues(status, priority, issueType, targetID, targetType, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list issues: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"issues":    issues,
		"count":     len(issues),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

func (e *ServerExecutor) getIssue(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	issue, err := e.ops.GetIssue(issueID)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get issue: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"issue": issue,
	}), nil
}

func (e *ServerExecutor) searchIssues(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	query := getStringArg(args, "query", "")
	if query == "" {
		return ErrorResult("query is required", CodeInvalidArgs), nil
	}
	limit := getIntArg(args, "limit", 20)

	issues, err := e.ops.SearchIssues(query, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to search issues: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"issues":    issues,
		"count":     len(issues),
		"query":     query,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

func (e *ServerExecutor) getIssueStatistics(_ context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	stats, err := e.ops.GetStatistics()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get statistics: %v", err), CodeInternalError), nil
	}

	return SuccessResult(stats), nil
}

func (e *ServerExecutor) listApplications(_ context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	apps, err := e.ops.ListApplications()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list applications: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"applications": apps,
		"count":        len(apps),
	}), nil
}

// -----------------------------------------------------------------------------
// Issue Management Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) createIssue(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	title := getStringArg(args, "title", "")
	if title == "" {
		return ErrorResult("title is required", CodeInvalidArgs), nil
	}

	issue, storagePath, err := e.ops.CreateIssue(args)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create issue: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"issue":        issue,
		"storage_path": storagePath,
		"message":      "Issue created successfully",
	}), nil
}

func (e *ServerExecutor) updateIssue(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	// Extract update fields (exclude issue_id from updates)
	updates := make(map[string]interface{})
	for k, v := range args {
		if k != "issue_id" {
			updates[k] = v
		}
	}

	issue, err := e.ops.UpdateIssue(issueID, updates)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to update issue: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"issue":   issue,
		"message": "Issue updated successfully",
	}), nil
}

func (e *ServerExecutor) transitionIssue(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	status := getStringArg(args, "status", "")
	if status == "" {
		return ErrorResult("status is required", CodeInvalidArgs), nil
	}

	reason := getStringArg(args, "reason", "")

	issue, err := e.ops.TransitionIssue(issueID, status, reason)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to transition issue: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"issue":   issue,
		"message": fmt.Sprintf("Issue transitioned to %s", status),
	}), nil
}

func (e *ServerExecutor) addIssueArtifacts(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	artifactsRaw, ok := args["artifacts"]
	if !ok {
		return ErrorResult("artifacts is required", CodeInvalidArgs), nil
	}

	artifacts, ok := artifactsRaw.([]interface{})
	if !ok {
		return ErrorResult("artifacts must be an array", CodeInvalidArgs), nil
	}

	if err := e.ops.AddArtifacts(issueID, artifacts); err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to add artifacts: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"issue_id":        issueID,
		"artifacts_added": len(artifacts),
		"message":         "Artifacts added successfully",
	}), nil
}

// -----------------------------------------------------------------------------
// Investigation Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) investigateIssue(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	agentID := getStringArg(args, "agent_id", "unified-resolver")
	autoResolve := getBoolArg(args, "auto_resolve", true)
	force := getBoolArg(args, "force", false)

	// Check if already running
	if !force && e.ops.IsIssueRunning(issueID) {
		return ErrorResult("investigation already running for this issue", CodeConflict), nil
	}

	runID, err := e.ops.TriggerInvestigation(issueID, agentID, autoResolve)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to trigger investigation: %v", err), CodeInternalError), nil
	}

	return AsyncResult(map[string]interface{}{
		"issue_id":     issueID,
		"agent_id":     agentID,
		"run_id":       runID,
		"status":       StatusActive,
		"auto_resolve": autoResolve,
		"message":      "Investigation started",
	}, runID), nil
}

func (e *ServerExecutor) getInvestigationStatus(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	status, err := e.ops.GetInvestigationStatus(issueID)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get investigation status: %v", err), CodeInternalError), nil
	}

	return SuccessResult(status), nil
}

func (e *ServerExecutor) previewInvestigationPrompt(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	agentID := getStringArg(args, "agent_id", "unified-resolver")

	prompt, err := e.ops.PreviewInvestigationPrompt(issueID, agentID)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to preview prompt: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"prompt":       prompt,
		"issue_id":     issueID,
		"agent_id":     agentID,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

func (e *ServerExecutor) getAgentConversation(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	maxEvents := getIntArg(args, "max_events", 100)

	conversation, err := e.ops.GetAgentConversation(issueID, maxEvents)
	if err != nil {
		if isNotFoundError(err) {
			return ErrorResult("issue not found", CodeNotFound), nil
		}
		return ErrorResult(fmt.Sprintf("failed to get conversation: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"conversation": conversation,
		"issue_id":     issueID,
	}), nil
}

func (e *ServerExecutor) listRunningInvestigations(_ context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	investigations, err := e.ops.ListRunningInvestigations()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list running investigations: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"investigations": investigations,
		"count":          len(investigations),
	}), nil
}

func (e *ServerExecutor) stopInvestigation(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	issueID := getStringArg(args, "issue_id", "")
	if issueID == "" {
		return ErrorResult("issue_id is required", CodeInvalidArgs), nil
	}

	stopped, err := e.ops.StopInvestigation(issueID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to stop investigation: %v", err), CodeInternalError), nil
	}

	if !stopped {
		return ErrorResult("no running investigation found for this issue", CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"issue_id": issueID,
		"stopped":  true,
		"message":  "Investigation stop requested",
	}), nil
}

// -----------------------------------------------------------------------------
// Agent/Automation Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) listAgents(_ context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	agents, err := e.ops.ListAgents()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list agents: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	}), nil
}

func (e *ServerExecutor) getAgentSettings(_ context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	settings, err := e.ops.GetAgentSettings()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get agent settings: %v", err), CodeInternalError), nil
	}

	return SuccessResult(settings), nil
}

func (e *ServerExecutor) updateAgentSettings(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	if err := e.ops.UpdateAgentSettings(args); err != nil {
		return ErrorResult(fmt.Sprintf("failed to update agent settings: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"message": "Agent settings updated successfully",
	}), nil
}

func (e *ServerExecutor) getProcessorState(_ context.Context, _ map[string]interface{}) (*ExecutionResult, error) {
	state, err := e.ops.GetProcessorState()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get processor state: %v", err), CodeInternalError), nil
	}

	return SuccessResult(state), nil
}

// -----------------------------------------------------------------------------
// Reporting Tools
// -----------------------------------------------------------------------------

func (e *ServerExecutor) exportIssues(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	format := getStringArg(args, "format", "json")
	status := getStringArg(args, "status", "")
	priority := getStringArg(args, "priority", "")
	issueType := getStringArg(args, "type", "")
	targetID := getStringArg(args, "target_id", "")

	data, err := e.ops.ExportIssues(format, status, priority, issueType, targetID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to export issues: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"format":      format,
		"data":        data,
		"exported_at": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

func (e *ServerExecutor) analyzeFailures(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	var since *time.Time
	if sinceStr := getStringArg(args, "since", ""); sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			since = &t
		}
	}
	targetID := getStringArg(args, "target_id", "")

	analysis, err := e.ops.AnalyzeFailures(since, targetID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to analyze failures: %v", err), CodeInternalError), nil
	}

	return SuccessResult(analysis), nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return defaultValue
}

func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	switch v := args[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultValue
}

func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

func isNotFoundError(err error) bool {
	// Check for common "not found" error patterns
	errStr := err.Error()
	return contains(errStr, "not found") || contains(errStr, "does not exist")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
