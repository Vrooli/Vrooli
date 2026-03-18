// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains async completion event persistence for multi-consumer callbacks.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// AsyncCompletionEventRecord represents a persisted completion event.
type AsyncCompletionEventRecord struct {
	ID         string
	ChatID     string
	ToolCallID string
	ToolName   string
	Status     string
	Result     json.RawMessage
	Error      string
	CreatedAt  time.Time
}

// Async Completion Events Persistence
// These methods enable multi-consumer callbacks by persisting completion events.

// CreateCompletionEvent inserts a new completion event record.
func (r *Repository) CreateCompletionEvent(ctx context.Context, event *AsyncCompletionEventRecord) error {
	id := newID()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO async_completion_events (id, chat_id, tool_call_id, tool_name, status, result, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, event.ChatID, event.ToolCallID, event.ToolName, event.Status, event.Result, event.Error)
	if err != nil {
		return fmt.Errorf("failed to create completion event: %w", err)
	}
	return nil
}

// GetCompletionEventsSince retrieves completion events for a chat after the specified time.
// Used by AI handlers to poll for completion events they haven't seen yet.
func (r *Repository) GetCompletionEventsSince(ctx context.Context, chatID string, since time.Time) ([]*AsyncCompletionEventRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, chat_id, tool_call_id, tool_name, status, result, error, created_at
		FROM async_completion_events
		WHERE chat_id = $1 AND created_at > $2
		ORDER BY created_at ASC
	`, chatID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion events: %w", err)
	}
	defer rows.Close()

	var events []*AsyncCompletionEventRecord
	for rows.Next() {
		var event AsyncCompletionEventRecord
		var result, errorMsg sql.NullString

		if err := rows.Scan(&event.ID, &event.ChatID, &event.ToolCallID, &event.ToolName,
			&event.Status, &result, &errorMsg, scanTime(&event.CreatedAt)); err != nil {
			continue
		}

		if result.Valid {
			event.Result = json.RawMessage(result.String)
		}
		if errorMsg.Valid {
			event.Error = errorMsg.String
		}

		events = append(events, &event)
	}

	return events, nil
}

// CleanupOldCompletionEvents removes completion events older than the specified duration.
// Returns the number of events removed.
func (r *Repository) CleanupOldCompletionEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM async_completion_events WHERE created_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old completion events: %w", err)
	}
	return result.RowsAffected()
}
