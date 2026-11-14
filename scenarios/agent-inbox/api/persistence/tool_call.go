// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains tool call record operations.
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"agent-inbox/domain"
)

// Tool Call Operations

// SaveToolCallRecord saves a tool call execution record.
func (r *Repository) SaveToolCallRecord(ctx context.Context, messageID string, record *domain.ToolCallRecord) error {
	return saveToolCallRecord(ctx, r.db, messageID, record)
}

type toolCallRecordWriter interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func saveToolCallRecord(ctx context.Context, writer toolCallRecordWriter, messageID string, record *domain.ToolCallRecord) error {
	if record == nil {
		return fmt.Errorf("cannot save nil tool call record")
	}
	if !record.Status.Valid() {
		return fmt.Errorf("cannot persist invalid tool call status %q", record.Status)
	}
	// Ensure arguments is valid JSON (empty string is not valid JSON)
	arguments := record.Arguments
	if arguments == "" {
		arguments = "{}"
	}

	// Ensure result is valid JSON or null
	var result interface{}
	if record.Result == "" {
		result = nil
	} else {
		result = record.Result
	}

	allowed := make([]domain.ToolCallStatus, 0, 8)
	for _, prior := range allToolCallStatuses {
		if prior.CanTransitionTo(record.Status) {
			allowed = append(allowed, prior)
		}
	}
	placeholders := make([]string, len(allowed))
	args := []any{record.ID, messageID, record.ChatID, record.ToolName, arguments, result, record.Status, record.ScenarioName, record.ExternalRunID, record.StartedAt.UTC(), record.CompletedAt.UTC(), record.ErrorMessage}
	for i, status := range allowed {
		placeholders[i] = fmt.Sprintf("$%d", 13+i)
		args = append(args, status)
	}
	query := `
		INSERT INTO tool_calls (id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			result = EXCLUDED.result,
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			error_message = EXCLUDED.error_message
		WHERE tool_calls.status IN (` + strings.Join(placeholders, ",") + `)
	`
	res, err := writer.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return fmt.Errorf("tool call %s rejected status transition to %q", record.ID, record.Status)
	}
	return nil
}

var allToolCallStatuses = []domain.ToolCallStatus{
	domain.StatusPending, domain.StatusPendingApproval, domain.StatusApproved, domain.StatusRejected,
	domain.StatusRunning, domain.StatusCompleted, domain.StatusFailed, domain.StatusCancelled,
}

// ListToolCallsForChat retrieves all tool calls for a chat.
func (r *Repository) ListToolCallsForChat(ctx context.Context, chatID string) ([]domain.ToolCallRecord, error) {
	return r.listToolCalls(ctx, "tool calls", `
		SELECT id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message
		FROM tool_calls WHERE chat_id = $1 ORDER BY started_at DESC
	`, chatID)
}

// ListOrphanedToolCalls finds tool calls in pending/running status that may have been
// orphaned by a server restart or crash. Used during startup reconciliation.
//
// TEMPORAL FLOW: This enables recovery from interruptions by identifying work
// that was in-progress when the server stopped.
func (r *Repository) ListOrphanedToolCalls(ctx context.Context) ([]domain.ToolCallRecord, error) {
	return r.listToolCalls(ctx, "orphaned tool calls", `
		SELECT id, message_id, chat_id, tool_name, arguments, result, status, scenario_name, external_run_id, started_at, completed_at, error_message
		FROM tool_calls
		WHERE status IN ('pending', 'running')
		ORDER BY started_at ASC
	`)
}

func (r *Repository) listToolCalls(ctx context.Context, operation, query string, args ...any) ([]domain.ToolCallRecord, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", operation, err)
	}
	defer rows.Close()

	records := make([]domain.ToolCallRecord, 0) // Always return [] instead of null in JSON
	for rows.Next() {
		var record domain.ToolCallRecord
		var completedAt sqliteNullTime
		var result, scenarioName, externalRunID, errorMessage sql.NullString
		if err := rows.Scan(&record.ID, &record.MessageID, &record.ChatID, &record.ToolName, &record.Arguments, &result, &record.Status, &scenarioName, &externalRunID, scanTime(&record.StartedAt), &completedAt, &errorMessage); err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", operation, err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading %s: %w", operation, err)
	}
	return records, nil
}

// UpdateToolCallStatus updates a tool call's status and optionally sets an error message.
// Used during reconciliation to mark orphaned tool calls as cancelled.
func (r *Repository) UpdateToolCallStatus(ctx context.Context, id string, status domain.ToolCallStatus, errorMessage string) error {
	if !status.Valid() {
		return fmt.Errorf("cannot persist invalid tool call status %q", status)
	}
	allowed := make([]domain.ToolCallStatus, 0, len(allToolCallStatuses))
	for _, prior := range allToolCallStatuses {
		if prior.CanTransitionTo(status) {
			allowed = append(allowed, prior)
		}
	}
	placeholders := make([]string, len(allowed))
	args := []any{status, errorMessage, id}
	for i, prior := range allowed {
		placeholders[i] = fmt.Sprintf("$%d", 4+i)
		args = append(args, prior)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE tool_calls
		SET status = $1, error_message = $2,
			completed_at = CASE WHEN $1 IN ('rejected', 'completed', 'failed', 'cancelled') THEN datetime('now') ELSE NULL END
		WHERE id = $3 AND status IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		var current domain.ToolCallStatus
		if err := r.db.QueryRowContext(ctx, `SELECT status FROM tool_calls WHERE id = $1`, id).Scan(&current); err != nil {
			return fmt.Errorf("tool call %s not found or unreadable: %w", id, err)
		}
		return domain.ErrInvalidToolCallTransition{From: current, To: status}
	}
	return nil
}
