// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains the chat fork operation.
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"agent-inbox/domain"
)

// messageForFork holds message data temporarily during fork operation.
type messageForFork struct {
	role          string
	content       string
	model         sql.NullString
	tokenCount    int
	toolCallID    sql.NullString
	toolCallsJSON []byte
	responseID    sql.NullString
	finishReason  sql.NullString
}

// ForkChat creates a new chat by copying messages from a source chat up to and including a specific message.
// Uses a recursive CTE to trace the message ancestry and copies all messages in the path.
func (r *Repository) ForkChat(ctx context.Context, sourceChatID, upToMessageID, newName, model string) (*domain.Chat, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Create the new chat
	chatID := newID()
	var newChat domain.Chat
	err = tx.QueryRowContext(ctx, `
		INSERT INTO chats (id, name, model, view_mode)
		VALUES ($1, $2, $3, 'bubble')
		RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, tools_enabled, created_at, updated_at
	`, chatID, newName, model).Scan(
		&newChat.ID, &newChat.Name, &newChat.Preview, &newChat.Model, &newChat.ViewMode,
		&newChat.IsRead, &newChat.IsArchived, &newChat.IsStarred, &newChat.ToolsEnabled, scanTime(&newChat.CreatedAt), scanTime(&newChat.UpdatedAt),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create forked chat: %w", err)
	}
	newChat.LabelIDs = []string{}

	// Get the message ancestry path from the target message back to root
	rows, err := tx.QueryContext(ctx, `
		WITH RECURSIVE message_path AS (
			-- Start from the target message
			SELECT id, parent_message_id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason, created_at, 0 as depth
			FROM messages
			WHERE id = $1 AND chat_id = $2

			UNION ALL

			-- Walk up to parent messages
			SELECT m.id, m.parent_message_id, m.role, m.content, m.model, m.token_count, m.tool_call_id, m.tool_calls, m.response_id, m.finish_reason, m.created_at, mp.depth + 1
			FROM messages m
			JOIN message_path mp ON m.id = mp.parent_message_id
			WHERE m.chat_id = $2
		)
		SELECT id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason
		FROM message_path
		ORDER BY depth DESC, created_at ASC
	`, upToMessageID, sourceChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message path: %w", err)
	}

	// Collect all messages first
	var messagesToCopy []messageForFork
	for rows.Next() {
		var msgID string
		var msg messageForFork
		if err := rows.Scan(&msgID, &msg.role, &msg.content, &msg.model, &msg.tokenCount, &msg.toolCallID, &msg.toolCallsJSON, &msg.responseID, &msg.finishReason); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messagesToCopy = append(messagesToCopy, msg)
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	// Now insert messages into the new chat
	var lastMessageID string
	var lastContent string

	for _, msg := range messagesToCopy {
		// Handle NULL tool_calls
		var toolCallsArg interface{}
		if len(msg.toolCallsJSON) > 0 {
			toolCallsArg = msg.toolCallsJSON
		} else {
			toolCallsArg = nil
		}

		msgID := newID()
		var newMsgID string
		err = tx.QueryRowContext(ctx, `
			INSERT INTO messages (id, chat_id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason, parent_message_id, sibling_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 0)
			RETURNING id
		`, msgID, newChat.ID, msg.role, msg.content, msg.model, msg.tokenCount, msg.toolCallID, toolCallsArg, msg.responseID, msg.finishReason,
			sql.NullString{String: lastMessageID, Valid: lastMessageID != ""}).Scan(&newMsgID)
		if err != nil {
			return nil, fmt.Errorf("failed to copy message: %w", err)
		}

		lastMessageID = newMsgID
		lastContent = msg.content
	}

	// Update preview and active leaf of the new chat
	if lastMessageID != "" {
		preview := lastContent
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE chats SET preview = $1, active_leaf_message_id = $2 WHERE id = $3
		`, preview, lastMessageID, newChat.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to update forked chat: %w", err)
		}
		newChat.Preview = preview
		newChat.ActiveLeafMessageID = lastMessageID
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &newChat, nil
}
