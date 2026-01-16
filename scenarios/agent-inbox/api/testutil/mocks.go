// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"context"
	"fmt"
	"sync"

	"agent-inbox/domain"
	"agent-inbox/services"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// MockToolExecutor provides a controllable tool executor for testing.
type MockToolExecutor struct {
	mu            sync.Mutex
	ExecuteCalls  []MockExecuteCall
	ExecuteResult *domain.ToolCallRecord
	ExecuteError  error
	// ExecuteFunc allows custom behavior per call
	ExecuteFunc func(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error)
}

// MockExecuteCall records a call to ExecuteTool.
type MockExecuteCall struct {
	ChatID     string
	ToolCallID string
	ToolName   string
	Arguments  string
}

// ExecuteTool implements the tool executor interface for tests.
func (m *MockToolExecutor) ExecuteTool(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExecuteCalls = append(m.ExecuteCalls, MockExecuteCall{
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  args,
	})

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, chatID, toolCallID, toolName, args)
	}

	return m.ExecuteResult, m.ExecuteError
}

// Reset clears all recorded calls.
func (m *MockToolExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExecuteCalls = nil
}

// CallCount returns the number of ExecuteTool calls.
func (m *MockToolExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.ExecuteCalls)
}

// LastCall returns the most recent call, or nil if none.
func (m *MockToolExecutor) LastCall() *MockExecuteCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.ExecuteCalls) == 0 {
		return nil
	}
	return &m.ExecuteCalls[len(m.ExecuteCalls)-1]
}

// FakeAsyncTracker provides a controllable async tracker for testing.
type FakeAsyncTracker struct {
	mu                  sync.Mutex
	Operations          map[string]*services.AsyncOperation
	StatusUpdates       chan services.AsyncStatusUpdate
	CompletionEvents    chan services.AsyncCompletionEvent
	StartTrackingCalls  []StartTrackingCall
	StartTrackingError  error
	CancelCalls         []string
	CancelError         error
}

// StartTrackingCall records a call to StartTracking.
type StartTrackingCall struct {
	ToolCallID string
	ChatID     string
	ToolName   string
	Scenario   string
	ToolResult interface{}
}

// NewFakeAsyncTracker creates a new fake tracker for testing.
func NewFakeAsyncTracker() *FakeAsyncTracker {
	return &FakeAsyncTracker{
		Operations:       make(map[string]*services.AsyncOperation),
		StatusUpdates:    make(chan services.AsyncStatusUpdate, 100),
		CompletionEvents: make(chan services.AsyncCompletionEvent, 10),
	}
}

// SimulateUpdate sends a status update to all channels.
func (f *FakeAsyncTracker) SimulateUpdate(update services.AsyncStatusUpdate) {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case f.StatusUpdates <- update:
	default:
	}
}

// SimulateCompletion sends a completion event.
func (f *FakeAsyncTracker) SimulateCompletion(event services.AsyncCompletionEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case f.CompletionEvents <- event:
	default:
	}
}

// GetOperation returns a tracked operation by ID.
func (f *FakeAsyncTracker) GetOperation(toolCallID string) *services.AsyncOperation {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Operations[toolCallID]
}

// Reset clears all state.
func (f *FakeAsyncTracker) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Operations = make(map[string]*services.AsyncOperation)
	f.StartTrackingCalls = nil
	f.CancelCalls = nil
	// Drain channels
	for len(f.StatusUpdates) > 0 {
		<-f.StatusUpdates
	}
	for len(f.CompletionEvents) > 0 {
		<-f.CompletionEvents
	}
}

// MockScenarioHandler provides a controllable scenario handler for testing.
type MockScenarioHandler struct {
	mu             sync.Mutex
	HandleCalls    []MockHandleCall
	HandleResult   *domain.ToolCallRecord
	HandleError    error
	HandleFunc     func(ctx context.Context, toolName, args string) (*domain.ToolCallRecord, error)
}

// MockHandleCall records a call to Handle.
type MockHandleCall struct {
	ToolName  string
	Arguments string
}

// Handle implements the scenario handler interface for tests.
func (m *MockScenarioHandler) Handle(ctx context.Context, toolName, args string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HandleCalls = append(m.HandleCalls, MockHandleCall{
		ToolName:  toolName,
		Arguments: args,
	})

	if m.HandleFunc != nil {
		return m.HandleFunc(ctx, toolName, args)
	}

	return m.HandleResult, m.HandleError
}

// Reset clears all recorded calls.
func (m *MockScenarioHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HandleCalls = nil
}

// MockStreamWriter records SSE events for testing.
type MockStreamWriter struct {
	mu     sync.Mutex
	Events []MockSSEEvent
}

// MockSSEEvent represents a recorded SSE event.
type MockSSEEvent struct {
	Type string
	Data interface{}
}

// NewMockStreamWriter creates a new mock stream writer.
func NewMockStreamWriter() *MockStreamWriter {
	return &MockStreamWriter{}
}

// RecordEvent adds an event to the recorded list.
func (m *MockStreamWriter) RecordEvent(eventType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, MockSSEEvent{Type: eventType, Data: data})
}

// GetEvents returns a copy of all recorded events.
func (m *MockStreamWriter) GetEvents() []MockSSEEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	events := make([]MockSSEEvent, len(m.Events))
	copy(events, m.Events)
	return events
}

// EventCount returns the number of recorded events.
func (m *MockStreamWriter) EventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Events)
}

// FindEvents returns all events of the given type.
func (m *MockStreamWriter) FindEvents(eventType string) []MockSSEEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	var found []MockSSEEvent
	for _, e := range m.Events {
		if e.Type == eventType {
			found = append(found, e)
		}
	}
	return found
}

// Reset clears all recorded events.
func (m *MockStreamWriter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = nil
}

// MockToolRegistry provides a controllable tool registry for testing.
// It implements the methods used by CompletionService.
type MockToolRegistry struct {
	mu sync.Mutex

	// ToolsByName maps tool names to their definitions
	ToolsByName map[string]*toolspb.ToolDefinition

	// ToolScenarios maps tool names to their scenario names
	ToolScenarios map[string]string

	// OpenAITools is the list returned by GetToolsForOpenAI
	OpenAITools []map[string]interface{}

	// ApprovalRequirements maps tool names to approval requirements
	ApprovalRequirements map[string]bool

	// Error responses for testing error paths
	GetToolByNameError        error
	GetToolsForOpenAIError    error
	GetToolApprovalError      error

	// Call tracking
	GetToolByNameCalls      []string
	GetToolsForOpenAICalls  []string
	GetToolApprovalCalls    []GetToolApprovalCall
}

// GetToolApprovalCall records a call to GetToolApprovalRequired.
type GetToolApprovalCall struct {
	ChatID   string
	ToolName string
}

// NewMockToolRegistry creates a new mock tool registry for testing.
func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		ToolsByName:          make(map[string]*toolspb.ToolDefinition),
		ToolScenarios:        make(map[string]string),
		ApprovalRequirements: make(map[string]bool),
	}
}

// AddTool adds a tool to the registry for testing.
func (m *MockToolRegistry) AddTool(scenario string, tool *toolspb.ToolDefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ToolsByName[tool.Name] = tool
	m.ToolScenarios[tool.Name] = scenario
	// Also add to OpenAI tools list
	m.OpenAITools = append(m.OpenAITools, domain.ToOpenAIFunction(tool))
}

// GetToolByName returns a tool by name, bypassing enabled filters.
func (m *MockToolRegistry) GetToolByName(ctx context.Context, toolName string) (*toolspb.ToolDefinition, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetToolByNameCalls = append(m.GetToolByNameCalls, toolName)

	if m.GetToolByNameError != nil {
		return nil, "", m.GetToolByNameError
	}

	tool, ok := m.ToolsByName[toolName]
	if !ok {
		return nil, "", fmt.Errorf("tool not found: %s", toolName)
	}

	scenario := m.ToolScenarios[toolName]
	return tool, scenario, nil
}

// GetToolsForOpenAI returns enabled tools in OpenAI format.
func (m *MockToolRegistry) GetToolsForOpenAI(ctx context.Context, chatID string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetToolsForOpenAICalls = append(m.GetToolsForOpenAICalls, chatID)

	if m.GetToolsForOpenAIError != nil {
		return nil, m.GetToolsForOpenAIError
	}

	return m.OpenAITools, nil
}

// GetToolApprovalRequired checks if a tool requires approval.
func (m *MockToolRegistry) GetToolApprovalRequired(ctx context.Context, chatID, toolName string) (bool, domain.ToolConfigurationScope, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetToolApprovalCalls = append(m.GetToolApprovalCalls, GetToolApprovalCall{
		ChatID:   chatID,
		ToolName: toolName,
	})

	if m.GetToolApprovalError != nil {
		return false, "", m.GetToolApprovalError
	}

	required := m.ApprovalRequirements[toolName]
	return required, "", nil
}

// Reset clears all recorded calls and state.
func (m *MockToolRegistry) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetToolByNameCalls = nil
	m.GetToolsForOpenAICalls = nil
	m.GetToolApprovalCalls = nil
}

// CreateSimpleTool creates a simple tool definition for testing.
func CreateSimpleTool(name, description string) *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        name,
		Description: description,
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: make(map[string]*toolspb.ParameterSchema),
		},
	}
}

// CreateToolWithMetadata creates a tool definition with metadata for testing.
func CreateToolWithMetadata(name, description string, metadata *toolspb.ToolMetadata) *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        name,
		Description: description,
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: make(map[string]*toolspb.ParameterSchema),
		},
		Metadata: metadata,
	}
}
