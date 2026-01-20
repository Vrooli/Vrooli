// Package persistence provides database operations for the Agent Inbox scenario.
// This file contains async operation persistence for crash recovery and multi-consumer callbacks.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// AsyncOperationRecord represents a persisted async operation.
// This matches the services.AsyncOperation structure but is decoupled to avoid
// circular imports between persistence and services packages.
type AsyncOperationRecord struct {
	ToolCallID    string
	ChatID        string
	ToolName      string
	ScenarioName  string
	OperationID   string
	Status        string
	Progress      *int
	Message       string
	Phase         string
	Result        json.RawMessage
	Error         string
	AsyncBehavior json.RawMessage // Serialized proto as JSON
	StartedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   *time.Time
}

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

// Async Operation Persistence
// These methods enable crash recovery by persisting async operation state to the database.

// CreateAsyncOperation inserts a new async operation record.
func (r *Repository) CreateAsyncOperation(ctx context.Context, op *AsyncOperationRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO async_operations (
			tool_call_id, chat_id, tool_name, scenario_name, operation_id,
			status, progress, message, phase, result, error, async_behavior,
			started_at, updated_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, op.ToolCallID, op.ChatID, op.ToolName, op.ScenarioName, op.OperationID,
		op.Status, op.Progress, op.Message, op.Phase, op.Result, op.Error, op.AsyncBehavior,
		op.StartedAt, op.UpdatedAt, op.CompletedAt)
	if err != nil {
		return fmt.Errorf("failed to create async operation: %w", err)
	}
	return nil
}

// UpdateAsyncOperation updates an existing async operation record.
func (r *Repository) UpdateAsyncOperation(ctx context.Context, op *AsyncOperationRecord) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE async_operations SET
			status = $1,
			progress = $2,
			message = $3,
			phase = $4,
			result = $5,
			error = $6,
			updated_at = $7,
			completed_at = $8
		WHERE tool_call_id = $9
	`, op.Status, op.Progress, op.Message, op.Phase, op.Result, op.Error,
		op.UpdatedAt, op.CompletedAt, op.ToolCallID)
	if err != nil {
		return fmt.Errorf("failed to update async operation: %w", err)
	}
	return nil
}

// GetAsyncOperationByToolCallID retrieves an async operation by its tool call ID.
func (r *Repository) GetAsyncOperationByToolCallID(ctx context.Context, toolCallID string) (*AsyncOperationRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tool_call_id, chat_id, tool_name, scenario_name, operation_id,
			   status, progress, message, phase, result, error, async_behavior,
			   started_at, updated_at, completed_at
		FROM async_operations
		WHERE tool_call_id = $1
	`, toolCallID)

	var op AsyncOperationRecord
	var progress sql.NullInt32
	var message, phase, errorMsg sql.NullString
	var result, asyncBehavior sql.NullString
	var completedAt sql.NullTime

	err := row.Scan(
		&op.ToolCallID, &op.ChatID, &op.ToolName, &op.ScenarioName, &op.OperationID,
		&op.Status, &progress, &message, &phase, &result, &errorMsg, &asyncBehavior,
		&op.StartedAt, &op.UpdatedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get async operation: %w", err)
	}

	if progress.Valid {
		p := int(progress.Int32)
		op.Progress = &p
	}
	if message.Valid {
		op.Message = message.String
	}
	if phase.Valid {
		op.Phase = phase.String
	}
	if result.Valid {
		op.Result = json.RawMessage(result.String)
	}
	if errorMsg.Valid {
		op.Error = errorMsg.String
	}
	if asyncBehavior.Valid {
		op.AsyncBehavior = json.RawMessage(asyncBehavior.String)
	}
	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}

	return &op, nil
}

// GetActiveAsyncOperationsByChatID retrieves all non-completed async operations for a chat.
func (r *Repository) GetActiveAsyncOperationsByChatID(ctx context.Context, chatID string) ([]*AsyncOperationRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tool_call_id, chat_id, tool_name, scenario_name, operation_id,
			   status, progress, message, phase, result, error, async_behavior,
			   started_at, updated_at, completed_at
		FROM async_operations
		WHERE chat_id = $1 AND status NOT IN ('completed', 'failed', 'cancelled', 'timeout')
		ORDER BY started_at ASC
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active async operations: %w", err)
	}
	defer rows.Close()

	return scanAsyncOperations(rows)
}

// GetAllActiveAsyncOperations retrieves all non-completed async operations.
// Used during service initialization for crash recovery.
func (r *Repository) GetAllActiveAsyncOperations(ctx context.Context) ([]*AsyncOperationRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tool_call_id, chat_id, tool_name, scenario_name, operation_id,
			   status, progress, message, phase, result, error, async_behavior,
			   started_at, updated_at, completed_at
		FROM async_operations
		WHERE status NOT IN ('completed', 'failed', 'cancelled', 'timeout')
		ORDER BY started_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active async operations: %w", err)
	}
	defer rows.Close()

	return scanAsyncOperations(rows)
}

// DeleteAsyncOperation removes an async operation record.
func (r *Repository) DeleteAsyncOperation(ctx context.Context, toolCallID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM async_operations WHERE tool_call_id = $1`, toolCallID)
	if err != nil {
		return fmt.Errorf("failed to delete async operation: %w", err)
	}
	return nil
}

// CleanupCompletedAsyncOperations removes completed operations older than the specified duration.
// Returns the number of operations removed.
func (r *Repository) CleanupCompletedAsyncOperations(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM async_operations
		WHERE status IN ('completed', 'failed', 'cancelled', 'timeout')
		  AND completed_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup completed async operations: %w", err)
	}
	return result.RowsAffected()
}

// UpdateAsyncOperationStatus updates just the status of an async operation.
// Used during graceful shutdown to mark operations as interrupted.
func (r *Repository) UpdateAsyncOperationStatus(ctx context.Context, toolCallID, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE async_operations SET status = $1, updated_at = NOW() WHERE tool_call_id = $2
	`, status, toolCallID)
	if err != nil {
		return fmt.Errorf("failed to update async operation status: %w", err)
	}
	return nil
}

// Async Completion Events Persistence
// These methods enable multi-consumer callbacks by persisting completion events.

// CreateCompletionEvent inserts a new completion event record.
func (r *Repository) CreateCompletionEvent(ctx context.Context, event *AsyncCompletionEventRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO async_completion_events (chat_id, tool_call_id, tool_name, status, result, error)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, event.ChatID, event.ToolCallID, event.ToolName, event.Status, event.Result, event.Error)
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
			&event.Status, &result, &errorMsg, &event.CreatedAt); err != nil {
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

// =============================================================================
// Helper Functions
// =============================================================================

// Helper function to scan async operation rows
func scanAsyncOperations(rows *sql.Rows) ([]*AsyncOperationRecord, error) {
	var ops []*AsyncOperationRecord
	for rows.Next() {
		var op AsyncOperationRecord
		var progress sql.NullInt32
		var message, phase, errorMsg sql.NullString
		var result, asyncBehavior sql.NullString
		var completedAt sql.NullTime

		if err := rows.Scan(
			&op.ToolCallID, &op.ChatID, &op.ToolName, &op.ScenarioName, &op.OperationID,
			&op.Status, &progress, &message, &phase, &result, &errorMsg, &asyncBehavior,
			&op.StartedAt, &op.UpdatedAt, &completedAt,
		); err != nil {
			continue
		}

		if progress.Valid {
			p := int(progress.Int32)
			op.Progress = &p
		}
		if message.Valid {
			op.Message = message.String
		}
		if phase.Valid {
			op.Phase = phase.String
		}
		if result.Valid {
			op.Result = json.RawMessage(result.String)
		}
		if errorMsg.Valid {
			op.Error = errorMsg.String
		}
		if asyncBehavior.Valid {
			op.AsyncBehavior = json.RawMessage(asyncBehavior.String)
		}
		if completedAt.Valid {
			op.CompletedAt = &completedAt.Time
		}

		ops = append(ops, &op)
	}

	return ops, nil
}
