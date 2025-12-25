// Package services contains business logic orchestration.
// Services coordinate between handlers, persistence, and integrations.
package services

import (
	"context"
	"fmt"

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
	if result.RequiresToolExecution() {
		return s.repo.SaveAssistantMessageWithToolCalls(
			ctx, chatID, model, result.Content, result.ToolCalls,
			result.ResponseID, result.FinishReason, result.TokenCount,
		)
	}
	return s.repo.SaveAssistantMessage(ctx, chatID, model, result.Content, result.TokenCount)
}

// ExecuteToolCalls runs all tool calls from a completion result.
// Returns results for each tool call in order.
func (s *CompletionService) ExecuteToolCalls(ctx context.Context, chatID, messageID string, toolCalls []domain.ToolCall) ([]domain.ToolExecutionResult, error) {
	results := make([]domain.ToolExecutionResult, 0, len(toolCalls))

	for _, tc := range toolCalls {
		record, err := s.executor.ExecuteTool(ctx, chatID, tc.ID, tc.Function.Name, tc.Function.Arguments)

		// Save the execution record
		if messageID != "" {
			s.repo.SaveToolCallRecord(ctx, messageID, record)
		}

		// Save tool response message
		s.repo.SaveToolResponseMessage(ctx, chatID, tc.ID, record.Result)

		// Build result using centralized factory
		results = append(results, NewToolExecutionResult(tc.ID, tc.Function.Name, record, err))
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
