// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains chat and message operations.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"agent-inbox/domain"
)

// Chat Operations

// ListChats returns all chats matching the given filters.
func (r *Repository) ListChats(ctx context.Context, archived, starred bool) ([]domain.Chat, error) {
	query := `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at,
			COALESCE(array_agg(cl.label_id) FILTER (WHERE cl.label_id IS NOT NULL), '{}') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE c.is_archived = $1
	`
	args := []interface{}{archived}

	if starred {
		query += " AND c.is_starred = true"
	}

	query += " GROUP BY c.id ORDER BY c.is_starred DESC, c.updated_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list chats: %w", err)
	}
	defer rows.Close()

	var chats []domain.Chat
	for rows.Next() {
		var c domain.Chat
		var labelIDs []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Preview, &c.Model, &c.ViewMode, &c.IsRead, &c.IsArchived, &c.IsStarred, &c.CreatedAt, &c.UpdatedAt, &labelIDs); err != nil {
			continue
		}
		c.LabelIDs = parsePostgresArray(string(labelIDs))
		chats = append(chats, c)
	}

	return chats, nil
}

// GetChat retrieves a single chat by ID.
func (r *Repository) GetChat(ctx context.Context, chatID string) (*domain.Chat, error) {
	var chat domain.Chat
	var labelIDs []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at,
			COALESCE(array_agg(cl.label_id) FILTER (WHERE cl.label_id IS NOT NULL), '{}') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE c.id = $1
		GROUP BY c.id
	`, chatID).Scan(&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode, &chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.CreatedAt, &chat.UpdatedAt, &labelIDs)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	chat.LabelIDs = parsePostgresArray(string(labelIDs))
	return &chat, nil
}

// GetChatSettings retrieves just model and tools_enabled for a chat.
func (r *Repository) GetChatSettings(ctx context.Context, chatID string) (model string, toolsEnabled bool, err error) {
	err = r.db.QueryRowContext(ctx, "SELECT model, tools_enabled FROM chats WHERE id = $1", chatID).Scan(&model, &toolsEnabled)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return model, toolsEnabled, err
}

// CreateChat creates a new chat with the given parameters.
func (r *Repository) CreateChat(ctx context.Context, name, model, viewMode string) (*domain.Chat, error) {
	var chat domain.Chat
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO chats (name, model, view_mode)
		VALUES ($1, $2, $3)
		RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, created_at, updated_at
	`, name, model, viewMode).Scan(
		&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode,
		&chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.CreatedAt, &chat.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}
	chat.LabelIDs = []string{}
	return &chat, nil
}

// UpdateChat updates a chat's name and/or model.
func (r *Repository) UpdateChat(ctx context.Context, chatID string, name, model *string) (*domain.Chat, error) {
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *name)
		argNum++
	}
	if model != nil {
		updates = append(updates, fmt.Sprintf("model = $%d", argNum))
		args = append(args, *model)
		argNum++
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, chatID)

	query := fmt.Sprintf("UPDATE chats SET %s WHERE id = $%d RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, created_at, updated_at",
		strings.Join(updates, ", "), argNum)

	var chat domain.Chat
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode,
		&chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.CreatedAt, &chat.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update chat: %w", err)
	}

	// Get label IDs
	chat.LabelIDs = r.getChatLabelIDs(ctx, chatID)
	return &chat, nil
}

// DeleteChat removes a chat by ID.
func (r *Repository) DeleteChat(ctx context.Context, chatID string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM chats WHERE id = $1", chatID)
	if err != nil {
		return false, fmt.Errorf("failed to delete chat: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

// ChatExists checks if a chat with the given ID exists.
func (r *Repository) ChatExists(ctx context.Context, chatID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1)", chatID).Scan(&exists)
	return exists, err
}

// ToggleChatBool toggles or sets a boolean field on a chat.
func (r *Repository) ToggleChatBool(ctx context.Context, chatID, field string, value *bool) (bool, error) {
	var query string
	var newValue bool
	var err error

	if value != nil {
		query = fmt.Sprintf("UPDATE chats SET %s = $1, updated_at = NOW() WHERE id = $2 RETURNING %s", field, field)
		err = r.db.QueryRowContext(ctx, query, *value, chatID).Scan(&newValue)
	} else {
		query = fmt.Sprintf("UPDATE chats SET %s = NOT %s, updated_at = NOW() WHERE id = $1 RETURNING %s", field, field, field)
		err = r.db.QueryRowContext(ctx, query, chatID).Scan(&newValue)
	}

	if err == sql.ErrNoRows {
		return false, fmt.Errorf("chat not found")
	}
	return newValue, err
}

// UpdateChatPreview updates the preview text and optionally marks as unread.
func (r *Repository) UpdateChatPreview(ctx context.Context, chatID, preview string, markUnread bool) error {
	query := "UPDATE chats SET preview = $1, updated_at = NOW()"
	if markUnread {
		query += ", is_read = false"
	}
	query += " WHERE id = $2"
	_, err := r.db.ExecContext(ctx, query, preview, chatID)
	return err
}

// Message Operations

// GetMessages retrieves all messages for a chat.
func (r *Repository) GetMessages(ctx context.Context, chatID string) ([]domain.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, chat_id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason, created_at
		FROM messages WHERE chat_id = $1 ORDER BY created_at ASC
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		var model, toolCallID, responseID, finishReason sql.NullString
		var toolCallsJSON []byte
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &model, &m.TokenCount, &toolCallID, &toolCallsJSON, &responseID, &finishReason, &m.CreatedAt); err != nil {
			continue
		}
		if model.Valid {
			m.Model = model.String
		}
		if toolCallID.Valid {
			m.ToolCallID = toolCallID.String
		}
		if len(toolCallsJSON) > 0 {
			json.Unmarshal(toolCallsJSON, &m.ToolCalls)
		}
		if responseID.Valid {
			m.ResponseID = responseID.String
		}
		if finishReason.Valid {
			m.FinishReason = finishReason.String
		}
		messages = append(messages, m)
	}

	return messages, nil
}

// GetMessagesForCompletion retrieves messages in the format needed for AI completion.
func (r *Repository) GetMessagesForCompletion(ctx context.Context, chatID string) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT role, content, tool_call_id, tool_calls FROM messages WHERE chat_id = $1 ORDER BY created_at ASC
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []map[string]interface{}
	for rows.Next() {
		var role, content string
		var toolCallID sql.NullString
		var toolCallsJSON []byte
		if err := rows.Scan(&role, &content, &toolCallID, &toolCallsJSON); err != nil {
			continue
		}
		msg := map[string]interface{}{
			"role":    role,
			"content": content,
		}
		if toolCallID.Valid {
			msg["tool_call_id"] = toolCallID.String
		}
		if len(toolCallsJSON) > 0 {
			var toolCalls []domain.ToolCall
			json.Unmarshal(toolCallsJSON, &toolCalls)
			msg["tool_calls"] = toolCalls
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// CreateMessage adds a new message to a chat.
func (r *Repository) CreateMessage(ctx context.Context, chatID, role, content, model, toolCallID string, tokenCount int) (*domain.Message, error) {
	var msg domain.Message
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, model, token_count, tool_call_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, chat_id, role, content, model, token_count, tool_call_id, created_at
	`, chatID, role, content,
		sql.NullString{String: model, Valid: model != ""},
		tokenCount,
		sql.NullString{String: toolCallID, Valid: toolCallID != ""}).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}
	msg.Model = model
	msg.ToolCallID = toolCallID
	return &msg, nil
}

// SaveAssistantMessage saves an assistant response message.
func (r *Repository) SaveAssistantMessage(ctx context.Context, chatID, model, content string, tokenCount int) (*domain.Message, error) {
	var msg domain.Message
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, model, token_count, finish_reason)
		VALUES ($1, 'assistant', $2, $3, $4, 'stop')
		RETURNING id, chat_id, role, content, model, token_count, finish_reason, created_at
	`, chatID, content, sql.NullString{String: model, Valid: model != ""}, tokenCount).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	msg.Model = model
	msg.FinishReason = "stop"
	return &msg, nil
}

// SaveAssistantMessageWithToolCalls saves an assistant message that includes tool calls.
func (r *Repository) SaveAssistantMessageWithToolCalls(ctx context.Context, chatID, model, content string, toolCalls []domain.ToolCall, responseID, finishReason string, tokenCount int) (*domain.Message, error) {
	toolCallsJSON, err := json.Marshal(toolCalls)
	if err != nil {
		return nil, err
	}

	var msg domain.Message
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, model, token_count, tool_calls, response_id, finish_reason)
		VALUES ($1, 'assistant', $2, $3, $4, $5, $6, $7)
		RETURNING id, chat_id, role, content, model, token_count, finish_reason, created_at
	`, chatID, content, sql.NullString{String: model, Valid: model != ""}, tokenCount, toolCallsJSON,
		sql.NullString{String: responseID, Valid: responseID != ""},
		sql.NullString{String: finishReason, Valid: finishReason != ""}).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	msg.Model = model
	msg.ToolCalls = toolCalls
	msg.ResponseID = responseID
	msg.FinishReason = finishReason
	return &msg, nil
}

// SaveToolResponseMessage saves a tool response message.
func (r *Repository) SaveToolResponseMessage(ctx context.Context, chatID, toolCallID, result string) (*domain.Message, error) {
	var msg domain.Message
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, tool_call_id)
		VALUES ($1, 'tool', $2, $3)
		RETURNING id, chat_id, role, content, tool_call_id, created_at
	`, chatID, result, toolCallID).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &msg.ToolCallID, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
