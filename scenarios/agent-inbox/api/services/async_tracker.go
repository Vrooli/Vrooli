// Package services provides application services for the Agent Inbox scenario.
//
// This file implements the AsyncTrackerService for tracking long-running tool
// operations and providing status updates via Server-Sent Events (SSE).
//
// ARCHITECTURE:
// - AsyncTrackerService: Manages background polling for async tools
// - AsyncOperation: Represents a tracked async tool execution
// - Subscribers receive updates via channels (for SSE endpoints)
//
// POLLING FLOW:
// 1. Tool executor calls StartTracking() after executing an async tool
// 2. Tracker extracts operation ID from result using AsyncBehavior config
// 3. Background goroutine polls status tool at configured intervals
// 4. Updates are pushed to all subscribers for the chat
// 5. Polling stops on terminal status or timeout
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"agent-inbox/domain"
	"agent-inbox/integrations"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// AsyncTrackerService manages background polling for async tool operations.
type AsyncTrackerService struct {
	mu sync.RWMutex

	// Active operations being tracked
	operations map[string]*AsyncOperation // toolCallID -> operation

	// Subscribers for status updates (chatID -> channels)
	subscribers map[string][]chan<- AsyncStatusUpdate

	// Cancellation for active pollers
	cancelFuncs map[string]context.CancelFunc // toolCallID -> cancel

	// Dependencies
	toolRegistry *ToolRegistry
	toolExecutor *integrations.ToolExecutor
}

// AsyncOperation represents a tracked async tool execution.
type AsyncOperation struct {
	ToolCallID    string                   `json:"tool_call_id"`
	ChatID        string                   `json:"chat_id"`
	ToolName      string                   `json:"tool_name"`
	Scenario      string                   `json:"scenario"`
	ExternalRunID string                   `json:"external_run_id"`
	AsyncBehavior *toolspb.AsyncBehavior   `json:"-"`
	Status        string                   `json:"status"`
	Progress      *int                     `json:"progress,omitempty"`
	Message       string                   `json:"message,omitempty"`
	Phase         string                   `json:"phase,omitempty"`
	Result        interface{}              `json:"result,omitempty"`
	Error         string                   `json:"error,omitempty"`
	StartedAt     time.Time                `json:"started_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	CompletedAt   *time.Time               `json:"completed_at,omitempty"`
}

// AsyncStatusUpdate represents a status update pushed to subscribers.
type AsyncStatusUpdate struct {
	ToolCallID  string      `json:"tool_call_id"`
	ChatID      string      `json:"chat_id"`
	ToolName    string      `json:"tool_name"`
	Status      string      `json:"status"`
	Progress    *int        `json:"progress,omitempty"`
	Message     string      `json:"message,omitempty"`
	Phase       string      `json:"phase,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	Error       string      `json:"error,omitempty"`
	IsTerminal  bool        `json:"is_terminal"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// NewAsyncTrackerService creates a new AsyncTrackerService.
func NewAsyncTrackerService(registry *ToolRegistry, executor *integrations.ToolExecutor) *AsyncTrackerService {
	return &AsyncTrackerService{
		operations:   make(map[string]*AsyncOperation),
		subscribers:  make(map[string][]chan<- AsyncStatusUpdate),
		cancelFuncs:  make(map[string]context.CancelFunc),
		toolRegistry: registry,
		toolExecutor: executor,
	}
}

// StartTracking begins tracking an async tool operation.
// This should be called after executing a tool that has AsyncBehavior defined.
func (s *AsyncTrackerService) StartTracking(
	ctx context.Context,
	toolCallID string,
	chatID string,
	toolName string,
	scenario string,
	toolResult interface{},
	asyncBehavior *toolspb.AsyncBehavior,
) error {
	if asyncBehavior == nil || asyncBehavior.StatusPolling == nil {
		return fmt.Errorf("no async behavior configuration provided")
	}

	// Extract the operation ID from the tool result
	externalRunID, err := s.extractOperationID(toolResult, asyncBehavior.StatusPolling.OperationIdField)
	if err != nil {
		return fmt.Errorf("failed to extract operation ID: %w", err)
	}

	// Create the operation record
	op := &AsyncOperation{
		ToolCallID:    toolCallID,
		ChatID:        chatID,
		ToolName:      toolName,
		Scenario:      scenario,
		ExternalRunID: externalRunID,
		AsyncBehavior: asyncBehavior,
		Status:        "pending",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Store the operation
	s.mu.Lock()
	s.operations[toolCallID] = op
	s.mu.Unlock()

	// Push initial update to subscribers so UI shows the operation immediately
	s.pushUpdate(op, false)

	// Create cancellable context for polling
	pollCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFuncs[toolCallID] = cancel
	s.mu.Unlock()

	// Start background polling
	go s.pollLoop(pollCtx, op)

	log.Printf("Started async tracking for %s/%s (toolCallID=%s, runID=%s)",
		scenario, toolName, toolCallID, externalRunID)

	return nil
}

// StopTracking cancels tracking for an operation.
func (s *AsyncTrackerService) StopTracking(toolCallID string) {
	s.mu.Lock()
	if cancel, ok := s.cancelFuncs[toolCallID]; ok {
		cancel()
		delete(s.cancelFuncs, toolCallID)
	}
	s.mu.Unlock()
}

// GetOperation returns an operation by ID.
func (s *AsyncTrackerService) GetOperation(toolCallID string) *AsyncOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.operations[toolCallID]
}

// GetActiveOperations returns all active operations for a chat.
func (s *AsyncTrackerService) GetActiveOperations(chatID string) []*AsyncOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*AsyncOperation
	for _, op := range s.operations {
		if op.ChatID == chatID && op.CompletedAt == nil {
			result = append(result, op)
		}
	}
	return result
}

// Subscribe creates a channel for receiving updates for a chat.
// The caller is responsible for calling Unsubscribe when done.
func (s *AsyncTrackerService) Subscribe(chatID string) <-chan AsyncStatusUpdate {
	ch := make(chan AsyncStatusUpdate, 100) // Buffer to prevent blocking

	s.mu.Lock()
	s.subscribers[chatID] = append(s.subscribers[chatID], ch)
	s.mu.Unlock()

	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *AsyncTrackerService) Unsubscribe(chatID string, ch <-chan AsyncStatusUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[chatID]
	for i, sub := range subs {
		// Compare channel addresses by converting to bidirectional channel type
		// Note: This works because we know the channel was created as bidirectional
		if fmt.Sprintf("%p", sub) == fmt.Sprintf("%p", ch) {
			s.subscribers[chatID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(s.subscribers[chatID]) == 0 {
		delete(s.subscribers, chatID)
	}
}

// pollLoop runs the background polling for an operation.
func (s *AsyncTrackerService) pollLoop(ctx context.Context, op *AsyncOperation) {
	polling := op.AsyncBehavior.StatusPolling
	conditions := op.AsyncBehavior.CompletionConditions

	interval := time.Duration(polling.PollIntervalSeconds) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second // Default to 5 seconds
	}

	maxDuration := time.Duration(polling.MaxPollDurationSeconds) * time.Second
	if maxDuration <= 0 {
		maxDuration = time.Hour // Default to 1 hour
	}

	deadline := op.StartedAt.Add(maxDuration)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Polling cancelled for %s", op.ToolCallID)
			return
		case <-ticker.C:
			if time.Now().After(deadline) {
				s.handleTimeout(op)
				return
			}

			// Call the status tool
			statusResult, err := s.callStatusTool(ctx, op)
			if err != nil {
				log.Printf("Error polling status for %s: %v", op.ToolCallID, err)
				continue // Keep trying
			}

			// Process the status result
			terminal := s.processStatusResult(op, statusResult, conditions)
			if terminal {
				log.Printf("Operation %s reached terminal status: %s", op.ToolCallID, op.Status)
				return
			}
		}
	}
}

// callStatusTool invokes the status tool to check operation progress.
func (s *AsyncTrackerService) callStatusTool(ctx context.Context, op *AsyncOperation) (interface{}, error) {
	polling := op.AsyncBehavior.StatusPolling

	// Build arguments for status tool
	args := map[string]interface{}{
		polling.StatusToolIdParam: op.ExternalRunID,
	}
	argsJSON, _ := json.Marshal(args)

	// Execute the status tool
	record, err := s.toolExecutor.ExecuteTool(ctx, op.ChatID, "", polling.StatusTool, string(argsJSON))
	if err != nil {
		return nil, err
	}
	if record.Status == domain.StatusFailed {
		return nil, fmt.Errorf("status tool failed: %s", record.ErrorMessage)
	}

	// Parse the result
	var result interface{}
	if err := json.Unmarshal([]byte(record.Result), &result); err != nil {
		return nil, fmt.Errorf("failed to parse status result: %w", err)
	}

	return result, nil
}

// processStatusResult updates the operation based on status tool response.
// Returns true if the operation has reached a terminal status.
func (s *AsyncTrackerService) processStatusResult(op *AsyncOperation, result interface{}, conditions *toolspb.CompletionConditions) bool {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return false
	}

	// Extract status
	status := extractStringField(resultMap, conditions.StatusField)
	if status == "" {
		return false
	}

	// Update operation
	s.mu.Lock()
	op.Status = status
	op.UpdatedAt = time.Now()

	// Extract progress if configured
	if op.AsyncBehavior.ProgressTracking != nil {
		if progress := extractIntField(resultMap, op.AsyncBehavior.ProgressTracking.ProgressField); progress != nil {
			op.Progress = progress
		}
		if message := extractStringField(resultMap, op.AsyncBehavior.ProgressTracking.MessageField); message != "" {
			op.Message = message
		}
		if phase := extractStringField(resultMap, op.AsyncBehavior.ProgressTracking.PhaseField); phase != "" {
			op.Phase = phase
		}
	}

	// Check for error
	if conditions.ErrorField != "" {
		if errMsg := extractStringField(resultMap, conditions.ErrorField); errMsg != "" {
			op.Error = errMsg
		}
	}

	// Check for result
	if conditions.ResultField != "" {
		if resultVal := extractField(resultMap, conditions.ResultField); resultVal != nil {
			op.Result = resultVal
		}
	}

	// Check terminal conditions
	isSuccess := contains(conditions.SuccessValues, status)
	isFailure := contains(conditions.FailureValues, status)
	isTerminal := isSuccess || isFailure

	if isTerminal {
		now := time.Now()
		op.CompletedAt = &now
	}
	s.mu.Unlock()

	// Push update to subscribers
	s.pushUpdate(op, isTerminal)

	return isTerminal
}

// handleTimeout marks an operation as timed out.
func (s *AsyncTrackerService) handleTimeout(op *AsyncOperation) {
	s.mu.Lock()
	op.Status = "timeout"
	op.Error = "Operation timed out"
	now := time.Now()
	op.CompletedAt = &now
	op.UpdatedAt = now
	s.mu.Unlock()

	s.pushUpdate(op, true)

	log.Printf("Operation %s timed out", op.ToolCallID)
}

// pushUpdate sends an update to all subscribers for the chat.
func (s *AsyncTrackerService) pushUpdate(op *AsyncOperation, isTerminal bool) {
	update := AsyncStatusUpdate{
		ToolCallID: op.ToolCallID,
		ChatID:     op.ChatID,
		ToolName:   op.ToolName,
		Status:     op.Status,
		Progress:   op.Progress,
		Message:    op.Message,
		Phase:      op.Phase,
		Result:     op.Result,
		Error:      op.Error,
		IsTerminal: isTerminal,
		UpdatedAt:  op.UpdatedAt,
	}

	s.mu.RLock()
	subs := s.subscribers[op.ChatID]
	s.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- update:
		default:
			// Channel full, skip this update
			log.Printf("Warning: subscriber channel full for chat %s", op.ChatID)
		}
	}
}

// extractOperationID extracts the operation ID from the tool result.
func (s *AsyncTrackerService) extractOperationID(result interface{}, fieldPath string) (string, error) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("result is not a map")
	}

	value := extractStringField(resultMap, fieldPath)
	if value == "" {
		return "", fmt.Errorf("field %s not found or empty", fieldPath)
	}

	return value, nil
}

// CancelOperation cancels a running async operation using the configured cancel tool.
func (s *AsyncTrackerService) CancelOperation(ctx context.Context, toolCallID string) error {
	s.mu.RLock()
	op := s.operations[toolCallID]
	s.mu.RUnlock()

	if op == nil {
		return fmt.Errorf("operation not found: %s", toolCallID)
	}

	if op.AsyncBehavior.Cancellation == nil {
		return fmt.Errorf("no cancellation behavior configured for this tool")
	}

	cancel := op.AsyncBehavior.Cancellation

	// Build arguments for cancel tool
	args := map[string]interface{}{
		cancel.CancelToolIdParam: op.ExternalRunID,
	}
	argsJSON, _ := json.Marshal(args)

	// Execute the cancel tool
	_, err := s.toolExecutor.ExecuteTool(ctx, op.ChatID, "", cancel.CancelTool, string(argsJSON))
	if err != nil {
		return fmt.Errorf("failed to cancel operation: %w", err)
	}

	// Stop tracking
	s.StopTracking(toolCallID)

	return nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// extractStringField extracts a string value from a nested map using dot notation.
func extractStringField(m map[string]interface{}, path string) string {
	val := extractField(m, path)
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// extractIntField extracts an int value from a nested map using dot notation.
func extractIntField(m map[string]interface{}, path string) *int {
	val := extractField(m, path)
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case float64:
		i := int(v)
		return &i
	case int:
		return &v
	case int64:
		i := int(v)
		return &i
	}
	return nil
}

// extractField extracts a value from a nested map using dot notation.
func extractField(m map[string]interface{}, path string) interface{} {
	parts := splitPath(path)
	current := interface{}(m)

	for _, part := range parts {
		if currentMap, ok := current.(map[string]interface{}); ok {
			current = currentMap[part]
			if current == nil {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

// splitPath splits a dot-notation path into parts.
func splitPath(path string) []string {
	var parts []string
	var current string
	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
