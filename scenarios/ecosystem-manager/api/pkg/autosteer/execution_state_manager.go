package autosteer

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// ExecutionStateManager handles persistence of the controller's execution
// state. It owns all SQL for profile_execution_state and profile_executions.
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
		SELECT task_id, profile_id, iteration, current_skill, current_rationale,
		       findings, score_history, trace, metrics, started_at, last_updated
		FROM profile_execution_state
		WHERE task_id = $1
	`

	var state ProfileExecutionState
	var findingsJSON, scoreHistoryJSON, traceJSON, metricsJSON []byte

	err := m.db.QueryRow(query, taskID).Scan(
		&state.TaskID,
		&state.ProfileID,
		&state.Iteration,
		&state.CurrentSkill,
		&state.CurrentRationale,
		&findingsJSON,
		&scoreHistoryJSON,
		&traceJSON,
		&metricsJSON,
		&state.StartedAt,
		&state.LastUpdated,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query execution state: %w", err)
	}

	if !isNullJSON(findingsJSON) {
		if err := json.Unmarshal(findingsJSON, &state.Findings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal findings: %w", err)
		}
	}
	if !isNullJSON(scoreHistoryJSON) {
		if err := json.Unmarshal(scoreHistoryJSON, &state.ScoreHistory); err != nil {
			return nil, fmt.Errorf("failed to unmarshal score history: %w", err)
		}
	}
	if !isNullJSON(traceJSON) {
		if err := json.Unmarshal(traceJSON, &state.Trace); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trace: %w", err)
		}
	}
	if !isNullJSON(metricsJSON) {
		if err := json.Unmarshal(metricsJSON, &state.Metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
		}
	}

	return &state, nil
}

// Save persists the execution state to the database (upsert).
func (m *ExecutionStateManager) Save(state *ProfileExecutionState) error {
	findingsJSON, err := json.Marshal(state.Findings)
	if err != nil {
		return fmt.Errorf("failed to marshal findings: %w", err)
	}
	scoreHistoryJSON, err := json.Marshal(state.ScoreHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal score history: %w", err)
	}
	traceJSON, err := json.Marshal(state.Trace)
	if err != nil {
		return fmt.Errorf("failed to marshal trace: %w", err)
	}
	metricsJSON, err := json.Marshal(state.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	state.LastUpdated = time.Now()

	query := `
		INSERT INTO profile_execution_state (
			task_id, profile_id, iteration, current_skill, current_rationale,
			findings, score_history, trace, metrics, started_at, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (task_id) DO UPDATE SET
			profile_id = EXCLUDED.profile_id,
			iteration = EXCLUDED.iteration,
			current_skill = EXCLUDED.current_skill,
			current_rationale = EXCLUDED.current_rationale,
			findings = EXCLUDED.findings,
			score_history = EXCLUDED.score_history,
			trace = EXCLUDED.trace,
			metrics = EXCLUDED.metrics,
			last_updated = EXCLUDED.last_updated
	`

	_, err = m.db.Exec(query,
		state.TaskID,
		state.ProfileID,
		state.Iteration,
		state.CurrentSkill,
		state.CurrentRationale,
		findingsJSON,
		scoreHistoryJSON,
		traceJSON,
		metricsJSON,
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

// InitializeState creates a new controller state for a task.
func (m *ExecutionStateManager) InitializeState(taskID, profileID string) *ProfileExecutionState {
	now := time.Now()
	return &ProfileExecutionState{
		TaskID:       taskID,
		ProfileID:    profileID,
		Iteration:    0,
		ScoreHistory: []float64{},
		Trace:        []DecisionTraceEntry{},
		StartedAt:    now,
		LastUpdated:  now,
	}
}

// FinalizeExecution archives the completed execution to history and removes
// active state. The skill-performance breakdown is derived from the decision
// trace (realized weighted-score reduction attributed to each skill).
func (m *ExecutionStateManager) FinalizeExecution(state *ProfileExecutionState, scenarioName string) error {
	breakdown := skillBreakdownFromTrace(state.Trace)

	metricsJSON, err := json.Marshal(state.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}
	breakdownJSON, err := json.Marshal(breakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal skill breakdown: %w", err)
	}

	totalDuration := time.Since(state.StartedAt).Milliseconds()

	query := `
		INSERT INTO profile_executions (
			profile_id, task_id, scenario_name, start_metrics, end_metrics,
			phase_breakdown, total_iterations, total_duration_ms, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	// Gap metrics are not the primary state; start and end carry the latest
	// snapshot for analytics continuity.
	if _, err := m.db.Exec(query,
		state.ProfileID,
		state.TaskID,
		scenarioName,
		metricsJSON,
		metricsJSON,
		breakdownJSON,
		state.Iteration,
		totalDuration,
		time.Now(),
	); err != nil {
		return fmt.Errorf("failed to insert profile execution: %w", err)
	}

	return m.Delete(state.TaskID)
}

// skillBreakdownFromTrace aggregates the decision trace into per-skill
// performance records (iterations run and total realized weighted-score
// reduction).
func skillBreakdownFromTrace(trace []DecisionTraceEntry) []SkillPerformance {
	bySkill := map[string]*SkillPerformance{}
	order := make([]string, 0)
	for _, e := range trace {
		if e.ChosenSkill == "" {
			continue
		}
		sp, ok := bySkill[e.ChosenSkill]
		if !ok {
			sp = &SkillPerformance{SkillName: e.ChosenSkill}
			bySkill[e.ChosenSkill] = sp
			order = append(order, e.ChosenSkill)
		}
		sp.Iterations++
		sp.WeightedDelta += e.RealizedDelta
	}
	sort.Strings(order)
	out := make([]SkillPerformance, 0, len(order))
	for _, name := range order {
		out = append(out, *bySkill[name])
	}
	return out
}

func isNullJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
