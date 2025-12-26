// Package services contains business logic orchestration.
// Services coordinate between handlers, persistence, and integrations.
package services

import (
	"context"
	"fmt"
	"log"

	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
)

// NewToolExecutionResult creates a ToolExecutionResult from a record and optional error.
// This centralizes the decision of how to populate the result based on success/failure.
func NewToolExecutionResult(toolCallID, toolName string, record *domain.ToolCallRecord, err error) domain.ToolExecutionResult {
	result := domain.ToolExecutionResult{
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Status:     record.Status,
	}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Result = record.Result
	}
	return result
}

// CompletionService orchestrates AI chat completion.
// It handles the decision flow for completing a chat with an AI model,
// including tool execution when requested by the model.
type CompletionService struct {
	repo     *persistence.Repository
	executor *integrations.ToolExecutor
}

// NewCompletionService creates a new completion service.
func NewCompletionService(repo *persistence.Repository) *CompletionService {
	return &CompletionService{
		repo:     repo,
		executor: integrations.NewToolExecutor(),
	}
}

// SaveCompletionResult persists a completion result to the database.
// This handles the decision of whether to save as a regular message or
// as a message with tool calls.
func (s *CompletionService) SaveCompletionResult(ctx context.Context, chatID, model string, result *domain.CompletionResult) (*domain.Message, error) {
	var msg *domain.Message
	var err error

	if result.RequiresToolExecution() {
		msg, err = s.repo.SaveAssistantMessageWithToolCalls(
			ctx, chatID, model, result.Content, result.ToolCalls,
			result.ResponseID, result.FinishReason, result.TokenCount,
		)
	} else {
		msg, err = s.repo.SaveAssistantMessage(ctx, chatID, model, result.Content, result.TokenCount)
	}

	if err != nil {
		return msg, err
	}

	// Save usage record if usage data is available
	if result.Usage != nil && msg != nil {
		usageRecord := integrations.CreateUsageRecord(chatID, msg.ID, model, result.Usage)
		if usageRecord != nil {
			if saveErr := s.repo.SaveUsageRecord(ctx, usageRecord); saveErr != nil {
				// Log but don't fail the request - usage tracking is non-critical
				log.Printf("warning: failed to save usage record: %v", saveErr)
			}
		}
	}

	return msg, nil
}

// ExecuteToolCalls runs all tool calls from a completion result.
// Returns results for each tool call in order.
//
// TEMPORAL FLOW NOTE: This function executes tool calls sequentially and
// collects all errors. The returned error aggregates all failures but individual
// tool results are still returned for partial success handling.
//
// Error Handling:
//   - Individual tool errors are captured in each ToolExecutionResult
//   - The returned error is non-nil if ANY tool call failed
//   - Callers can inspect individual results for partial success scenarios
func (s *CompletionService) ExecuteToolCalls(ctx context.Context, chatID, messageID string, toolCalls []domain.ToolCall) ([]domain.ToolExecutionResult, error) {
	results := make([]domain.ToolExecutionResult, 0, len(toolCalls))
	var executionErrors []error

	for _, tc := range toolCalls {
		record, err := s.executor.ExecuteTool(ctx, chatID, tc.ID, tc.Function.Name, tc.Function.Arguments)
		// Track errors for aggregated reporting
		if err != nil {
			executionErrors = append(executionErrors, fmt.Errorf("tool %s failed: %w", tc.Function.Name, err))
		}

		// Save the execution record
		if messageID != "" {
			s.repo.SaveToolCallRecord(ctx, messageID, record)
		}

		// Save tool response message
		s.repo.SaveToolResponseMessage(ctx, chatID, tc.ID, record.Result)

		// Build result using centralized factory
		results = append(results, NewToolExecutionResult(tc.ID, tc.Function.Name, record, err))
	}

	// Return aggregated error if any tool failed
	if len(executionErrors) > 0 {
		return results, fmt.Errorf("%d of %d tool calls failed: %v", len(executionErrors), len(toolCalls), executionErrors[0])
	}

	return results, nil
}

// UpdateChatPreview updates the chat's preview text based on completion result.
func (s *CompletionService) UpdateChatPreview(ctx context.Context, chatID string, result *domain.CompletionResult) error {
	preview := result.PreviewText()
	return s.repo.UpdateChatPreview(ctx, chatID, preview, true)
}

// ChatSettings contains the settings needed for chat completion.
type ChatSettings struct {
	Model        string
	ToolsEnabled bool
}

// GetChatSettings retrieves settings for a chat completion.
// Returns nil if chat doesn't exist.
func (s *CompletionService) GetChatSettings(ctx context.Context, chatID string) (*ChatSettings, error) {
	model, toolsEnabled, err := s.repo.GetChatSettings(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, nil // Chat not found
	}
	return &ChatSettings{
		Model:        model,
		ToolsEnabled: toolsEnabled,
	}, nil
}

// CompletionRequest contains validated data needed to make a completion.
type CompletionRequest struct {
	ChatID    string
	Model     string
	Messages  []integrations.OpenRouterMessage
	Tools     []integrations.ToolDefinition
	Streaming bool
}

// ShouldIncludeTools returns true if tools should be sent with the request.
func (r *CompletionRequest) ShouldIncludeTools() bool {
	return len(r.Tools) > 0
}

// PrepareCompletionRequest builds a validated completion request.
// Returns an error if the chat doesn't exist or has no messages.
// Uses sentinel errors (ErrChatNotFound, ErrNoMessages) for type-safe error handling.
func (s *CompletionService) PrepareCompletionRequest(ctx context.Context, chatID string, streaming bool) (*CompletionRequest, error) {
	// Get chat settings
	settings, err := s.GetChatSettings(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	if settings == nil {
		return nil, ErrChatNotFound
	}

	// Get messages
	messages, err := s.repo.GetMessagesForCompletion(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMessagesFailed, err)
	}
	if len(messages) == 0 {
		return nil, ErrNoMessages
	}

	req := &CompletionRequest{
		ChatID:    chatID,
		Model:     settings.Model,
		Messages:  integrations.ConvertMessages(messages),
		Streaming: streaming,
	}

	if settings.ToolsEnabled {
		req.Tools = integrations.AvailableTools()
	}

	return req, nil
}
