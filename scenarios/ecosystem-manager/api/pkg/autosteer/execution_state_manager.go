package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ExecutionStateManager handles persistence of profile execution state.
// It owns all SQL operations for the profile_execution_state and profile_executions tables.
// Implements ExecutionStateRepository (assertion in repositories.go).
type ExecutionStateManager struct {
	db *sql.DB
}

// NewExecutionStateManager creates a new ExecutionStateManager
func NewExecutionStateManager(db *sql.DB) *ExecutionStateManager {
	return &ExecutionStateManager{
		db: db,
	}
}

// Get retrieves the current execution state for a task.
// Returns nil, nil if no execution state exists for the task.
func (m *ExecutionStateManager) Get(taskID string) (*ProfileExecutionState, error) {
	query := `
		SELECT task_id, profile_id, current_phase_index, current_phase_iteration,
		       auto_steer_iteration, phase_started_at, phase_history, metrics, phase_start_metrics, started_at, last_updated
		FROM profile_execution_state
		WHERE task_id = $1
	`

	var state ProfileExecutionState
	var phaseHistoryJSON, metricsJSON, phaseStartMetricsJSON []byte
	var phaseStartedAt sql.NullTime

	err := m.db.QueryRow(query, taskID).Scan(
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
		return nil, nil // No execution state for this task
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query execution state: %w", err)
	}

	// Unmarshal JSON fields
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

// Save persists the execution state to the database (upsert).
func (m *ExecutionStateManager) Save(state *ProfileExecutionState) error {
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

	_, err = m.db.Exec(query,
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
func (m *ExecutionStateManager) Delete(taskID string) error {
	query := `DELETE FROM profile_execution_state WHERE task_id = $1`
	if _, err := m.db.Exec(query, taskID); err != nil {
		return fmt.Errorf("failed to delete execution state for task %s: %w", taskID, err)
	}
	return nil
}

// RecordPhaseCompletion appends a completed phase to the execution history.
func (m *ExecutionStateManager) RecordPhaseCompletion(state *ProfileExecutionState, phase SteerPhase, stopReason string) error {
	now := time.Now()

	phaseExecution := PhaseExecution{
		PhaseID:      phase.ID,
		Mode:         phase.Mode,
		Iterations:   state.CurrentPhaseIteration,
		StartMetrics: state.PhaseStartMetrics,
		EndMetrics:   state.Metrics,
		Commits:      []string{}, // Git commits collection deferred
		StartedAt:    state.PhaseStartedAt,
		CompletedAt:  &now,
		StopReason:   stopReason,
	}

	state.PhaseHistory = append(state.PhaseHistory, phaseExecution)

	return m.Save(state)
}

// FinalizeExecution archives the completed execution to history and removes active state.
func (m *ExecutionStateManager) FinalizeExecution(state *ProfileExecutionState, scenarioName string) error {
	// Get start metrics (from beginning of first phase)
	var startMetrics MetricsSnapshot
	if len(state.PhaseHistory) > 0 {
		startMetrics = state.PhaseHistory[0].StartMetrics
	} else {
		startMetrics = state.Metrics
	}

	// Calculate total duration and iterations
	totalDuration := time.Since(state.StartedAt).Milliseconds()
	totalIterations := state.AutoSteerIteration
	if totalIterations == 0 {
		for _, phase := range state.PhaseHistory {
			totalIterations += phase.Iterations
		}
	}

	// Create phase breakdown
	phaseBreakdown := make([]PhasePerformance, len(state.PhaseHistory))
	for i, phase := range state.PhaseHistory {
		var duration int64
		if phase.CompletedAt != nil {
			duration = phase.CompletedAt.Sub(phase.StartedAt).Milliseconds()
		}

		phaseBreakdown[i] = PhasePerformance{
			Mode:          phase.Mode,
			Iterations:    phase.Iterations,
			MetricDeltas:  calculateMetricDeltas(phase.StartMetrics, phase.EndMetrics),
			Duration:      duration,
			Effectiveness: calculateEffectiveness(phase),
		}
	}

	// Marshal JSON fields
	startMetricsJSON, err := json.Marshal(startMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal start metrics: %w", err)
	}

	endMetricsJSON, err := json.Marshal(state.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal end metrics: %w", err)
	}

	phaseBreakdownJSON, err := json.Marshal(phaseBreakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal phase breakdown: %w", err)
	}

	// Insert into profile_executions table
	query := `
		INSERT INTO profile_executions (
			profile_id, task_id, scenario_name, start_metrics, end_metrics,
			phase_breakdown, total_iterations, total_duration_ms, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = m.db.Exec(query,
		state.ProfileID,
		state.TaskID,
		scenarioName,
		startMetricsJSON,
		endMetricsJSON,
		phaseBreakdownJSON,
		totalIterations,
		totalDuration,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to insert profile execution: %w", err)
	}

	// Delete execution state (it's now in history)
	return m.Delete(state.TaskID)
}

// InitializeState creates a new execution state for a task.
func (m *ExecutionStateManager) InitializeState(taskID, profileID string, initialMetrics MetricsSnapshot) *ProfileExecutionState {
	now := time.Now()
	return &ProfileExecutionState{
		TaskID:                taskID,
		ProfileID:             profileID,
		CurrentPhaseIndex:     0,
		CurrentPhaseIteration: 0,
		AutoSteerIteration:    0,
		PhaseStartedAt:        now,
		PhaseHistory:          []PhaseExecution{},
		Metrics:               initialMetrics,
		PhaseStartMetrics:     initialMetrics,
		StartedAt:             now,
		LastUpdated:           now,
	}
}

// AdvanceToNextPhase updates the state to move to the next phase.
func (m *ExecutionStateManager) AdvanceToNextPhase(state *ProfileExecutionState) {
	state.CurrentPhaseIndex++
	state.CurrentPhaseIteration = 0
	state.PhaseStartMetrics = state.Metrics
	state.PhaseStartedAt = time.Now()
	state.LastUpdated = time.Now()
}

// IncrementIteration increments the iteration counters and updates metrics.
func (m *ExecutionStateManager) IncrementIteration(state *ProfileExecutionState, newMetrics MetricsSnapshot) {
	state.AutoSteerIteration++
	state.CurrentPhaseIteration++
	state.Metrics = newMetrics
	state.LastUpdated = time.Now()
}
