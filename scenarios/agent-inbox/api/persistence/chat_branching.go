// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains message branching and active leaf operations.
package persistence

import (
	"agent-inbox/domain"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Branching Operations

// GetMessageByID retrieves a single message by ID.
func (r *Repository) GetMessageByID(ctx context.Context, messageID string) (*domain.Message, error) {
	var m domain.Message
	var model, toolCallID, responseID, finishReason, parentMessageID sql.NullString
	var siblingIndex sql.NullInt32
	var toolCallsJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, chat_id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason, parent_message_id, sibling_index, created_at
		FROM messages WHERE id = $1
	`, messageID).Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &model, &m.TokenCount, &toolCallID, &toolCallsJSON, &responseID, &finishReason, &parentMessageID, &siblingIndex, scanTime(&m.CreatedAt))

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if model.Valid {
		m.Model = model.String
	}
	if toolCallID.Valid {
		m.ToolCallID = toolCallID.String
	}
	if len(toolCallsJSON) > 0 {
		_ = json.Unmarshal(toolCallsJSON, &m.ToolCalls) // Error ignored: fallback to empty slice
	}
	if responseID.Valid {
		m.ResponseID = responseID.String
	}
	if finishReason.Valid {
		m.FinishReason = finishReason.String
	}
	if parentMessageID.Valid {
		m.ParentMessageID = parentMessageID.String
	}
	if siblingIndex.Valid {
		m.SiblingIndex = int(siblingIndex.Int32)
	}

	return &m, nil
}

// GetMessageSiblings returns all messages that share the same parent as the given message.
// Includes the message itself. Returns in sibling_index order.
func (r *Repository) GetMessageSiblings(ctx context.Context, messageID string) ([]domain.Message, error) {
	// First get the parent_message_id of the target message
	msg, err := r.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}

	// Query for all siblings (messages with same parent_message_id)
	if msg.ParentMessageID == "" {
		// For root messages (no parent), return just this message
		return []domain.Message{*msg}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, chat_id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason, parent_message_id, sibling_index, created_at
		FROM messages WHERE parent_message_id = $1 ORDER BY sibling_index ASC
	`, msg.ParentMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get siblings: %w", err)
	}
	defer rows.Close()

	siblings := make([]domain.Message, 0)
	for rows.Next() {
		var m domain.Message
		var model, toolCallID, responseID, finishReason, parentMessageID sql.NullString
		var siblingIndex sql.NullInt32
		var toolCallsJSON []byte
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &model, &m.TokenCount, &toolCallID, &toolCallsJSON, &responseID, &finishReason, &parentMessageID, &siblingIndex, scanTime(&m.CreatedAt)); err != nil {
			continue
		}
		if model.Valid {
			m.Model = model.String
		}
		if toolCallID.Valid {
			m.ToolCallID = toolCallID.String
		}
		if len(toolCallsJSON) > 0 {
			_ = json.Unmarshal(toolCallsJSON, &m.ToolCalls) // Error ignored: fallback to empty slice
		}
		if responseID.Valid {
			m.ResponseID = responseID.String
		}
		if finishReason.Valid {
			m.FinishReason = finishReason.String
		}
		if parentMessageID.Valid {
			m.ParentMessageID = parentMessageID.String
		}
		if siblingIndex.Valid {
			m.SiblingIndex = int(siblingIndex.Int32)
		}
		siblings = append(siblings, m)
	}

	return siblings, nil
}

// SetActiveLeaf updates the active_leaf_message_id for a chat.
func (r *Repository) SetActiveLeaf(ctx context.Context, chatID, messageID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chats SET active_leaf_message_id = $1, updated_at = datetime('now') WHERE id = $2
	`, sql.NullString{String: messageID, Valid: messageID != ""}, chatID)
	return err
}

// GetActiveLeaf returns the active_leaf_message_id for a chat.
func (r *Repository) GetActiveLeaf(ctx context.Context, chatID string) (string, error) {
	var activeLeaf sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT active_leaf_message_id FROM chats WHERE id = $1`, chatID).Scan(&activeLeaf)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if activeLeaf.Valid {
		return activeLeaf.String, nil
	}
	return "", nil
}
