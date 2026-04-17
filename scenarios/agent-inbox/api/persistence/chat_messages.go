// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains message CRUD operations.
package persistence

import (
	"agent-inbox/domain"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Message Operations

// GetMessages retrieves all messages for a chat.
// Returns all messages including branching metadata (parent_message_id, sibling_index).
func (r *Repository) GetMessages(ctx context.Context, chatID string) ([]domain.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, chat_id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason, parent_message_id, sibling_index, created_at
		FROM messages WHERE chat_id = $1 ORDER BY created_at ASC
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.Message, 0) // Always return [] instead of null in JSON
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
		messages = append(messages, m)
	}

	return messages, nil
}

// GetMessagesForCompletion retrieves messages in the format needed for AI completion.
// For branching support, this returns only messages on the active branch path.
// Falls back to all messages (ordered by created_at) if no active_leaf_message_id is set.
// Includes token_count for context window management.
func (r *Repository) GetMessagesForCompletion(ctx context.Context, chatID string) ([]domain.Message, error) {
	// Get the active leaf for this chat
	activeLeaf, err := r.GetActiveLeaf(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active leaf: %w", err)
	}

	var rows *sql.Rows
	if activeLeaf != "" {
		// Use recursive CTE to get only messages on the active path
		rows, err = r.db.QueryContext(ctx, `
			WITH RECURSIVE active_path AS (
				-- Start from the active leaf
				SELECT id, parent_message_id, role, content, tool_call_id, tool_calls, web_search, token_count, created_at
				FROM messages
				WHERE id = $2 AND chat_id = $1

				UNION ALL

				-- Walk up the tree to parents
				SELECT m.id, m.parent_message_id, m.role, m.content, m.tool_call_id, m.tool_calls, m.web_search, m.token_count, m.created_at
				FROM messages m
				JOIN active_path ap ON m.id = ap.parent_message_id
				WHERE m.chat_id = $1
			)
			SELECT id, role, content, tool_call_id, tool_calls, web_search, token_count
			FROM active_path
			ORDER BY created_at ASC
		`, chatID, activeLeaf)
	} else {
		// Legacy fallback: get all messages ordered by created_at
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, role, content, tool_call_id, tool_calls, web_search, token_count FROM messages WHERE chat_id = $1 ORDER BY created_at ASC
		`, chatID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.Message, 0) // Always return [] instead of null in JSON
	for rows.Next() {
		var msg domain.Message
		var toolCallID sql.NullString
		var toolCallsJSON []byte
		var webSearch sql.NullBool
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &toolCallID, &toolCallsJSON, &webSearch, &msg.TokenCount); err != nil {
			continue
		}
		if toolCallID.Valid {
			msg.ToolCallID = toolCallID.String
		}
		if len(toolCallsJSON) > 0 {
			_ = json.Unmarshal(toolCallsJSON, &msg.ToolCalls) // Error ignored: fallback to empty slice
		}
		if webSearch.Valid {
			val := webSearch.Bool
			msg.WebSearch = &val
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// CreateMessage adds a new message to a chat with optional parent for branching.
// If parentMessageID is provided, sibling_index is auto-calculated based on existing siblings.
// webSearch enables per-message web search override (nil = use chat default).
func (r *Repository) CreateMessage(ctx context.Context, chatID, role, content, model, toolCallID string, tokenCount int, parentMessageID string, webSearch *bool) (*domain.Message, error) {
	// Calculate sibling_index for branching support
	siblingIndex := 0
	if parentMessageID != "" {
		siblingIndex = r.getNextSiblingIndex(ctx, parentMessageID)
	}

	var msg domain.Message
	var webSearchNull sql.NullBool
	if webSearch != nil {
		webSearchNull = sql.NullBool{Bool: *webSearch, Valid: true}
	}

	id := newID()
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (id, chat_id, role, content, model, token_count, tool_call_id, parent_message_id, sibling_index, web_search)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, chat_id, role, content, model, token_count, tool_call_id, parent_message_id, sibling_index, created_at
	`, id, chatID, role, content,
		sql.NullString{String: model, Valid: model != ""},
		tokenCount,
		sql.NullString{String: toolCallID, Valid: toolCallID != ""},
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex,
		webSearchNull).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &sql.NullString{}, &msg.SiblingIndex, scanTime(&msg.CreatedAt),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}
	msg.Model = model
	msg.ToolCallID = toolCallID
	msg.ParentMessageID = parentMessageID
	msg.WebSearch = webSearch
	return &msg, nil
}

// getNextSiblingIndex returns the next available sibling index for a parent message.
func (r *Repository) getNextSiblingIndex(ctx context.Context, parentMessageID string) int {
	var maxIndex sql.NullInt32
	err := r.db.QueryRowContext(ctx, `
		SELECT MAX(sibling_index) FROM messages WHERE parent_message_id = $1
	`, parentMessageID).Scan(&maxIndex)
	if err != nil || !maxIndex.Valid {
		return 0
	}
	return int(maxIndex.Int32) + 1
}

// SaveAssistantMessage saves an assistant response message with optional parent for branching.
func (r *Repository) SaveAssistantMessage(ctx context.Context, chatID, model, content string, tokenCount int, parentMessageID string) (*domain.Message, error) {
	// Calculate sibling_index for branching support
	siblingIndex := 0
	if parentMessageID != "" {
		siblingIndex = r.getNextSiblingIndex(ctx, parentMessageID)
	}

	id := newID()
	var msg domain.Message
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (id, chat_id, role, content, model, token_count, finish_reason, parent_message_id, sibling_index)
		VALUES ($1, $2, 'assistant', $3, $4, $5, 'stop', $6, $7)
		RETURNING id, chat_id, role, content, model, token_count, finish_reason, parent_message_id, sibling_index, created_at
	`, id, chatID, content, sql.NullString{String: model, Valid: model != ""}, tokenCount,
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &sql.NullString{}, &msg.SiblingIndex, scanTime(&msg.CreatedAt),
	)
	if err != nil {
		return nil, err
	}
	msg.Model = model
	msg.FinishReason = "stop"
	msg.ParentMessageID = parentMessageID
	return &msg, nil
}

// SaveAssistantMessageWithToolCalls saves an assistant message that includes tool calls with optional parent for branching.
func (r *Repository) SaveAssistantMessageWithToolCalls(ctx context.Context, chatID, model, content string, toolCalls []domain.ToolCall, responseID, finishReason string, tokenCount int, parentMessageID string) (*domain.Message, error) {
	toolCallsJSON, err := json.Marshal(toolCalls)
	if err != nil {
		return nil, err
	}

	// Calculate sibling_index for branching support
	siblingIndex := 0
	if parentMessageID != "" {
		siblingIndex = r.getNextSiblingIndex(ctx, parentMessageID)
	}

	id := newID()
	var msg domain.Message
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO messages (id, chat_id, role, content, model, token_count, tool_calls, response_id, finish_reason, parent_message_id, sibling_index)
		VALUES ($1, $2, 'assistant', $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, chat_id, role, content, model, token_count, finish_reason, parent_message_id, sibling_index, created_at
	`, id, chatID, content, sql.NullString{String: model, Valid: model != ""}, tokenCount, toolCallsJSON,
		sql.NullString{String: responseID, Valid: responseID != ""},
		sql.NullString{String: finishReason, Valid: finishReason != ""},
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &sql.NullString{}, &msg.SiblingIndex, scanTime(&msg.CreatedAt),
	)
	if err != nil {
		return nil, err
	}
	msg.Model = model
	msg.ToolCalls = toolCalls
	msg.ResponseID = responseID
	msg.FinishReason = finishReason
	msg.ParentMessageID = parentMessageID
	return &msg, nil
}

// SaveToolResponseMessage saves a tool response message with optional parent for branching.
func (r *Repository) SaveToolResponseMessage(ctx context.Context, chatID, toolCallID, result string, parentMessageID string) (*domain.Message, error) {
	// Calculate sibling_index for branching support
	siblingIndex := 0
	if parentMessageID != "" {
		siblingIndex = r.getNextSiblingIndex(ctx, parentMessageID)
	}

	id := newID()
	var msg domain.Message
	err := r.db.QueryRowContext(ctx, `
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
