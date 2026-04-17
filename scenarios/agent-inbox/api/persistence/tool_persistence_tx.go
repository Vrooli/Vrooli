// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains transaction-aware tool persistence operations.
package persistence

import (
	"agent-inbox/domain"
	"context"
	"database/sql"
)

// BeginTx starts a new database transaction.
// This method enables the ToolPersistence service to manage transactions.
func (r *Repository) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, opts)
}

// SaveToolCallRecordTx saves a tool call execution record within a transaction.
// This is the transaction-aware variant of SaveToolCallRecord.
func (r *Repository) SaveToolCallRecordTx(ctx context.Context, tx *sql.Tx, messageID string, record *domain.ToolCallRecord) error {
	// Ensure arguments is valid JSON (empty string is not valid JSON)
	arguments := record.Arguments
	if arguments == "" {
		arguments = "{}"
	}

	// Ensure result is valid JSON or null
	var result interface{}
	if record.Result == "" {
		result = nil
	} else {
		result = record.Result
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO tool_calls (id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			result = EXCLUDED.result,
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			error_message = EXCLUDED.error_message
	`, record.ID, messageID, record.ChatID, record.ToolName, arguments, result, record.Status, record.ScenarioName, record.ExternalRunID, record.StartedAt, record.CompletedAt, record.ErrorMessage)
	return err
}

// SaveToolResponseMessageTx saves a tool response message within a transaction.
// This is the transaction-aware variant of SaveToolResponseMessage.
func (r *Repository) SaveToolResponseMessageTx(ctx context.Context, tx *sql.Tx, chatID, toolCallID, result, parentMessageID string) (*domain.Message, error) {
	// Calculate sibling_index for branching support
	siblingIndex := 0
	if parentMessageID != "" {
		siblingIndex = r.getNextSiblingIndexTx(ctx, tx, parentMessageID)
	}

	id := newID()
	var msg domain.Message
	err := tx.QueryRowContext(ctx, `
		INSERT INTO messages (id, chat_id, role, content, tool_call_id, parent_message_id, sibling_index)
		VALUES ($1, $2, 'tool', $3, $4, $5, $6)
		RETURNING id, chat_id, role, content, tool_call_id, parent_message_id, sibling_index, created_at
	`, id, chatID, result, toolCallID,
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &msg.ToolCallID, &sql.NullString{}, &msg.SiblingIndex, scanTime(&msg.CreatedAt),
	)
	if err != nil {
		return nil, err
	}
	msg.ParentMessageID = parentMessageID
	return &msg, nil
}

// SetActiveLeafTx updates the active_leaf_message_id for a chat within a transaction.
// This is the transaction-aware variant of SetActiveLeaf.
func (r *Repository) SetActiveLeafTx(ctx context.Context, tx *sql.Tx, chatID, messageID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE chats SET active_leaf_message_id = $1, updated_at = datetime('now') WHERE id = $2
	`, sql.NullString{String: messageID, Valid: messageID != ""}, chatID)
	return err
}

// getNextSiblingIndexTx returns the next available sibling index for a parent message within a transaction.
func (r *Repository) getNextSiblingIndexTx(ctx context.Context, tx *sql.Tx, parentMessageID string) int {
	var maxIndex sql.NullInt32
	err := tx.QueryRowContext(ctx, `
		SELECT MAX(sibling_index) FROM messages WHERE parent_message_id = $1
	`, parentMessageID).Scan(&maxIndex)
	if err != nil || !maxIndex.Valid {
		return 0
	}
	return int(maxIndex.Int32) + 1
}
