package steering

import (
	"database/sql"
	"fmt"
	"time"
)

// PostgresQueueStateRepository implements QueueStateRepository using PostgreSQL.
type PostgresQueueStateRepository struct {
	db *sql.DB
}

// Compile-time interface assertion
var _ QueueStateRepository = (*PostgresQueueStateRepository)(nil)

// NewPostgresQueueStateRepository creates a new PostgreSQL-backed queue state repository.
func NewPostgresQueueStateRepository(db *sql.DB) *PostgresQueueStateRepository {
	return &PostgresQueueStateRepository{db: db}
}

// Get retrieves the queue state for a task from the database.
// Returns nil, nil if no state exists.
func (r *PostgresQueueStateRepository) Get(taskID string) (*QueueState, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT task_id, current_index, created_at, updated_at
		FROM steering_queue_state
		WHERE task_id = $1
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
func (r *PostgresQueueStateRepository) Save(state *QueueState) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if state == nil {
		return fmt.Errorf("state is nil")
	}

	query := `
		INSERT INTO steering_queue_state (task_id, current_index, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id) DO UPDATE SET
			current_index = EXCLUDED.current_index,
			updated_at = EXCLUDED.updated_at
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
func (r *PostgresQueueStateRepository) Delete(taskID string) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `DELETE FROM steering_queue_state WHERE task_id = $1`

	_, err := r.db.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete queue state: %w", err)
	}

	return nil
}

// ResetPosition resets the queue position to 0 without deleting the state.
func (r *PostgresQueueStateRepository) ResetPosition(taskID string) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `UPDATE steering_queue_state SET current_index = 0, updated_at = NOW() WHERE task_id = $1`

	_, err := r.db.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("failed to reset queue position: %w", err)
	}

	return nil
}

// SetPosition sets the queue position to a specific index.
func (r *PostgresQueueStateRepository) SetPosition(taskID string, position int) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if position < 0 {
		return fmt.Errorf("position must be non-negative, got %d", position)
	}

	query := `UPDATE steering_queue_state SET current_index = $2, updated_at = NOW() WHERE task_id = $1`

	result, err := r.db.Exec(query, taskID, position)
	if err != nil {
		return fmt.Errorf("failed to set queue position: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no queue state found for task %s", taskID)
	}

	return nil
}

// InMemoryQueueStateRepository implements QueueStateRepository using an in-memory map.
// Useful for testing and development.
type InMemoryQueueStateRepository struct {
	states map[string]*QueueState
}

// Compile-time interface assertion
var _ QueueStateRepository = (*InMemoryQueueStateRepository)(nil)

// NewInMemoryQueueStateRepository creates a new in-memory queue state repository.
func NewInMemoryQueueStateRepository() *InMemoryQueueStateRepository {
	return &InMemoryQueueStateRepository{
		states: make(map[string]*QueueState),
	}
}

// Get retrieves the queue state for a task from memory.
func (r *InMemoryQueueStateRepository) Get(taskID string) (*QueueState, error) {
	state, ok := r.states[taskID]
	if !ok {
		return nil, nil
	}
	// Return a copy to prevent mutation
	cp := *state
	return &cp, nil
}

// Save persists the queue state to memory.
func (r *InMemoryQueueStateRepository) Save(state *QueueState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}
	// Store a copy to prevent mutation
	cp := *state
	r.states[state.TaskID] = &cp
	return nil
}

// Delete removes the queue state for a task from memory.
func (r *InMemoryQueueStateRepository) Delete(taskID string) error {
	delete(r.states, taskID)
	return nil
}

// ResetPosition resets the queue position to 0 without deleting the state.
func (r *InMemoryQueueStateRepository) ResetPosition(taskID string) error {
	if state, ok := r.states[taskID]; ok {
		state.Reset() // Uses existing QueueState.Reset() method
	}
	return nil
}

// SetPosition sets the queue position to a specific index.
func (r *InMemoryQueueStateRepository) SetPosition(taskID string, position int) error {
	if position < 0 {
		return fmt.Errorf("position must be non-negative, got %d", position)
	}

	state, ok := r.states[taskID]
	if !ok {
		return fmt.Errorf("no queue state found for task %s", taskID)
	}

	state.CurrentIndex = position
	state.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}
