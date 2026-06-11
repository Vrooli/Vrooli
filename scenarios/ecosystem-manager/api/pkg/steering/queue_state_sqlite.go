package steering

import (
	"database/sql"
	_ "embed"
	"fmt"
	"time"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the steering queue-state SQLite DDL for registration with
// database.EnsureSchemas (see pkg/dbschema). It owns steering_queue_state.
func Schema() string { return schemaSQL }

// SQLiteQueueStateRepository implements QueueStateRepository using SQLite.
type SQLiteQueueStateRepository struct {
	db *sql.DB
}

// Compile-time interface assertion
var _ QueueStateRepository = (*SQLiteQueueStateRepository)(nil)

// NewSQLiteQueueStateRepository creates a new SQLite-backed queue state repository.
func NewSQLiteQueueStateRepository(db *sql.DB) *SQLiteQueueStateRepository {
	return &SQLiteQueueStateRepository{db: db}
}

// nowRFC3339 is the canonical timestamp the repository writes for queue-state
// mutations, matching the string CreatedAt/UpdatedAt fields on QueueState.
func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

// Get retrieves the queue state for a task from the database.
// Returns nil, nil if no state exists.
func (r *SQLiteQueueStateRepository) Get(taskID string) (*QueueState, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT task_id, current_index, created_at, updated_at
		FROM steering_queue_state
		WHERE task_id = ?
	`

	var state QueueState

	err := r.db.QueryRow(query, taskID).Scan(
		&state.TaskID,
		&state.CurrentIndex,
		&state.CreatedAt,
		&state.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query queue state: %w", err)
	}

	return &state, nil
}

// Save persists the queue state to the database (upsert).
func (r *SQLiteQueueStateRepository) Save(state *QueueState) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if state == nil {
		return fmt.Errorf("state is nil")
	}

	query := `
		INSERT INTO steering_queue_state (task_id, current_index, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (task_id) DO UPDATE SET
			current_index = excluded.current_index,
			updated_at = excluded.updated_at
	`

	_, err := r.db.Exec(query,
		state.TaskID,
		state.CurrentIndex,
		state.CreatedAt,
		state.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save queue state: %w", err)
	}

	return nil
}

// Delete removes the queue state for a task from the database.
func (r *SQLiteQueueStateRepository) Delete(taskID string) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `DELETE FROM steering_queue_state WHERE task_id = ?`

	_, err := r.db.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete queue state: %w", err)
	}

	return nil
}

// ResetPosition resets the queue position to 0 without deleting the state.
func (r *SQLiteQueueStateRepository) ResetPosition(taskID string) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `UPDATE steering_queue_state SET current_index = 0, updated_at = ? WHERE task_id = ?`

	_, err := r.db.Exec(query, nowRFC3339(), taskID)
	if err != nil {
		return fmt.Errorf("failed to reset queue position: %w", err)
	}

	return nil
}

// SetPosition sets the queue position to a specific index.
func (r *SQLiteQueueStateRepository) SetPosition(taskID string, position int) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if position < 0 {
		return fmt.Errorf("position must be non-negative, got %d", position)
	}

	query := `UPDATE steering_queue_state SET current_index = ?, updated_at = ? WHERE task_id = ?`

	result, err := r.db.Exec(query, position, nowRFC3339(), taskID)
	if err != nil {
		return fmt.Errorf("failed to set queue position: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no queue state found for task %s", taskID)
	}

	return nil
}
