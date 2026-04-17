// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains core chat CRUD operations.
package persistence

import (
	"agent-inbox/domain"
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Chat Operations

// ListChats returns all chats matching the given filters.
func (r *Repository) ListChats(ctx context.Context, archived, starred bool) ([]domain.Chat, error) {
	query := `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.tools_enabled, c.chat_mode, c.agent_run_id, c.agent_task_id, c.created_at, c.updated_at,
			COALESCE(GROUP_CONCAT(cl.label_id), '') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE c.is_archived = $1
	`
	args := []interface{}{archived}

	if starred {
		query += " AND c.is_starred = 1"
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
		var labelIDs string
		var chatMode sql.NullString
		var agentRunID, agentTaskID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Preview, &c.Model, &c.ViewMode, &c.IsRead, &c.IsArchived, &c.IsStarred, &c.ToolsEnabled, &chatMode, &agentRunID, &agentTaskID, scanTime(&c.CreatedAt), scanTime(&c.UpdatedAt), &labelIDs); err != nil {
			continue
		}
		c.LabelIDs = parseArrayString(labelIDs)
		if chatMode.Valid {
			c.ChatMode = chatMode.String
		} else {
			c.ChatMode = domain.ChatModeLLM // Default to llm mode
		}
		if agentRunID.Valid {
			c.AgentRunID = agentRunID.String
		}
		if agentTaskID.Valid {
			c.AgentTaskID = agentTaskID.String
		}
		chats = append(chats, c)
	}

	return chats, nil
}

// GetChat retrieves a single chat by ID.
func (r *Repository) GetChat(ctx context.Context, chatID string) (*domain.Chat, error) {
	var chat domain.Chat
	var labelIDs string
	var activeLeafMessageID sql.NullString
	var webSearchEnabled sql.NullBool
	var activeTemplateID sql.NullString
	var activeTemplateToolIDs sql.NullString
	var chatMode sql.NullString
	var agentRunID, agentTaskID sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.tools_enabled, c.web_search_enabled, c.active_leaf_message_id, c.active_template_id, COALESCE(c.active_template_tool_ids, '[]'), c.chat_mode, c.agent_run_id, c.agent_task_id, c.created_at, c.updated_at,
			COALESCE(GROUP_CONCAT(cl.label_id), '') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE c.id = $1
		GROUP BY c.id
	`, chatID).Scan(&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode, &chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.ToolsEnabled, &webSearchEnabled, &activeLeafMessageID, &activeTemplateID, &activeTemplateToolIDs, &chatMode, &agentRunID, &agentTaskID, scanTime(&chat.CreatedAt), scanTime(&chat.UpdatedAt), &labelIDs)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	chat.LabelIDs = parseArrayString(labelIDs)
	if activeLeafMessageID.Valid {
		chat.ActiveLeafMessageID = activeLeafMessageID.String
	}
	if webSearchEnabled.Valid {
		chat.WebSearchEnabled = webSearchEnabled.Bool
	}
	if activeTemplateID.Valid {
		chat.ActiveTemplateID = activeTemplateID.String
	}
	if activeTemplateToolIDs.Valid {
		chat.ActiveTemplateToolIDs = parseArrayString(activeTemplateToolIDs.String)
	} else {
		chat.ActiveTemplateToolIDs = []string{}
	}
	if chatMode.Valid {
		chat.ChatMode = chatMode.String
	} else {
		chat.ChatMode = domain.ChatModeLLM // Default to llm mode
	}
	if agentRunID.Valid {
		chat.AgentRunID = agentRunID.String
	}
	if agentTaskID.Valid {
		chat.AgentTaskID = agentTaskID.String
	}
	return &chat, nil
}

// CreateChat creates a new chat with the given parameters.
// chatMode defaults to "llm" if empty.
func (r *Repository) CreateChat(ctx context.Context, name, model, viewMode, chatMode string) (*domain.Chat, error) {
	if chatMode == "" {
		chatMode = domain.ChatModeLLM
	}
	if viewMode == "" {
		viewMode = domain.ViewModeBubble
	}
	id := newID()
	var chat domain.Chat
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO chats (id, name, model, view_mode, chat_mode)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, tools_enabled, chat_mode, created_at, updated_at
	`, id, name, model, viewMode, chatMode).Scan(
		&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode,
		&chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.ToolsEnabled, &chat.ChatMode, scanTime(&chat.CreatedAt), scanTime(&chat.UpdatedAt),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}
	chat.LabelIDs = []string{}
	return &chat, nil
}

// UpdateChat updates a chat's name, model, and/or tools_enabled.
func (r *Repository) UpdateChat(ctx context.Context, chatID string, name, model *string, toolsEnabled *bool) (*domain.Chat, error) {
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
	if toolsEnabled != nil {
		updates = append(updates, fmt.Sprintf("tools_enabled = $%d", argNum))
		args = append(args, *toolsEnabled)
		argNum++
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = datetime('now')")
	args = append(args, chatID)

	query := fmt.Sprintf("UPDATE chats SET %s WHERE id = $%d RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, tools_enabled, created_at, updated_at",
		strings.Join(updates, ", "), argNum)

	var chat domain.Chat
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode,
		&chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.ToolsEnabled, scanTime(&chat.CreatedAt), scanTime(&chat.UpdatedAt),
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

// DeleteArchivedChats removes all archived chats and returns the count deleted.
func (r *Repository) DeleteArchivedChats(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM chats WHERE is_archived = 1")
	if err != nil {
		return 0, fmt.Errorf("failed to delete archived chats: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// MarkAllChatsRead marks all unread chats as read and returns the count updated.
func (r *Repository) MarkAllChatsRead(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE chats SET is_read = 1, updated_at = datetime('now') WHERE is_read = 0")
	if err != nil {
		return 0, fmt.Errorf("failed to mark all chats as read: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
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
		query = fmt.Sprintf("UPDATE chats SET %s = $1, updated_at = datetime('now') WHERE id = $2 RETURNING %s", field, field)
		err = r.db.QueryRowContext(ctx, query, *value, chatID).Scan(&newValue)
	} else {
		query = fmt.Sprintf("UPDATE chats SET %s = NOT %s, updated_at = datetime('now') WHERE id = $1 RETURNING %s", field, field, field)
		err = r.db.QueryRowContext(ctx, query, chatID).Scan(&newValue)
	}

	if err == sql.ErrNoRows {
		return false, fmt.Errorf("chat not found")
	}
	return newValue, err
}

// UpdateChatPreview updates the preview text and optionally marks as unread.
func (r *Repository) UpdateChatPreview(ctx context.Context, chatID, preview string, markUnread bool) error {
	query := "UPDATE chats SET preview = $1, updated_at = datetime('now')"
	if markUnread {
		query += ", is_read = 0"
	}
	query += " WHERE id = $2"
	_, err := r.db.ExecContext(ctx, query, preview, chatID)
	return err
}
