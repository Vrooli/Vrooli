// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains tool call record operations.
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"agent-inbox/domain"
)

// Tool Call Operations

// SaveToolCallRecord saves a tool call execution record.
func (r *Repository) SaveToolCallRecord(ctx context.Context, messageID string, record *domain.ToolCallRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tool_calls (id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			result = EXCLUDED.result,
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			error_message = EXCLUDED.error_message
	`, record.ID, messageID, record.ChatID, record.ToolName, record.Arguments, record.Result, record.Status, record.ScenarioName, record.ExternalRunID, record.StartedAt, record.CompletedAt, record.ErrorMessage)
	return err
}

// ListToolCallsForChat retrieves all tool calls for a chat.
func (r *Repository) ListToolCallsForChat(ctx context.Context, chatID string) ([]domain.ToolCallRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message
		FROM tool_calls WHERE chat_id = $1 ORDER BY started_at DESC
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tool calls: %w", err)
	}
	defer rows.Close()

	records := make([]domain.ToolCallRecord, 0) // Always return [] instead of null in JSON
	for rows.Next() {
		var record domain.ToolCallRecord
		var completedAt sql.NullTime
		var result, scenarioName, externalRunID, errorMessage sql.NullString
		if err := rows.Scan(&record.ID, &record.MessageID, &record.ChatID, &record.ToolName, &record.Arguments, &result, &record.Status, &scenarioName, &externalRunID, &record.StartedAt, &completedAt, &errorMessage); err != nil {
			continue
		}
		if result.Valid {
			record.Result = result.String
		}
		if scenarioName.Valid {
			record.ScenarioName = scenarioName.String
		}
		if externalRunID.Valid {
			record.ExternalRunID = externalRunID.String
		}
		if completedAt.Valid {
			record.CompletedAt = completedAt.Time
		}
		if errorMessage.Valid {
			record.ErrorMessage = errorMessage.String
		}
		records = append(records, record)
	}

	return records, nil
}

// ListOrphanedToolCalls finds tool calls in pending/running status that may have been
// orphaned by a server restart or crash. Used during startup reconciliation.
//
// TEMPORAL FLOW: This enables recovery from interruptions by identifying work
// that was in-progress when the server stopped.
func (r *Repository) ListOrphanedToolCalls(ctx context.Context) ([]domain.ToolCallRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message
		FROM tool_calls
		WHERE status IN ('pending', 'running')
		ORDER BY started_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list orphaned tool calls: %w", err)
	}
	defer rows.Close()

	records := make([]domain.ToolCallRecord, 0) // Always return [] instead of null in JSON
	for rows.Next() {
		var record domain.ToolCallRecord
		var completedAt sql.NullTime
		var result, scenarioName, externalRunID, errorMessage sql.NullString
		if err := rows.Scan(&record.ID, &record.MessageID, &record.ChatID, &record.ToolName, &record.Arguments, &result, &record.Status, &scenarioName, &externalRunID, &record.StartedAt, &completedAt, &errorMessage); err != nil {
			continue
		}
		if result.Valid {
			record.Result = result.String
		}
		if scenarioName.Valid {
			record.ScenarioName = scenarioName.String
		}
		if externalRunID.Valid {
			record.ExternalRunID = externalRunID.String
		}
		if completedAt.Valid {
			record.CompletedAt = completedAt.Time
		}
		if errorMessage.Valid {
			record.ErrorMessage = errorMessage.String
		}
		records = append(records, record)
	}

	return records, nil
}

// UpdateToolCallStatus updates a tool call's status and optionally sets an error message.
// Used during reconciliation to mark orphaned tool calls as cancelled.
func (r *Repository) UpdateToolCallStatus(ctx context.Context, id, status, errorMessage string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tool_calls
		SET status = $1, error_message = $2, completed_at = NOW()
		WHERE id = $3
	`, status, errorMessage, id)
	return err
}
