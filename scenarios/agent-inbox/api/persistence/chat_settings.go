// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains chat settings, template, and agent mode operations.
package persistence

import (
	"context"
	"database/sql"

	"agent-inbox/domain"
)

// GetChatSettings retrieves just the model for a chat.
func (r *Repository) GetChatSettings(ctx context.Context, chatID string) (model string, err error) {
	err = r.db.QueryRowContext(ctx, "SELECT model FROM chats WHERE id = $1", chatID).Scan(&model)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return model, err
}

// GetChatSettingsWithWebSearch retrieves model and web_search_enabled for a chat.
func (r *Repository) GetChatSettingsWithWebSearch(ctx context.Context, chatID string) (model string, webSearchEnabled bool, err error) {
	var webSearchNull sql.NullBool
	err = r.db.QueryRowContext(ctx, "SELECT model, web_search_enabled FROM chats WHERE id = $1", chatID).Scan(&model, &webSearchNull)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if webSearchNull.Valid {
		webSearchEnabled = webSearchNull.Bool
	}
	return model, webSearchEnabled, err
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
		UPDATE chats SET web_search_enabled = $1, updated_at = datetime('now') WHERE id = $2
	`, enabled, chatID)
	return err
}

// Active Template Operations

// SetActiveTemplate sets the active template for a chat.
func (r *Repository) SetActiveTemplate(ctx context.Context, chatID, templateID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chats SET active_template_id = $1, updated_at = datetime('now') WHERE id = $2
	`, sql.NullString{String: templateID, Valid: templateID != ""}, chatID)
	return err
}

// ClearActiveTemplate removes the active template state from a chat.
// Called when the user manually deactivates a template.
func (r *Repository) ClearActiveTemplate(ctx context.Context, chatID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chats SET active_template_id = NULL, updated_at = datetime('now') WHERE id = $1
	`, chatID)
	return err
}

// Agent Mode Operations

// SetAgentMode updates the chat to agent mode and stores the task/run IDs.
func (r *Repository) SetAgentMode(ctx context.Context, chatID, taskID, runID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chats SET chat_mode = $1, agent_task_id = $2, agent_run_id = $3, updated_at = datetime('now') WHERE id = $4
	`, domain.ChatModeAgent, sql.NullString{String: taskID, Valid: taskID != ""}, sql.NullString{String: runID, Valid: runID != ""}, chatID)
	return err
}

// UpdateAgentRunID updates just the run ID for an agent mode chat.
// Used when continuing a chat creates a new run.
func (r *Repository) UpdateAgentRunID(ctx context.Context, chatID, runID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chats SET agent_run_id = $1, updated_at = datetime('now') WHERE id = $2
	`, sql.NullString{String: runID, Valid: runID != ""}, chatID)
	return err
}

// ClearAgentMode resets a chat back to LLM mode.
func (r *Repository) ClearAgentMode(ctx context.Context, chatID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chats SET chat_mode = $1, agent_task_id = NULL, agent_run_id = NULL, updated_at = datetime('now') WHERE id = $2
	`, domain.ChatModeLLM, chatID)
	return err
}

// GetAgentMode returns the chat mode and agent IDs for a chat.
func (r *Repository) GetAgentMode(ctx context.Context, chatID string) (chatMode, taskID, runID string, err error) {
	var chatModeNull, taskIDNull, runIDNull sql.NullString
	err = r.db.QueryRowContext(ctx, `SELECT chat_mode, agent_task_id, agent_run_id FROM chats WHERE id = $1`, chatID).Scan(&chatModeNull, &taskIDNull, &runIDNull)
	if err == sql.ErrNoRows {
		return domain.ChatModeLLM, "", "", nil
	}
	if err != nil {
		return "", "", "", err
	}
	if chatModeNull.Valid {
		chatMode = chatModeNull.String
	} else {
		chatMode = domain.ChatModeLLM
	}
	if taskIDNull.Valid {
		taskID = taskIDNull.String
	}
	if runIDNull.Valid {
		runID = runIDNull.String
	}
	return chatMode, taskID, runID, nil
}
