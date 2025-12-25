package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"agent-inbox/domain"
)

// ToolHandler defines how a single tool processes its arguments.
// Each tool implements this to handle its specific logic.
type ToolHandler func(ctx context.Context, executor *ToolExecutor, args map[string]interface{}) (interface{}, error)

// ToolExecutor handles execution of tool calls from AI responses.
// It routes tool calls to the appropriate handler and tracks execution.
type ToolExecutor struct {
	agentManager *AgentManagerClient
	registry     map[string]toolRegistration
}

// toolRegistration contains metadata about a registered tool.
type toolRegistration struct {
	handler      ToolHandler
	scenarioName string
}

// NewToolExecutor creates a new tool executor with all tools registered.
func NewToolExecutor() *ToolExecutor {
	e := &ToolExecutor{
		registry: make(map[string]toolRegistration),
	}
	e.registerAgentManagerTools()
	return e
}

// registerAgentManagerTools registers all agent-manager integration tools.
func (e *ToolExecutor) registerAgentManagerTools() {
	e.registerTool("spawn_coding_agent", "agent-manager", e.spawnCodingAgent)
	e.registerTool("check_agent_status", "agent-manager", e.checkAgentStatus)
	e.registerTool("stop_agent", "agent-manager", e.stopAgent)
	e.registerTool("list_active_agents", "agent-manager", e.listActiveAgents)
	e.registerTool("get_agent_diff", "agent-manager", e.getAgentDiff)
	e.registerTool("approve_agent_changes", "agent-manager", e.approveAgentChanges)
}

// registerTool adds a tool to the registry.
func (e *ToolExecutor) registerTool(name, scenario string, handler ToolHandler) {
	e.registry[name] = toolRegistration{
		handler:      handler,
		scenarioName: scenario,
	}
}

// IsKnownTool checks if a tool name is registered.
func (e *ToolExecutor) IsKnownTool(toolName string) bool {
	_, exists := e.registry[toolName]
	return exists
}

// GetToolScenario returns the scenario that provides a tool.
func (e *ToolExecutor) GetToolScenario(toolName string) string {
	if reg, exists := e.registry[toolName]; exists {
		return reg.scenarioName
	}
	return ""
}

// ExecuteTool runs a tool and returns the result record.
// Decision: Route to handler based on tool name
// This is the central dispatch point for all tool execution.
func (e *ToolExecutor) ExecuteTool(ctx context.Context, chatID, toolCallID, toolName, arguments string) (*domain.ToolCallRecord, error) {
	record := e.initToolCallRecord(chatID, toolCallID, toolName, arguments)

	// Parse arguments
	args, err := e.parseArguments(arguments)
	if err != nil {
		return e.failRecord(record, err), err
	}

	// Route to handler
	result, err := e.routeToHandler(ctx, toolName, args, record)
	if err != nil {
		return e.failRecord(record, err), err
	}

	return e.completeRecord(record, result), nil
}

// initToolCallRecord creates a new record for tracking execution.
func (e *ToolExecutor) initToolCallRecord(chatID, toolCallID, toolName, arguments string) *domain.ToolCallRecord {
	record := &domain.ToolCallRecord{
		ID:        toolCallID,
		ChatID:    chatID,
		ToolName:  toolName,
		Arguments: arguments,
		Status:    domain.StatusRunning,
		StartedAt: time.Now(),
	}

	// Set scenario name if known
	if scenario := e.GetToolScenario(toolName); scenario != "" {
		record.ScenarioName = scenario
	}

	return record
}

// parseArguments parses JSON arguments into a map.
func (e *ToolExecutor) parseArguments(arguments string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	return args, nil
}

// routeToHandler dispatches to the appropriate tool handler.
// Decision boundary: Which handler processes this tool call?
func (e *ToolExecutor) routeToHandler(ctx context.Context, toolName string, args map[string]interface{}, record *domain.ToolCallRecord) (interface{}, error) {
	reg, exists := e.registry[toolName]
	if !exists {
		return nil, &UnknownToolError{ToolName: toolName}
	}

	result, err := reg.handler(ctx, e, args)

	// Extract external run ID if present (for agent-manager tools)
	if res, ok := result.(map[string]interface{}); ok {
		if runID, ok := res["run_id"].(string); ok {
			record.ExternalRunID = runID
		}
	}

	return result, err
}

// failRecord marks a record as failed and returns it.
func (e *ToolExecutor) failRecord(record *domain.ToolCallRecord, err error) *domain.ToolCallRecord {
	record.Status = domain.StatusFailed
	record.ErrorMessage = err.Error()
	record.CompletedAt = time.Now()
	resultJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
	record.Result = string(resultJSON)
	return record
}

// completeRecord marks a record as completed and returns it.
func (e *ToolExecutor) completeRecord(record *domain.ToolCallRecord, result interface{}) *domain.ToolCallRecord {
	record.Status = domain.StatusCompleted
	record.CompletedAt = time.Now()
	resultJSON, _ := json.Marshal(result)
	record.Result = string(resultJSON)
	return record
}

// UnknownToolError indicates a tool name that isn't registered.
type UnknownToolError struct {
	ToolName string
}

func (e *UnknownToolError) Error() string {
	return fmt.Sprintf("unknown tool: %s", e.ToolName)
}

// getClient returns or creates the agent manager client.
func (e *ToolExecutor) getClient() (*AgentManagerClient, error) {
	if e.agentManager == nil {
		client, err := NewAgentManagerClient()
		if err != nil {
			return nil, err
		}
		e.agentManager = client
	}
	return e.agentManager, nil
}

// Tool handler implementations

func (e *ToolExecutor) spawnCodingAgent(ctx context.Context, _ *ToolExecutor, args map[string]interface{}) (interface{}, error) {
	client, err := e.getClient()
	if err != nil {
		return nil, err
	}

	task, _ := args["task"].(string)
	if task == "" {
		return nil, fmt.Errorf("task is required")
	}

	runnerType := "claude-code"
	if rt, ok := args["runner_type"].(string); ok && rt != "" {
		runnerType = rt
	}

	workspacePath := os.Getenv("VROOLI_ROOT")
	if workspacePath == "" {
		workspacePath = os.Getenv("HOME") + "/Vrooli"
	}
	if wp, ok := args["workspace_path"].(string); ok && wp != "" {
		workspacePath = wp
	}

	timeoutMinutes := 30
	if tm, ok := args["timeout_minutes"].(float64); ok {
		timeoutMinutes = int(tm)
	}

	return client.SpawnCodingAgent(ctx, task, runnerType, workspacePath, timeoutMinutes)
}

func (e *ToolExecutor) checkAgentStatus(ctx context.Context, _ *ToolExecutor, args map[string]interface{}) (interface{}, error) {
	client, err := e.getClient()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	return client.CheckAgentStatus(ctx, runID)
}

func (e *ToolExecutor) stopAgent(ctx context.Context, _ *ToolExecutor, args map[string]interface{}) (interface{}, error) {
	client, err := e.getClient()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	return client.StopAgent(ctx, runID)
}

func (e *ToolExecutor) listActiveAgents(ctx context.Context, _ *ToolExecutor, _ map[string]interface{}) (interface{}, error) {
	client, err := e.getClient()
	if err != nil {
		return nil, err
	}

	return client.ListActiveAgents(ctx)
}

func (e *ToolExecutor) getAgentDiff(ctx context.Context, _ *ToolExecutor, args map[string]interface{}) (interface{}, error) {
	client, err := e.getClient()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	return client.GetAgentDiff(ctx, runID)
}

func (e *ToolExecutor) approveAgentChanges(ctx context.Context, _ *ToolExecutor, args map[string]interface{}) (interface{}, error) {
	client, err := e.getClient()
	if err != nil {
		return nil, err
	}

	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	return client.ApproveAgentChanges(ctx, runID)
}
