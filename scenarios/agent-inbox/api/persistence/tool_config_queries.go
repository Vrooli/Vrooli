// Package persistence provides database operations for the Agent Inbox scenario.
//
// This file provides query helpers and lookup operations for tool configurations.
package persistence

import (
	"agent-inbox/domain"
	"context"
	"database/sql"
	"fmt"
)

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
		rec, err := scanToolCallRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending approval: %w", err)
		}
		records = append(records, rec)
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
	var completedAt sqliteNullTime
	var scenarioName, externalRunID sql.NullString

	err := r.db.QueryRowContext(ctx, query, toolCallID).Scan(
		&rec.ID, &rec.MessageID, &rec.ChatID, &rec.ToolName,
		&rec.Arguments, &result, &rec.Status,
		&scenarioName, &externalRunID, scanTime(&rec.StartedAt), &completedAt, &errorMessage,
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

// scanToolCallRecord scans a tool call record from a rows scanner.
func scanToolCallRecord(rows *sql.Rows) (*domain.ToolCallRecord, error) {
	var rec domain.ToolCallRecord
	var result, errorMessage sql.NullString
	var completedAt sqliteNullTime
	var scenarioName, externalRunID sql.NullString

	if err := rows.Scan(
		&rec.ID, &rec.MessageID, &rec.ChatID, &rec.ToolName,
		&rec.Arguments, &result, &rec.Status,
		&scenarioName, &externalRunID, scanTime(&rec.StartedAt), &completedAt, &errorMessage,
	); err != nil {
		return nil, err
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
