// Package persistence provides database operations for the Agent Inbox scenario.
//
// This file provides tool configuration persistence operations.
// Tool configurations store user preferences for enabling/disabling tools
// at both global and per-chat levels.
package persistence

import (
	"agent-inbox/domain"
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SaveToolConfiguration upserts a tool configuration.
// If chatID is empty, this is a global configuration.
func (r *Repository) SaveToolConfiguration(ctx context.Context, cfg *domain.ToolConfiguration) error {
	now := time.Now()

	// Handle null approval_override (empty string means default)
	var approvalOverride interface{}
	if cfg.ApprovalOverride == "" {
		approvalOverride = nil
	} else {
		approvalOverride = string(cfg.ApprovalOverride)
	}

	var query string
	var args []interface{}

	if cfg.ChatID == "" {
		// Global configuration - use partial unique index for conflict detection
		id := newID()
		query = `
			INSERT INTO tool_configurations (id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
			VALUES ($1, NULL, $2, $3, $4, $5, $6, $6)
			ON CONFLICT (scenario, tool_name) WHERE chat_id IS NULL
			DO UPDATE SET enabled = $4, approval_override = $5, updated_at = $6
			RETURNING id, created_at, updated_at`
		args = []interface{}{id, cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now}
	} else {
		// Chat-specific configuration - use regular unique constraint
		id := newID()
		query = `
			INSERT INTO tool_configurations (id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
			ON CONFLICT (chat_id, scenario, tool_name)
			DO UPDATE SET enabled = $5, approval_override = $6, updated_at = $7
			RETURNING id, created_at, updated_at`
		args = []interface{}{id, cfg.ChatID, cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now}
	}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&cfg.ID, scanTime(&cfg.CreatedAt), scanTime(&cfg.UpdatedAt))
	if err != nil {
		return fmt.Errorf("failed to save tool configuration: %w", err)
	}

	return nil
}

// GetToolConfiguration retrieves a specific tool configuration.
// Pass empty chatID for global configuration.
func (r *Repository) GetToolConfiguration(ctx context.Context, chatID, scenario, toolName string) (*domain.ToolConfiguration, error) {
	query := `
		SELECT id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at
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
	var nullApprovalOverride sql.NullString

	err := r.db.QueryRowContext(ctx, query, chatIDValue, scenario, toolName).Scan(
		&cfg.ID, &nullChatID, &cfg.Scenario, &cfg.ToolName,
		&cfg.Enabled, &nullApprovalOverride, scanTime(&cfg.CreatedAt), scanTime(&cfg.UpdatedAt),
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
	if nullApprovalOverride.Valid {
		cfg.ApprovalOverride = domain.ApprovalOverride(nullApprovalOverride.String)
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
			SELECT id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at
			FROM tool_configurations
			WHERE chat_id IS NULL
			ORDER BY scenario, tool_name`
	} else {
		// Both global and chat-specific configurations
		// SQLite does not support NULLS FIRST; use CASE expression instead.
		query = `
			SELECT id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at
			FROM tool_configurations
			WHERE chat_id IS NULL OR chat_id = $1
			ORDER BY scenario, tool_name, (CASE WHEN chat_id IS NULL THEN 0 ELSE 1 END)`
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
		var nullApprovalOverride sql.NullString

		if err := rows.Scan(
			&cfg.ID, &nullChatID, &cfg.Scenario, &cfg.ToolName,
			&cfg.Enabled, &nullApprovalOverride, scanTime(&cfg.CreatedAt), scanTime(&cfg.UpdatedAt),
		); err != nil {
			return nil, fmt.Errorf("failed to scan tool configuration: %w", err)
		}

		if nullChatID.Valid {
			cfg.ChatID = nullChatID.String
		}
		if nullApprovalOverride.Valid {
			cfg.ApprovalOverride = domain.ApprovalOverride(nullApprovalOverride.String)
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

// BulkSaveToolConfigurations saves multiple tool configurations in a transaction.
func (r *Repository) BulkSaveToolConfigurations(ctx context.Context, configs []*domain.ToolConfiguration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Prepare separate statements for global vs chat-specific configs
	globalStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tool_configurations (id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
		VALUES ($1, NULL, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (scenario, tool_name) WHERE chat_id IS NULL
		DO UPDATE SET enabled = $4, approval_override = $5, updated_at = $6`)
	if err != nil {
		return fmt.Errorf("failed to prepare global statement: %w", err)
	}
	defer globalStmt.Close()

	chatStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tool_configurations (id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (chat_id, scenario, tool_name)
		DO UPDATE SET enabled = $5, approval_override = $6, updated_at = $7`)
	if err != nil {
		return fmt.Errorf("failed to prepare chat statement: %w", err)
	}
	defer chatStmt.Close()

	now := time.Now()
	for _, cfg := range configs {
		var approvalOverride interface{}
		if cfg.ApprovalOverride == "" {
			approvalOverride = nil
		} else {
			approvalOverride = string(cfg.ApprovalOverride)
		}

		id := newID()
		if cfg.ChatID == "" {
			// Global configuration
			if _, err := globalStmt.ExecContext(ctx, id, cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now); err != nil {
				return fmt.Errorf("failed to save global tool configuration for %s/%s: %w", cfg.Scenario, cfg.ToolName, err)
			}
		} else {
			// Chat-specific configuration
			if _, err := chatStmt.ExecContext(ctx, id, cfg.ChatID, cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now); err != nil {
				return fmt.Errorf("failed to save chat tool configuration for %s/%s: %w", cfg.Scenario, cfg.ToolName, err)
			}
		}
	}

	return tx.Commit()
}

// SetToolApprovalOverride updates only the approval_override for a tool configuration.
// Pass empty chatID for global configuration.
// Pass empty override to reset to default (use tool metadata).
func (r *Repository) SetToolApprovalOverride(ctx context.Context, chatID, scenario, toolName string, override domain.ApprovalOverride) error {
	now := time.Now()

	var chatIDValue interface{}
	if chatID == "" {
		chatIDValue = nil
	} else {
		chatIDValue = chatID
	}

	var approvalOverride interface{}
	if override == "" {
		approvalOverride = nil
	} else {
		approvalOverride = string(override)
	}

	// First try to update existing
	result, err := r.db.ExecContext(ctx, `
		UPDATE tool_configurations
		SET approval_override = $4, updated_at = $5
		WHERE (chat_id = $1 OR ($1 IS NULL AND chat_id IS NULL))
		  AND scenario = $2
		  AND tool_name = $3`,
		chatIDValue, scenario, toolName, approvalOverride, now)
	if err != nil {
		return fmt.Errorf("failed to update approval override: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	// No existing config, create one with default enabled=true
	id := newID()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tool_configurations (id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 1, $5, $6, $6)`,
		id, chatIDValue, scenario, toolName, approvalOverride, now)
	if err != nil {
		return fmt.Errorf("failed to create tool configuration with approval override: %w", err)
	}

	return nil
}
