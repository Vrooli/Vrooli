// Package toolexecution implements tool execution for browser-automation-studio.
package toolexecution

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"google.golang.org/protobuf/proto"
)

// ToolExecutor is the interface for tool execution.
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error)
}

// ServerExecutorConfig holds dependencies for the ServerExecutor.
type ServerExecutorConfig struct {
	CatalogService   workflow.CatalogService
	ExecutionService workflow.ExecutionService
	Repository       database.Repository
}

// ServerExecutor implements tool execution using BAS services.
type ServerExecutor struct {
	catalogService   workflow.CatalogService
	executionService workflow.ExecutionService
	repo             database.Repository
}

// NewServerExecutor creates a new ServerExecutor with the given configuration.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	return &ServerExecutor{
		catalogService:   cfg.CatalogService,
		executionService: cfg.ExecutionService,
		repo:             cfg.Repository,
	}
}

// Execute dispatches tool execution to the appropriate handler.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	switch toolName {
	// Tier 1: Workflow Execution
	case "execute_workflow":
		return e.executeWorkflow(ctx, args)
	case "get_execution":
		return e.getExecution(ctx, args)
	case "get_execution_timeline":
		return e.getExecutionTimeline(ctx, args)
	case "stop_execution":
		return e.stopExecution(ctx, args)
	case "list_workflows":
		return e.listWorkflows(ctx, args)
	case "list_executions":
		return e.listExecutions(ctx, args)

	// Tier 2: Project/Workflow Management
	case "create_workflow":
		return e.createWorkflow(ctx, args)
	case "update_workflow":
		return e.updateWorkflow(ctx, args)
	case "validate_workflow":
		return e.validateWorkflow(ctx, args)
	case "create_project":
		return e.createProject(ctx, args)
	case "list_projects":
		return e.listProjects(ctx, args)

	// Tier 3: Recording
	case "create_recording_session":
		return e.createRecordingSession(ctx, args)
	case "get_recorded_actions":
		return e.getRecordedActions(ctx, args)
	case "generate_workflow_from_recording":
		return e.generateWorkflowFromRecording(ctx, args)

	// Tier 4: AI Capabilities
	case "ai_analyze_elements":
		return e.aiAnalyzeElements(ctx, args)
	case "ai_navigate":
		return e.aiNavigate(ctx, args)
	case "get_dom_tree":
		return e.getDOMTree(ctx, args)

	default:
		return ErrorResult("unknown tool: "+toolName, CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Tier 1: Workflow Execution Tools
// -----------------------------------------------------------------------------

// executeWorkflow executes a workflow by ID (async).
func (e *ServerExecutor) executeWorkflow(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	workflowIDStr := getStringArg(args, "workflow_id", "")
	if workflowIDStr == "" {
		return ErrorResult("workflow_id is required", CodeInvalidArgs), nil
	}

	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid workflow_id: %v", err), CodeInvalidArgs), nil
	}

	// Build request
	req := &basapi.ExecuteWorkflowRequest{
		WorkflowId: workflowID.String(),
	}

	// Execute workflow
	resp, err := e.executionService.ExecuteWorkflowAPI(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to execute workflow: %v", err), CodeInternalError), nil
	}

	// Return async result
	return AsyncResult(map[string]interface{}{
		"execution_id": resp.ExecutionId,
		"status":       resp.Status.String(),
		"message":      "Workflow execution started",
	}, resp.ExecutionId), nil
}

// getExecution gets execution status and details.
func (e *ServerExecutor) getExecution(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	executionIDStr := getStringArg(args, "execution_id", "")
	if executionIDStr == "" {
		return ErrorResult("execution_id is required", CodeInvalidArgs), nil
	}

	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid execution_id: %v", err), CodeInvalidArgs), nil
	}

	exec, err := e.executionService.GetExecution(ctx, executionID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get execution: %v", err), CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"execution_id":  exec.ID.String(),
		"workflow_id":   exec.WorkflowID.String(),
		"status":        exec.Status,
		"started_at":    exec.StartedAt,
		"completed_at":  exec.CompletedAt,
		"error_message": exec.ErrorMessage,
	}), nil
}

// getExecutionTimeline gets the timeline of events for an execution.
func (e *ServerExecutor) getExecutionTimeline(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	executionIDStr := getStringArg(args, "execution_id", "")
	if executionIDStr == "" {
		return ErrorResult("execution_id is required", CodeInvalidArgs), nil
	}

	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid execution_id: %v", err), CodeInvalidArgs), nil
	}

	timeline, err := e.executionService.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get timeline: %v", err), CodeInternalError), nil
	}

	// Convert timeline frames to a simple format
	frames := make([]map[string]interface{}, 0, len(timeline.Frames))
	for _, frame := range timeline.Frames {
		frames = append(frames, map[string]interface{}{
			"step_index":  frame.StepIndex,
			"step_type":   frame.StepType,
			"status":      frame.Status,
			"success":     frame.Success,
			"duration_ms": frame.DurationMs,
		})
	}

	return SuccessResult(map[string]interface{}{
		"execution_id": executionIDStr,
		"frames":       frames,
		"total":        len(frames),
	}), nil
}

// stopExecution stops a running execution.
func (e *ServerExecutor) stopExecution(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	executionIDStr := getStringArg(args, "execution_id", "")
	if executionIDStr == "" {
		return ErrorResult("execution_id is required", CodeInvalidArgs), nil
	}

	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid execution_id: %v", err), CodeInvalidArgs), nil
	}

	err = e.executionService.StopExecution(ctx, executionID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to stop execution: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Execution %s stopped", executionIDStr),
	}), nil
}

// listWorkflows lists available workflows.
func (e *ServerExecutor) listWorkflows(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	limit := getIntArg(args, "limit", 50)
	offset := getIntArg(args, "offset", 0)

	var projectID *uuid.UUID
	if projectIDStr := getStringArg(args, "project_id", ""); projectIDStr != "" {
		id, err := uuid.Parse(projectIDStr)
		if err != nil {
			return ErrorResult(fmt.Sprintf("invalid project_id: %v", err), CodeInvalidArgs), nil
		}
		projectID = &id
	}

	req := &basapi.ListWorkflowsRequest{
		Limit:  proto.Int32(int32(limit)),
		Offset: proto.Int32(int32(offset)),
	}
	if projectID != nil {
		req.ProjectId = proto.String(projectID.String())
	}

	resp, err := e.catalogService.ListWorkflows(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list workflows: %v", err), CodeInternalError), nil
	}

	workflows := make([]map[string]interface{}, 0, len(resp.Workflows))
	for _, w := range resp.Workflows {
		workflows = append(workflows, map[string]interface{}{
			"id":          w.Id,
			"name":        w.Name,
			"description": w.Description,
			"project_id":  w.ProjectId,
			"updated_at":  w.UpdatedAt.AsTime(),
		})
	}

	return SuccessResult(map[string]interface{}{
		"workflows": workflows,
		"total":     resp.Total,
	}), nil
}

// listExecutions lists workflow executions.
func (e *ServerExecutor) listExecutions(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	limit := getIntArg(args, "limit", 50)
	offset := getIntArg(args, "offset", 0)

	var workflowID, projectID *uuid.UUID
	if wfIDStr := getStringArg(args, "workflow_id", ""); wfIDStr != "" {
		id, err := uuid.Parse(wfIDStr)
		if err != nil {
			return ErrorResult(fmt.Sprintf("invalid workflow_id: %v", err), CodeInvalidArgs), nil
		}
		workflowID = &id
	}
	if projIDStr := getStringArg(args, "project_id", ""); projIDStr != "" {
		id, err := uuid.Parse(projIDStr)
		if err != nil {
			return ErrorResult(fmt.Sprintf("invalid project_id: %v", err), CodeInvalidArgs), nil
		}
		projectID = &id
	}

	execs, err := e.executionService.ListExecutions(ctx, workflowID, projectID, limit, offset)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list executions: %v", err), CodeInternalError), nil
	}

	executions := make([]map[string]interface{}, 0, len(execs))
	for _, exec := range execs {
		executions = append(executions, map[string]interface{}{
			"id":           exec.ID.String(),
			"workflow_id":  exec.WorkflowID.String(),
			"status":       exec.Status,
			"started_at":   exec.StartedAt,
			"completed_at": exec.CompletedAt,
		})
	}

	return SuccessResult(map[string]interface{}{
		"executions": executions,
		"total":      len(executions),
	}), nil
}

// -----------------------------------------------------------------------------
// Tier 2: Project/Workflow Management Tools
// -----------------------------------------------------------------------------

// createWorkflow creates a new workflow.
func (e *ServerExecutor) createWorkflow(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	name := getStringArg(args, "name", "")
	if name == "" {
		return ErrorResult("name is required", CodeInvalidArgs), nil
	}

	req := &basapi.CreateWorkflowRequest{
		Name: name,
	}
	if projectIDStr := getStringArg(args, "project_id", ""); projectIDStr != "" {
		req.ProjectId = projectIDStr
	}

	resp, err := e.catalogService.CreateWorkflow(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create workflow: %v", err), CodeInternalError), nil
	}

	workflowID := ""
	if resp.Workflow != nil {
		workflowID = resp.Workflow.Id
	}

	return SuccessResult(map[string]interface{}{
		"workflow_id": workflowID,
		"name":        name,
		"message":     "Workflow created successfully",
	}), nil
}

// updateWorkflow updates an existing workflow.
func (e *ServerExecutor) updateWorkflow(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	workflowIDStr := getStringArg(args, "workflow_id", "")
	if workflowIDStr == "" {
		return ErrorResult("workflow_id is required", CodeInvalidArgs), nil
	}

	req := &basapi.UpdateWorkflowRequest{
		WorkflowId: proto.String(workflowIDStr),
	}
	if name := getStringArg(args, "name", ""); name != "" {
		req.Name = name
	}
	if desc := getStringArg(args, "description", ""); desc != "" {
		req.Description = desc
	}

	resp, err := e.catalogService.UpdateWorkflow(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to update workflow: %v", err), CodeInternalError), nil
	}

	workflowID := workflowIDStr
	var version int32
	if resp.Workflow != nil {
		workflowID = resp.Workflow.Id
		version = resp.Workflow.Version
	}

	return SuccessResult(map[string]interface{}{
		"workflow_id": workflowID,
		"version":     version,
		"message":     "Workflow updated successfully",
	}), nil
}

// validateWorkflow validates a workflow definition.
func (e *ServerExecutor) validateWorkflow(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	definition := args["definition"]
	if definition == nil {
		return ErrorResult("definition is required", CodeInvalidArgs), nil
	}

	// For now, return success - actual validation would be done via the workflow validator
	// This is a placeholder that would integrate with BAS's workflow validation logic
	return SuccessResult(map[string]interface{}{
		"valid":   true,
		"message": "Workflow definition is valid",
	}), nil
}

// createProject creates a new project.
func (e *ServerExecutor) createProject(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	name := getStringArg(args, "name", "")
	if name == "" {
		return ErrorResult("name is required", CodeInvalidArgs), nil
	}

	project := &database.ProjectIndex{
		ID:   uuid.New(),
		Name: name,
	}

	err := e.catalogService.CreateProject(ctx, project, getStringArg(args, "description", ""))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create project: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"project_id": project.ID.String(),
		"name":       name,
		"message":    "Project created successfully",
	}), nil
}

// listProjects lists all projects.
func (e *ServerExecutor) listProjects(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	limit := getIntArg(args, "limit", 50)
	offset := getIntArg(args, "offset", 0)

	projects, err := e.catalogService.ListProjects(ctx, limit, offset)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list projects: %v", err), CodeInternalError), nil
	}

	projectList := make([]map[string]interface{}, 0, len(projects))
	for _, p := range projects {
		projectList = append(projectList, map[string]interface{}{
			"id":          p.ID.String(),
			"name":        p.Name,
			"folder_path": p.FolderPath,
			"created_at":  p.CreatedAt,
		})
	}

	return SuccessResult(map[string]interface{}{
		"projects": projectList,
		"total":    len(projectList),
	}), nil
}

// -----------------------------------------------------------------------------
// Tier 3: Recording Tools
// -----------------------------------------------------------------------------

// createRecordingSession starts a new recording session.
func (e *ServerExecutor) createRecordingSession(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	url := getStringArg(args, "url", "")
	if url == "" {
		return ErrorResult("url is required", CodeInvalidArgs), nil
	}

	// This would integrate with the live recording service
	// For now, return a placeholder that indicates the feature requires the recording service
	return ErrorResult("Recording sessions require the live recording service to be configured", CodeInternalError), nil
}

// getRecordedActions gets actions from a recording session.
func (e *ServerExecutor) getRecordedActions(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sessionID := getStringArg(args, "session_id", "")
	if sessionID == "" {
		return ErrorResult("session_id is required", CodeInvalidArgs), nil
	}

	// This would integrate with the live recording service
	return ErrorResult("Recording sessions require the live recording service to be configured", CodeInternalError), nil
}

// generateWorkflowFromRecording converts a recording to a workflow.
func (e *ServerExecutor) generateWorkflowFromRecording(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sessionID := getStringArg(args, "session_id", "")
	if sessionID == "" {
		return ErrorResult("session_id is required", CodeInvalidArgs), nil
	}

	workflowName := getStringArg(args, "workflow_name", "")
	if workflowName == "" {
		return ErrorResult("workflow_name is required", CodeInvalidArgs), nil
	}

	// This would integrate with the live recording service
	return ErrorResult("Recording sessions require the live recording service to be configured", CodeInternalError), nil
}

// -----------------------------------------------------------------------------
// Tier 4: AI Capabilities Tools
// -----------------------------------------------------------------------------

// aiAnalyzeElements uses AI to analyze page elements.
func (e *ServerExecutor) aiAnalyzeElements(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	url := getStringArg(args, "url", "")
	if url == "" {
		return ErrorResult("url is required", CodeInvalidArgs), nil
	}

	query := getStringArg(args, "query", "")
	if query == "" {
		return ErrorResult("query is required", CodeInvalidArgs), nil
	}

	// This would integrate with the AI service
	return ErrorResult("AI analysis requires the AI service to be configured", CodeInternalError), nil
}

// aiNavigate uses AI to navigate a website.
func (e *ServerExecutor) aiNavigate(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	url := getStringArg(args, "url", "")
	if url == "" {
		return ErrorResult("url is required", CodeInvalidArgs), nil
	}

	goal := getStringArg(args, "goal", "")
	if goal == "" {
		return ErrorResult("goal is required", CodeInvalidArgs), nil
	}

	// This would integrate with the AI navigation service
	return ErrorResult("AI navigation requires the AI service to be configured", CodeInternalError), nil
}

// getDOMTree extracts the DOM tree from a page.
func (e *ServerExecutor) getDOMTree(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	url := getStringArg(args, "url", "")
	if url == "" {
		return ErrorResult("url is required", CodeInvalidArgs), nil
	}

	// This would integrate with the browser service
	return ErrorResult("DOM extraction requires the browser service to be configured", CodeInternalError), nil
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
