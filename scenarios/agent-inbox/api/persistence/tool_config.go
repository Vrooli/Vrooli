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
//
// NOTE: PostgreSQL treats NULL values as distinct in unique constraints by default.
// This means ON CONFLICT (chat_id, scenario, tool_name) won't match rows where chat_id IS NULL.
// We handle this by using different queries for global vs chat-specific configs:
// - Global (NULL chat_id): Use ON CONFLICT with partial index on (scenario, tool_name) WHERE chat_id IS NULL
// - Chat-specific: Use regular ON CONFLICT on the composite unique constraint
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
		// The partial index idx_tool_configurations_global_unique is defined as:
		// UNIQUE(scenario, tool_name) WHERE chat_id IS NULL
		query = `
			INSERT INTO tool_configurations (chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
			VALUES (NULL, $1, $2, $3, $4, $5, $5)
			ON CONFLICT (scenario, tool_name) WHERE chat_id IS NULL
			DO UPDATE SET enabled = $3, approval_override = $4, updated_at = $5
			RETURNING id, created_at, updated_at`
		args = []interface{}{cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now}
	} else {
		// Chat-specific configuration - use regular unique constraint
		query = `
			INSERT INTO tool_configurations (chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6)
			ON CONFLICT (chat_id, scenario, tool_name)
			DO UPDATE SET enabled = $4, approval_override = $5, updated_at = $6
			RETURNING id, created_at, updated_at`
		args = []interface{}{cfg.ChatID, cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now}
	}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
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
		&cfg.Enabled, &nullApprovalOverride, &cfg.CreatedAt, &cfg.UpdatedAt,
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
		query = `
			SELECT id, chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at
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
		var nullApprovalOverride sql.NullString

		if err := rows.Scan(
			&cfg.ID, &nullChatID, &cfg.Scenario, &cfg.ToolName,
			&cfg.Enabled, &nullApprovalOverride, &cfg.CreatedAt, &cfg.UpdatedAt,
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
// Uses separate queries for global vs chat-specific configs to handle PostgreSQL NULL uniqueness.
func (r *Repository) BulkSaveToolConfigurations(ctx context.Context, configs []*domain.ToolConfiguration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare separate statements for global vs chat-specific configs
	globalStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tool_configurations (chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
		VALUES (NULL, $1, $2, $3, $4, $5, $5)
		ON CONFLICT (scenario, tool_name) WHERE chat_id IS NULL
		DO UPDATE SET enabled = $3, approval_override = $4, updated_at = $5`)
	if err != nil {
		return fmt.Errorf("failed to prepare global statement: %w", err)
	}
	defer globalStmt.Close()

	chatStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tool_configurations (chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (chat_id, scenario, tool_name)
		DO UPDATE SET enabled = $4, approval_override = $5, updated_at = $6`)
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

		if cfg.ChatID == "" {
			// Global configuration
			if _, err := globalStmt.ExecContext(ctx, cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now); err != nil {
				return fmt.Errorf("failed to save global tool configuration for %s/%s: %w", cfg.Scenario, cfg.ToolName, err)
			}
		} else {
			// Chat-specific configuration
			if _, err := chatStmt.ExecContext(ctx, cfg.ChatID, cfg.Scenario, cfg.ToolName, cfg.Enabled, approvalOverride, now); err != nil {
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
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tool_configurations (chat_id, scenario, tool_name, enabled, approval_override, created_at, updated_at)
		VALUES ($1, $2, $3, true, $4, $5, $5)`,
		chatIDValue, scenario, toolName, approvalOverride, now)
	if err != nil {
		return fmt.Errorf("failed to create tool configuration with approval override: %w", err)
	}

	return nil
}

// GetEffectiveToolApproval determines if a tool requires approval.
// It checks: YOLO mode > chat-specific override > global override > tool metadata default.
// Returns the effective approval requirement and its source.
func (r *Repository) GetEffectiveToolApproval(ctx context.Context, chatID, scenario, toolName string, metadataDefault bool) (bool, domain.ToolConfigurationScope, error) {
	// First check for chat-specific configuration
	if chatID != "" {
		cfg, err := r.GetToolConfiguration(ctx, chatID, scenario, toolName)
		if err != nil {
			return false, "", err
		}
		if cfg != nil && cfg.ApprovalOverride != "" {
			requiresApproval := cfg.ApprovalOverride == domain.ApprovalRequire
			return requiresApproval, domain.ScopeChat, nil
		}
	}

	// Fall back to global configuration
	cfg, err := r.GetToolConfiguration(ctx, "", scenario, toolName)
	if err != nil {
		return false, "", err
	}
	if cfg != nil && cfg.ApprovalOverride != "" {
		requiresApproval := cfg.ApprovalOverride == domain.ApprovalRequire
		return requiresApproval, domain.ScopeGlobal, nil
	}

	// Use tool's default metadata
	return metadataDefault, "", nil
}

// GetPendingApprovals returns all tool calls pending approval for a chat.
func (r *Repository) GetPendingApprovals(ctx context.Context, chatID string) ([]*domain.ToolCallRecord, error) {
	query := `
		SELECT id, message_id, chat_id, tool_name, arguments, result, status,
		       scenario_name, external_run_id, started_at, completed_at, error_message
		FROM tool_calls
		WHERE chat_id = $1 AND status = $2
		ORDER BY started_at ASC`

	rows, err := r.db.QueryContext(ctx, query, chatID, domain.StatusPendingApproval)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}
	defer rows.Close()

	var records []*domain.ToolCallRecord
	for rows.Next() {
		var rec domain.ToolCallRecord
		var result, errorMessage sql.NullString
		var completedAt sql.NullTime
		var scenarioName, externalRunID sql.NullString

		if err := rows.Scan(
			&rec.ID, &rec.MessageID, &rec.ChatID, &rec.ToolName,
			&rec.Arguments, &result, &rec.Status,
			&scenarioName, &externalRunID, &rec.StartedAt, &completedAt, &errorMessage,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pending approval: %w", err)
		}

		if result.Valid {
			rec.Result = result.String
		}
		if errorMessage.Valid {
			rec.ErrorMessage = errorMessage.String
		}
		if completedAt.Valid {
			rec.CompletedAt = completedAt.Time
		}
		if scenarioName.Valid {
			rec.ScenarioName = scenarioName.String
		}
		if externalRunID.Valid {
			rec.ExternalRunID = externalRunID.String
		}

		records = append(records, &rec)
	}

	return records, rows.Err()
}

// GetToolCallByID retrieves a tool call by its ID.
func (r *Repository) GetToolCallByID(ctx context.Context, toolCallID string) (*domain.ToolCallRecord, error) {
	query := `
		SELECT id, message_id, chat_id, tool_name, arguments, result, status,
		       scenario_name, external_run_id, started_at, completed_at, error_message
		FROM tool_calls
		WHERE id = $1`

	var rec domain.ToolCallRecord
	var result, errorMessage sql.NullString
	var completedAt sql.NullTime
	var scenarioName, externalRunID sql.NullString

	err := r.db.QueryRowContext(ctx, query, toolCallID).Scan(
		&rec.ID, &rec.MessageID, &rec.ChatID, &rec.ToolName,
		&rec.Arguments, &result, &rec.Status,
		&scenarioName, &externalRunID, &rec.StartedAt, &completedAt, &errorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tool call: %w", err)
	}

	if result.Valid {
		rec.Result = result.String
	}
	if errorMessage.Valid {
		rec.ErrorMessage = errorMessage.String
	}
	if completedAt.Valid {
		rec.CompletedAt = completedAt.Time
	}
	if scenarioName.Valid {
		rec.ScenarioName = scenarioName.String
	}
	if externalRunID.Valid {
		rec.ExternalRunID = externalRunID.String
	}

	return &rec, nil
}
