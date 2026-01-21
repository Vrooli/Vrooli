// Package services contains business logic orchestration.
// This file provides atomic tool persistence operations with transaction support.
package services

import (
	"context"
	"database/sql"
	"fmt"

	"agent-inbox/domain"
)

// ToolPersistenceRepository defines the interface for database operations used by ToolPersistence.
// This interface enables dependency injection for testing.
type ToolPersistenceRepository interface {
	// BeginTx starts a new transaction.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)

	// Transaction-aware methods
	SaveToolCallRecordTx(ctx context.Context, tx *sql.Tx, messageID string, record *domain.ToolCallRecord) error
	SaveToolResponseMessageTx(ctx context.Context, tx *sql.Tx, chatID, toolCallID, result, parentMessageID string) (*domain.Message, error)
	SetActiveLeafTx(ctx context.Context, tx *sql.Tx, chatID, messageID string) error
}

// SaveToolResultParams contains parameters for SaveToolResult operation.
type SaveToolResultParams struct {
	// ChatID is the chat this tool result belongs to.
	ChatID string
	// MessageID is the assistant message that made the tool call.
	MessageID string
	// ToolCallID is the unique identifier for this tool call.
	ToolCallID string
	// Record contains the tool execution details (status, result, etc.).
	Record *domain.ToolCallRecord
	// Result is the JSON-encoded tool execution result.
	Result string
	// ParentMessageID is the parent message for branching support.
	ParentMessageID string
}

// ToolPersistence provides atomic tool result saving operations.
// It wraps multiple database operations in a transaction to prevent orphaned records.
//
// DESIGN: This addresses the bug where SaveToolCallRecord, SaveToolResponseMessage,
// and SetActiveLeaf could fail independently, leaving the database in an inconsistent state.
// By wrapping all three in a transaction, we ensure all-or-nothing semantics.
type ToolPersistence struct {
	repo ToolPersistenceRepository
}

// NewToolPersistence creates a new ToolPersistence with the given repository.
func NewToolPersistence(repo ToolPersistenceRepository) *ToolPersistence {
	return &ToolPersistence{repo: repo}
}

// SaveToolResult atomically saves a tool execution result.
//
// This performs three operations in a single transaction:
// 1. Save the tool call record (execution status, result, timing)
// 2. Save the tool response message (for conversation continuity)
// 3. Update the active leaf to point to the new message
//
// If any operation fails, the entire transaction is rolled back,
// ensuring no orphaned records are created.
//
// Returns the created tool response message on success, or an error if any step fails.
func (tp *ToolPersistence) SaveToolResult(ctx context.Context, params SaveToolResultParams) (*domain.Message, error) {
	// Start transaction
	tx, err := tp.repo.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	// Always rollback on error - no-op if already committed
	defer tx.Rollback()

	// 1. Save tool call record
	if err := tp.repo.SaveToolCallRecordTx(ctx, tx, params.MessageID, params.Record); err != nil {
		return nil, fmt.Errorf("save tool call record: %w", err)
	}

	// 2. Save tool response message
	toolMsg, err := tp.repo.SaveToolResponseMessageTx(ctx, tx, params.ChatID, params.ToolCallID, params.Result, params.ParentMessageID)
	if err != nil {
		return nil, fmt.Errorf("save tool response message: %w", err)
	}

	// 3. Update active leaf
	if err := tp.repo.SetActiveLeafTx(ctx, tx, params.ChatID, toolMsg.ID); err != nil {
		return nil, fmt.Errorf("set active leaf: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return toolMsg, nil
}

// SaveToolResultWithoutLeafUpdate atomically saves a tool execution result without updating the active leaf.
// This is used when processing multiple tool calls - only the last one should update the leaf.
//
// This performs two operations in a single transaction:
// 1. Save the tool call record (execution status, result, timing)
// 2. Save the tool response message (for conversation continuity)
func (tp *ToolPersistence) SaveToolResultWithoutLeafUpdate(ctx context.Context, params SaveToolResultParams) (*domain.Message, error) {
	// Start transaction
	tx, err := tp.repo.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Save tool call record
	if err := tp.repo.SaveToolCallRecordTx(ctx, tx, params.MessageID, params.Record); err != nil {
		return nil, fmt.Errorf("save tool call record: %w", err)
	}

	// 2. Save tool response message
	toolMsg, err := tp.repo.SaveToolResponseMessageTx(ctx, tx, params.ChatID, params.ToolCallID, params.Result, params.ParentMessageID)
	if err != nil {
		return nil, fmt.Errorf("save tool response message: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return toolMsg, nil
}

// UpdateActiveLeaf updates only the active leaf for a chat.
// This is a non-transactional convenience method for cases where
// only the leaf update is needed (e.g., after processing all tool calls).
func (tp *ToolPersistence) UpdateActiveLeaf(ctx context.Context, chatID, messageID string) error {
	tx, err := tp.repo.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tp.repo.SetActiveLeafTx(ctx, tx, chatID, messageID); err != nil {
		return fmt.Errorf("set active leaf: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
