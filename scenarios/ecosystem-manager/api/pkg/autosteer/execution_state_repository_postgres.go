package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// PostgresExecutionStateRepository implements ExecutionStateRepository using PostgreSQL.
type PostgresExecutionStateRepository struct {
	db *sql.DB
}

// NewPostgresExecutionStateRepository creates a new PostgreSQL execution state repository.
func NewPostgresExecutionStateRepository(db *sql.DB) *PostgresExecutionStateRepository {
	return &PostgresExecutionStateRepository{db: db}
}

// Get retrieves the execution state for a task.
func (r *PostgresExecutionStateRepository) Get(taskID string) (*ProfileExecutionState, error) {
	query := `
		SELECT task_id, profile_id, current_phase_index, current_phase_iteration,
		       auto_steer_iteration, phase_started_at, phase_history, metrics, phase_start_metrics, started_at, last_updated
		FROM profile_execution_state
		WHERE task_id = $1
	`

	var state ProfileExecutionState
	var phaseHistoryJSON, metricsJSON, phaseStartMetricsJSON []byte
	var phaseStartedAt sql.NullTime

	err := r.db.QueryRow(query, taskID).Scan(
		&state.TaskID,
		&state.ProfileID,
		&state.CurrentPhaseIndex,
		&state.CurrentPhaseIteration,
		&state.AutoSteerIteration,
		&phaseStartedAt,
		&phaseHistoryJSON,
		&metricsJSON,
		&phaseStartMetricsJSON,
		&state.StartedAt,
		&state.LastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query execution state: %w", err)
	}

	if err := json.Unmarshal(phaseHistoryJSON, &state.PhaseHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phase history: %w", err)
	}

	if err := json.Unmarshal(metricsJSON, &state.Metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	if err := json.Unmarshal(phaseStartMetricsJSON, &state.PhaseStartMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phase start metrics: %w", err)
	}

	if phaseStartedAt.Valid {
		state.PhaseStartedAt = phaseStartedAt.Time
	} else {
		state.PhaseStartedAt = state.StartedAt
	}

	return &state, nil
}

// Save persists the execution state (upsert).
func (r *PostgresExecutionStateRepository) Save(state *ProfileExecutionState) error {
	phaseHistoryJSON, err := json.Marshal(state.PhaseHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal phase history: %w", err)
	}

	metricsJSON, err := json.Marshal(state.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	phaseStartMetricsJSON, err := json.Marshal(state.PhaseStartMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal phase start metrics: %w", err)
	}

	query := `
		INSERT INTO profile_execution_state (
			task_id, profile_id, current_phase_index, current_phase_iteration, auto_steer_iteration, phase_started_at,
			phase_history, metrics, phase_start_metrics, started_at, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (task_id) DO UPDATE SET
			profile_id = EXCLUDED.profile_id,
			current_phase_index = EXCLUDED.current_phase_index,
			current_phase_iteration = EXCLUDED.current_phase_iteration,
			auto_steer_iteration = EXCLUDED.auto_steer_iteration,
			phase_started_at = EXCLUDED.phase_started_at,
			phase_history = EXCLUDED.phase_history,
			metrics = EXCLUDED.metrics,
			phase_start_metrics = EXCLUDED.phase_start_metrics,
			last_updated = EXCLUDED.last_updated
	`

	_, err = r.db.Exec(query,
		state.TaskID,
		state.ProfileID,
		state.CurrentPhaseIndex,
		state.CurrentPhaseIteration,
		state.AutoSteerIteration,
		state.PhaseStartedAt,
		phaseHistoryJSON,
		metricsJSON,
		phaseStartMetricsJSON,
		state.StartedAt,
		state.LastUpdated,
	)
	if err != nil {
		return fmt.Errorf("failed to save execution state: %w", err)
	}

	return nil
}

// Delete removes the execution state for a task.
func (r *PostgresExecutionStateRepository) Delete(taskID string) error {
	query := `DELETE FROM profile_execution_state WHERE task_id = $1`
	if _, err := r.db.Exec(query, taskID); err != nil {
		return fmt.Errorf("failed to delete execution state for task %s: %w", taskID, err)
	}
	return nil
}
