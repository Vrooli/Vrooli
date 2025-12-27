// Package persistence provides database operations for the Agent Inbox scenario.
//
// This file provides tool configuration persistence operations.
// Tool configurations store user preferences for enabling/disabling tools
// at both global and per-chat levels.
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"agent-inbox/domain"
)

// SaveToolConfiguration upserts a tool configuration.
// If chatID is empty, this is a global configuration.
func (r *Repository) SaveToolConfiguration(ctx context.Context, cfg *domain.ToolConfiguration) error {
	now := time.Now()

	// Handle null chat_id for global configurations
	var chatID interface{}
	if cfg.ChatID == "" {
		chatID = nil
	} else {
		chatID = cfg.ChatID
	}

	query := `
		INSERT INTO tool_configurations (chat_id, scenario, tool_name, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (chat_id, scenario, tool_name)
		DO UPDATE SET enabled = $4, updated_at = $5
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		chatID, cfg.Scenario, cfg.ToolName, cfg.Enabled, now,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save tool configuration: %w", err)
	}

	return nil
}

// GetToolConfiguration retrieves a specific tool configuration.
// Pass empty chatID for global configuration.
func (r *Repository) GetToolConfiguration(ctx context.Context, chatID, scenario, toolName string) (*domain.ToolConfiguration, error) {
	query := `
		SELECT id, chat_id, scenario, tool_name, enabled, created_at, updated_at
		FROM tool_configurations
		WHERE (chat_id = $1 OR ($1 = '' AND chat_id IS NULL))
		  AND scenario = $2
		  AND tool_name = $3`

	var chatIDValue interface{}
	if chatID == "" {
		chatIDValue = nil
	} else {
		chatIDValue = chatID
	}

	var cfg domain.ToolConfiguration
	var nullChatID sql.NullString

	err := r.db.QueryRowContext(ctx, query, chatIDValue, scenario, toolName).Scan(
		&cfg.ID, &nullChatID, &cfg.Scenario, &cfg.ToolName,
		&cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tool configuration: %w", err)
	}

	if nullChatID.Valid {
		cfg.ChatID = nullChatID.String
	}

	return &cfg, nil
}

// ListToolConfigurations retrieves all tool configurations.
// Pass empty chatID to get only global configurations.
// Pass a chatID to get configurations for that chat (including global).
func (r *Repository) ListToolConfigurations(ctx context.Context, chatID string) ([]*domain.ToolConfiguration, error) {
	var query string
	var args []interface{}

	if chatID == "" {
		// Global configurations only
		query = `
			SELECT id, chat_id, scenario, tool_name, enabled, created_at, updated_at
			FROM tool_configurations
			WHERE chat_id IS NULL
			ORDER BY scenario, tool_name`
	} else {
		// Both global and chat-specific configurations
		query = `
			SELECT id, chat_id, scenario, tool_name, enabled, created_at, updated_at
			FROM tool_configurations
			WHERE chat_id IS NULL OR chat_id = $1
			ORDER BY scenario, tool_name, chat_id NULLS FIRST`
		args = append(args, chatID)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tool configurations: %w", err)
	}
	defer rows.Close()

	var configs []*domain.ToolConfiguration
	for rows.Next() {
		var cfg domain.ToolConfiguration
		var nullChatID sql.NullString

		if err := rows.Scan(
			&cfg.ID, &nullChatID, &cfg.Scenario, &cfg.ToolName,
			&cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tool configuration: %w", err)
		}

		if nullChatID.Valid {
			cfg.ChatID = nullChatID.String
		}

		configs = append(configs, &cfg)
	}

	return configs, rows.Err()
}

// DeleteToolConfiguration removes a tool configuration.
// Pass empty chatID for global configuration.
func (r *Repository) DeleteToolConfiguration(ctx context.Context, chatID, scenario, toolName string) error {
	query := `
		DELETE FROM tool_configurations
		WHERE (chat_id = $1 OR ($1 = '' AND chat_id IS NULL))
		  AND scenario = $2
		  AND tool_name = $3`

	var chatIDValue interface{}
	if chatID == "" {
		chatIDValue = nil
	} else {
		chatIDValue = chatID
	}

	_, err := r.db.ExecContext(ctx, query, chatIDValue, scenario, toolName)
	if err != nil {
		return fmt.Errorf("failed to delete tool configuration: %w", err)
	}

	return nil
}

// GetEffectiveToolEnabled determines if a tool is enabled for a chat.
// It checks chat-specific config first, then falls back to global config,
// then to the tool's default enabled state.
func (r *Repository) GetEffectiveToolEnabled(ctx context.Context, chatID, scenario, toolName string, defaultEnabled bool) (bool, domain.ToolConfigurationScope, error) {
	// First check for chat-specific configuration
	if chatID != "" {
		cfg, err := r.GetToolConfiguration(ctx, chatID, scenario, toolName)
		if err != nil {
			return false, "", err
		}
		if cfg != nil {
			return cfg.Enabled, domain.ScopeChat, nil
		}
	}

	// Fall back to global configuration
	cfg, err := r.GetToolConfiguration(ctx, "", scenario, toolName)
	if err != nil {
		return false, "", err
	}
	if cfg != nil {
		return cfg.Enabled, domain.ScopeGlobal, nil
	}

	// Use tool's default
	return defaultEnabled, "", nil
}

// BulkSaveToolConfigurations saves multiple tool configurations in a transaction.
func (r *Repository) BulkSaveToolConfigurations(ctx context.Context, configs []*domain.ToolConfiguration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tool_configurations (chat_id, scenario, tool_name, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (chat_id, scenario, tool_name)
		DO UPDATE SET enabled = $4, updated_at = $5`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, cfg := range configs {
		var chatID interface{}
		if cfg.ChatID == "" {
			chatID = nil
		} else {
			chatID = cfg.ChatID
		}

		if _, err := stmt.ExecContext(ctx, chatID, cfg.Scenario, cfg.ToolName, cfg.Enabled, now); err != nil {
			return fmt.Errorf("failed to save tool configuration for %s/%s: %w", cfg.Scenario, cfg.ToolName, err)
		}
	}

	return tx.Commit()
}
