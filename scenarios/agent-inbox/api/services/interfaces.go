// Package services contains business logic orchestration.
// This file defines interfaces for dependency injection and testing.
package services

import (
	"context"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// =============================================================================
// Repository Interface
// =============================================================================

// CompletionRepository defines the database operations needed by CompletionService.
// This interface enables dependency injection for testing.
type CompletionRepository interface {
	// Chat settings
	GetChatSettingsWithWebSearch(ctx context.Context, chatID string) (model string, toolsEnabled bool, webSearchEnabled bool, err error)
	UpdateChatPreview(ctx context.Context, chatID, preview string, markUnread bool) error

	// Message operations
	GetMessagesForCompletion(ctx context.Context, chatID string) ([]domain.Message, error)
	SaveAssistantMessage(ctx context.Context, chatID, model, content string, tokenCount int, parentMessageID string) (*domain.Message, error)
	SaveAssistantMessageWithToolCalls(ctx context.Context, chatID, model, content string, toolCalls []domain.ToolCall, responseID, finishReason string, tokenCount int, parentMessageID string) (*domain.Message, error)
	SaveToolResponseMessage(ctx context.Context, chatID, toolCallID, result string, parentMessageID string) (*domain.Message, error)

	// Message tree operations
	SetActiveLeaf(ctx context.Context, chatID, messageID string) error

	// Attachment operations
	GetAttachmentsForMessages(ctx context.Context, messageIDs []string) (map[string][]domain.Attachment, error)
	CreateAttachment(ctx context.Context, att *domain.Attachment) error
	AttachToMessage(ctx context.Context, attachmentID, messageID string) error

	// Tool call operations
	SaveToolCallRecord(ctx context.Context, messageID string, record *domain.ToolCallRecord) error
	GetToolCallByID(ctx context.Context, toolCallID string) (*domain.ToolCallRecord, error)
	UpdateToolCallStatus(ctx context.Context, id, status, errorMessage string) error
	GetPendingApprovals(ctx context.Context, chatID string) ([]*domain.ToolCallRecord, error)
}

// =============================================================================
// Tool Executor Interface
// =============================================================================

// ToolExecutorInterface defines the methods needed by CompletionService to execute tools.
// This interface enables dependency injection for testing.
type ToolExecutorInterface interface {
	// ExecuteTool executes a tool and returns the result record.
	ExecuteTool(ctx context.Context, chatID, toolCallID, toolName, arguments string) (*domain.ToolCallRecord, error)
}

// =============================================================================
// Async Tracker Interface
// =============================================================================

// AsyncTrackerInterface defines the methods needed by CompletionService for async tracking.
// This interface enables dependency injection for testing.
type AsyncTrackerInterface interface {
	// GetActiveOperations returns all active (non-completed) operations for a chat.
	GetActiveOperations(chatID string) []*AsyncOperation

	// GetOperation returns a specific operation by tool call ID, or nil if not found.
	GetOperation(toolCallID string) *AsyncOperation

	// StartTracking begins tracking an async operation with polling.
	StartTracking(ctx context.Context, toolCallID, chatID, toolName, scenario string, toolResult interface{}, asyncBehavior *toolspb.AsyncBehavior) error
}

// =============================================================================
// Storage Interface (already exists, documenting here for completeness)
// =============================================================================

// StorageService is defined in storage.go - provides file storage operations.
// Already an interface, no changes needed.
