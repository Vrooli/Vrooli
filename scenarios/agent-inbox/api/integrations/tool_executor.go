// Package integrations provides clients for external services.
//
// This file implements the ToolExecutor for executing tool calls from AI responses.
// The executor routes tool calls to the appropriate scenario handler and tracks execution.
//
// ARCHITECTURE:
// - ToolExecutor: Central dispatcher for tool execution
// - ScenarioHandler: Interface for scenario-specific tool handling
// - Uses ToolRegistry for tool metadata lookup
// - All scenarios use the Tool Execution Protocol via ProtocolHandler
//
// TESTING SEAMS:
// - ScenarioHandler interface for mocking scenario implementations
// - ToolRegistry can be injected for testing
package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"agent-inbox/domain"
)

// ScenarioHandler defines the interface for executing tools from a specific scenario.
// Each scenario that provides tools implements this interface.
type ScenarioHandler interface {
	// Scenario returns the name of the scenario this handler serves.
	Scenario() string

	// CanHandle checks if this handler can execute the given tool.
	CanHandle(toolName string) bool

	// Execute runs the tool and returns the result.
	Execute(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error)
}

// ToolExecutor handles execution of tool calls from AI responses.
// It routes tool calls to the appropriate scenario handler and tracks execution.
type ToolExecutor struct {
	mu       sync.RWMutex
	handlers map[string]ScenarioHandler // scenario name -> handler

	// Legacy support: direct tool handlers for backward compatibility
	legacyHandlers map[string]legacyToolHandler
}

// legacyToolHandler is the old-style handler function signature.
// Kept for backward compatibility during migration.
type legacyToolHandler struct {
	handler      func(ctx context.Context, args map[string]interface{}) (interface{}, error)
	scenarioName string
}

// NewToolExecutor creates a new tool executor.
// Scenario handlers are registered automatically by the ToolRegistry when tools are refreshed.
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		handlers:       make(map[string]ScenarioHandler),
		legacyHandlers: make(map[string]legacyToolHandler),
	}
}

// NewToolExecutorWithHandlers creates a ToolExecutor with custom handlers.
// This is the constructor for testing.
func NewToolExecutorWithHandlers(handlers ...ScenarioHandler) *ToolExecutor {
	e := &ToolExecutor{
		handlers:       make(map[string]ScenarioHandler),
		legacyHandlers: make(map[string]legacyToolHandler),
	}

	for _, h := range handlers {
		e.RegisterHandler(h)
	}

	return e
}

// RegisterHandler adds a scenario handler to the executor.
func (e *ToolExecutor) RegisterHandler(handler ScenarioHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[handler.Scenario()] = handler
}

// UnregisterHandler removes a scenario handler from the executor.
func (e *ToolExecutor) UnregisterHandler(scenario string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.handlers, scenario)
}

// RegisterProtocolHandler creates and registers a ProtocolHandler for a scenario.
// This is a convenience method for scenarios that implement the Tool Execution Protocol.
// Parameters:
//   - scenarioName: The name of the scenario (e.g., "scenario-to-cloud")
//   - baseURL: The base URL of the scenario's API (e.g., "http://localhost:8080")
//   - toolNames: List of tool names that this scenario provides
func (e *ToolExecutor) RegisterProtocolHandler(scenarioName, baseURL string, toolNames []string) {
	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: scenarioName,
		BaseURL:      baseURL,
		ToolNames:    toolNames,
	})
	e.RegisterHandler(handler)
}

// IsKnownTool checks if any handler can execute this tool.
func (e *ToolExecutor) IsKnownTool(toolName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, handler := range e.handlers {
		if handler.CanHandle(toolName) {
			return true
		}
	}

	_, exists := e.legacyHandlers[toolName]
	return exists
}

// GetToolScenario returns the scenario that provides a tool.
func (e *ToolExecutor) GetToolScenario(toolName string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, handler := range e.handlers {
		if handler.CanHandle(toolName) {
			return handler.Scenario()
		}
	}

	if legacy, exists := e.legacyHandlers[toolName]; exists {
		return legacy.scenarioName
	}

	return ""
}

// ExecuteTool runs a tool and returns the result record.
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

// routeToHandler dispatches to the appropriate scenario handler.
func (e *ToolExecutor) routeToHandler(ctx context.Context, toolName string, args map[string]interface{}, record *domain.ToolCallRecord) (interface{}, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Try scenario handlers first
	for _, handler := range e.handlers {
		if handler.CanHandle(toolName) {
			result, err := handler.Execute(ctx, toolName, args)

			// Extract external run ID if present (for long-running tasks)
			e.extractExternalRunID(result, record)

			return result, err
		}
	}

	// Fall back to legacy handlers
	if legacy, exists := e.legacyHandlers[toolName]; exists {
		result, err := legacy.handler(ctx, args)
		e.extractExternalRunID(result, record)
		return result, err
	}

	return nil, &UnknownToolError{ToolName: toolName}
}

// extractExternalRunID extracts run_id from result if present.
func (e *ToolExecutor) extractExternalRunID(result interface{}, record *domain.ToolCallRecord) {
	if res, ok := result.(map[string]interface{}); ok {
		if runID, ok := res["run_id"].(string); ok {
			record.ExternalRunID = runID
		}
	}
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
	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("[WARN] Failed to marshal tool result for %s: %v", record.ToolName, err)
		// Store a structured error message instead
		record.Result = fmt.Sprintf(`{"error": "result marshal failed: %s", "raw_type": "%T"}`, err.Error(), result)
	} else {
		record.Result = string(resultJSON)
	}
	return record
}

// UnknownToolError indicates a tool name that isn't registered.
type UnknownToolError struct {
	ToolName string
}

func (e *UnknownToolError) Error() string {
	return fmt.Sprintf("unknown tool: %s", e.ToolName)
}

