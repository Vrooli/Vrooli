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

	chats := make([]domain.Chat, 0) // Always return [] instead of null in JSON
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
	var activeLeafMessageID sql.NullString
	var webSearchEnabled sql.NullBool

	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.web_search_enabled, c.active_leaf_message_id, c.created_at, c.updated_at,
			COALESCE(array_agg(cl.label_id) FILTER (WHERE cl.label_id IS NOT NULL), '{}') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE c.id = $1
		GROUP BY c.id
	`, chatID).Scan(&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode, &chat.IsRead, &chat.IsArchived, &chat.IsStarred, &webSearchEnabled, &activeLeafMessageID, &chat.CreatedAt, &chat.UpdatedAt, &labelIDs)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	chat.LabelIDs = parsePostgresArray(string(labelIDs))
	if activeLeafMessageID.Valid {
		chat.ActiveLeafMessageID = activeLeafMessageID.String
	}
	if webSearchEnabled.Valid {
		chat.WebSearchEnabled = webSearchEnabled.Bool
	}
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

// GetChatSettingsWithWebSearch retrieves model, tools_enabled, and web_search_enabled for a chat.
func (r *Repository) GetChatSettingsWithWebSearch(ctx context.Context, chatID string) (model string, toolsEnabled bool, webSearchEnabled bool, err error) {
	var webSearchNull sql.NullBool
	err = r.db.QueryRowContext(ctx, "SELECT model, tools_enabled, web_search_enabled FROM chats WHERE id = $1", chatID).Scan(&model, &toolsEnabled, &webSearchNull)
	if err == sql.ErrNoRows {
		return "", false, false, nil
	}
	if webSearchNull.Valid {
		webSearchEnabled = webSearchNull.Bool
	}
	return model, toolsEnabled, webSearchEnabled, err
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

// GetWebSearchEnabled returns the web search setting for a chat.
func (r *Repository) GetWebSearchEnabled(ctx context.Context, chatID string) (bool, error) {
	var enabled sql.NullBool
	err := r.db.QueryRowContext(ctx, "SELECT web_search_enabled FROM chats WHERE id = $1", chatID).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled.Valid && enabled.Bool, nil
}

// SetWebSearchEnabled updates the web search setting for a chat.
func (r *Repository) SetWebSearchEnabled(ctx context.Context, chatID string, enabled bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chats SET web_search_enabled = $1, updated_at = NOW() WHERE id = $2
	`, enabled, chatID)
	return err
}

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
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &model, &m.TokenCount, &toolCallID, &toolCallsJSON, &responseID, &finishReason, &parentMessageID, &siblingIndex, &m.CreatedAt); err != nil {
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
			json.Unmarshal(toolCallsJSON, &msg.ToolCalls)
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

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, model, token_count, tool_call_id, parent_message_id, sibling_index, web_search)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, chat_id, role, content, model, token_count, tool_call_id, parent_message_id, sibling_index, created_at
	`, chatID, role, content,
		sql.NullString{String: model, Valid: model != ""},
		tokenCount,
		sql.NullString{String: toolCallID, Valid: toolCallID != ""},
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex,
		webSearchNull).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &sql.NullString{}, &msg.SiblingIndex, &msg.CreatedAt,
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

	var msg domain.Message
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, model, token_count, finish_reason, parent_message_id, sibling_index)
		VALUES ($1, 'assistant', $2, $3, $4, 'stop', $5, $6)
		RETURNING id, chat_id, role, content, model, token_count, finish_reason, parent_message_id, sibling_index, created_at
	`, chatID, content, sql.NullString{String: model, Valid: model != ""}, tokenCount,
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &sql.NullString{}, &msg.SiblingIndex, &msg.CreatedAt,
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

	var msg domain.Message
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, model, token_count, tool_calls, response_id, finish_reason, parent_message_id, sibling_index)
		VALUES ($1, 'assistant', $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, chat_id, role, content, model, token_count, finish_reason, parent_message_id, sibling_index, created_at
	`, chatID, content, sql.NullString{String: model, Valid: model != ""}, tokenCount, toolCallsJSON,
		sql.NullString{String: responseID, Valid: responseID != ""},
		sql.NullString{String: finishReason, Valid: finishReason != ""},
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &sql.NullString{}, &sql.NullString{}, &msg.SiblingIndex, &msg.CreatedAt,
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

	var msg domain.Message
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, tool_call_id, parent_message_id, sibling_index)
		VALUES ($1, 'tool', $2, $3, $4, $5)
		RETURNING id, chat_id, role, content, tool_call_id, parent_message_id, sibling_index, created_at
	`, chatID, result, toolCallID,
		sql.NullString{String: parentMessageID, Valid: parentMessageID != ""},
		siblingIndex).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &msg.ToolCallID, &sql.NullString{}, &msg.SiblingIndex, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	msg.ParentMessageID = parentMessageID
	return &msg, nil
}

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
	`, messageID).Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &model, &m.TokenCount, &toolCallID, &toolCallsJSON, &responseID, &finishReason, &parentMessageID, &siblingIndex, &m.CreatedAt)

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
		json.Unmarshal(toolCallsJSON, &m.ToolCalls)
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
	var rows *sql.Rows
	if msg.ParentMessageID == "" {
		// For root messages (no parent), return just this message
		return []domain.Message{*msg}, nil
	}

	rows, err = r.db.QueryContext(ctx, `
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
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &model, &m.TokenCount, &toolCallID, &toolCallsJSON, &responseID, &finishReason, &parentMessageID, &siblingIndex, &m.CreatedAt); err != nil {
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
		UPDATE chats SET active_leaf_message_id = $1, updated_at = NOW() WHERE id = $2
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

// Search Operations

// SearchResult represents a single search match.
type SearchResult struct {
	Chat      domain.Chat `json:"chat"`
	MessageID string      `json:"message_id,omitempty"`
	Snippet   string      `json:"snippet,omitempty"`
	Rank      float64     `json:"rank"`
	MatchType string      `json:"match_type"` // "chat_name" or "message_content"
}

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
	defer tx.Rollback()

	// Create the new chat
	var newChat domain.Chat
	err = tx.QueryRowContext(ctx, `
		INSERT INTO chats (name, model, view_mode)
		VALUES ($1, $2, 'bubble')
		RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, created_at, updated_at
	`, newName, model).Scan(
		&newChat.ID, &newChat.Name, &newChat.Preview, &newChat.Model, &newChat.ViewMode,
		&newChat.IsRead, &newChat.IsArchived, &newChat.IsStarred, &newChat.CreatedAt, &newChat.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create forked chat: %w", err)
	}
	newChat.LabelIDs = []string{}

	// Get the message ancestry path from the target message back to root
	// This ensures we only copy messages that are ancestors of the fork point
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

	// Collect all messages first - we must close rows before executing any other queries
	// on the same transaction connection (pq driver limitation)
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
		// Handle NULL tool_calls - empty byte slice should become NULL, not empty string
		var toolCallsArg interface{}
		if len(msg.toolCallsJSON) > 0 {
			toolCallsArg = msg.toolCallsJSON
		} else {
			toolCallsArg = nil
		}

		var newMsgID string
		err = tx.QueryRowContext(ctx, `
			INSERT INTO messages (chat_id, role, content, model, token_count, tool_call_id, tool_calls, response_id, finish_reason, parent_message_id, sibling_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0)
			RETURNING id
		`, newChat.ID, msg.role, msg.content, msg.model, msg.tokenCount, msg.toolCallID, toolCallsArg, msg.responseID, msg.finishReason,
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

// SearchChats performs full-text search across chat names and message content.
// Returns results ranked by relevance, with snippet highlighting for message matches.
func (r *Repository) SearchChats(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	// Convert user query to tsquery format (prefix matching for partial words)
	// This handles multi-word queries by adding :* to each word for prefix matching
	words := strings.Fields(query)
	var tsQueryParts []string
	for _, word := range words {
		// Escape special characters and add prefix matching
		escaped := strings.ReplaceAll(word, "'", "''")
		tsQueryParts = append(tsQueryParts, escaped+":*")
	}
	tsQuery := strings.Join(tsQueryParts, " & ")

	if limit <= 0 {
		limit = 20
	}

	// Search chat names and message content with ranking
	// Chat name matches rank higher than message matches
	searchSQL := `
		WITH chat_matches AS (
			SELECT
				c.id as chat_id,
				'' as message_id,
				ts_headline('english', c.name, to_tsquery('english', $1), 'StartSel=<mark>, StopSel=</mark>, MaxWords=50') as snippet,
				ts_rank(c.search_vector, to_tsquery('english', $1)) * 2 as rank,
				'chat_name' as match_type
			FROM chats c
			WHERE c.search_vector @@ to_tsquery('english', $1)
		),
		message_matches AS (
			SELECT
				m.chat_id,
				m.id as message_id,
				ts_headline('english', m.content, to_tsquery('english', $1), 'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15') as snippet,
				ts_rank(m.search_vector, to_tsquery('english', $1)) as rank,
				'message_content' as match_type
			FROM messages m
			WHERE m.search_vector @@ to_tsquery('english', $1)
				AND m.role IN ('user', 'assistant')
		),
		all_matches AS (
			SELECT * FROM chat_matches
			UNION ALL
			SELECT * FROM message_matches
		),
		ranked AS (
			SELECT DISTINCT ON (chat_id, match_type) *
			FROM all_matches
			ORDER BY chat_id, match_type, rank DESC
		)
		SELECT
			r.chat_id, r.message_id, r.snippet, r.rank, r.match_type,
			c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at,
			COALESCE(array_agg(cl.label_id) FILTER (WHERE cl.label_id IS NOT NULL), '{}') as label_ids
		FROM ranked r
		JOIN chats c ON c.id = r.chat_id
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		GROUP BY r.chat_id, r.message_id, r.snippet, r.rank, r.match_type,
			c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at
		ORDER BY r.rank DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, searchSQL, tsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search chats: %w", err)
	}
	defer rows.Close()

	results := make([]SearchResult, 0) // Always return [] instead of null in JSON
	for rows.Next() {
		var r SearchResult
		var messageID sql.NullString
		var labelIDs []byte

		if err := rows.Scan(
			&r.Chat.ID, &messageID, &r.Snippet, &r.Rank, &r.MatchType,
			&r.Chat.ID, &r.Chat.Name, &r.Chat.Preview, &r.Chat.Model, &r.Chat.ViewMode,
			&r.Chat.IsRead, &r.Chat.IsArchived, &r.Chat.IsStarred, &r.Chat.CreatedAt, &r.Chat.UpdatedAt,
			&labelIDs,
		); err != nil {
			continue
		}

		if messageID.Valid {
			r.MessageID = messageID.String
		}
		r.Chat.LabelIDs = parsePostgresArray(string(labelIDs))
		results = append(results, r)
	}

	return results, nil
}
