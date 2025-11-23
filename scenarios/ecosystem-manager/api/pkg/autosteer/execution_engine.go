package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ExecutionEngine manages profile execution state and phase transitions
type ExecutionEngine struct {
	db                *sql.DB
	profileService    *ProfileService
	conditionEvaluator *ConditionEvaluator
	metricsCollector  *MetricsCollector
	promptEnhancer    *PromptEnhancer
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(db *sql.DB, profileService *ProfileService, metricsCollector *MetricsCollector) *ExecutionEngine {
	return &ExecutionEngine{
		db:                db,
		profileService:    profileService,
		conditionEvaluator: NewConditionEvaluator(),
		metricsCollector:  metricsCollector,
		promptEnhancer:    NewPromptEnhancer(),
	}
}

// StartExecution initializes a new profile execution for a task
func (e *ExecutionEngine) StartExecution(taskID string, profileID string, scenarioName string) (*ProfileExecutionState, error) {
	// Get profile
	profile, err := e.profileService.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Collect initial metrics
	metrics, err := e.metricsCollector.CollectMetrics(scenarioName, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to collect initial metrics: %w", err)
	}

	// Create execution state
	state := &ProfileExecutionState{
		TaskID:               taskID,
		ProfileID:            profileID,
		CurrentPhaseIndex:    0,
		CurrentPhaseIteration: 0,
		PhaseHistory:         []PhaseExecution{},
		Metrics:              *metrics,
		PhaseStartMetrics:    *metrics, // First phase starts with initial metrics
		StartedAt:            time.Now(),
		LastUpdated:          time.Now(),
	}

	// Save to database
	if err := e.saveExecutionState(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	// Start first phase
	if err := e.startPhase(state, profile); err != nil {
		return nil, fmt.Errorf("failed to start first phase: %w", err)
	}

	return state, nil
}

// GetExecutionState retrieves the current execution state for a task
func (e *ExecutionEngine) GetExecutionState(taskID string) (*ProfileExecutionState, error) {
	query := `
		SELECT task_id, profile_id, current_phase_index, current_phase_iteration,
		       phase_history, metrics, phase_start_metrics, started_at, last_updated
		FROM profile_execution_state
		WHERE task_id = $1
	`

	var state ProfileExecutionState
	var phaseHistoryJSON, metricsJSON, phaseStartMetricsJSON []byte

	err := e.db.QueryRow(query, taskID).Scan(
		&state.TaskID,
		&state.ProfileID,
		&state.CurrentPhaseIndex,
		&state.CurrentPhaseIteration,
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

	return &state, nil
}

// EvaluateIteration evaluates the current iteration and determines if the phase should stop
func (e *ExecutionEngine) EvaluateIteration(taskID string, scenarioName string, loops int) (*IterationEvaluation, error) {
	// Get execution state
	state, err := e.GetExecutionState(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
		// No Auto Steer profile for this task
		return &IterationEvaluation{
			ShouldStop: false,
		}, nil
	}

	// Get profile
	profile, err := e.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if state.CurrentPhaseIndex >= len(profile.Phases) {
		// All phases completed
		return &IterationEvaluation{
			ShouldStop: true,
			Reason:     "all_phases_completed",
		}, nil
	}

	currentPhase := profile.Phases[state.CurrentPhaseIndex]

	// Collect current metrics
	metrics, err := e.metricsCollector.CollectMetrics(scenarioName, loops)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Update state with new metrics
	state.Metrics = *metrics
	state.CurrentPhaseIteration++
	state.LastUpdated = time.Now()

	// Check if max iterations reached
	if state.CurrentPhaseIteration >= currentPhase.MaxIterations {
		if err := e.saveExecutionState(state); err != nil {
			return nil, fmt.Errorf("failed to save execution state: %w", err)
		}

		return &IterationEvaluation{
			ShouldStop: true,
			Reason:     "max_iterations",
		}, nil
	}

	// Evaluate stop conditions
	for i, condition := range currentPhase.StopConditions {
		result, err := e.conditionEvaluator.Evaluate(condition, *metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate condition %d: %w", i, err)
		}

		if result {
			if err := e.saveExecutionState(state); err != nil {
				return nil, fmt.Errorf("failed to save execution state: %w", err)
			}

			return &IterationEvaluation{
				ShouldStop: true,
				Reason:     "condition_met",
			}, nil
		}
	}

	// Save updated state
	if err := e.saveExecutionState(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	// Continue in current phase
	return &IterationEvaluation{
		ShouldStop: false,
	}, nil
}

// AdvancePhase advances to the next phase or completes execution
func (e *ExecutionEngine) AdvancePhase(taskID string, scenarioName string) (*PhaseAdvanceResult, error) {
	// Get execution state
	state, err := e.GetExecutionState(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
		return nil, fmt.Errorf("no execution state found for task: %s", taskID)
	}

	// Get profile
	profile, err := e.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Complete current phase
	if err := e.completePhase(state, profile); err != nil {
		return nil, fmt.Errorf("failed to complete phase: %w", err)
	}

	// Check if there are more phases
	if state.CurrentPhaseIndex+1 >= len(profile.Phases) {
		// All phases completed - finalize execution
		if err := e.completeExecution(state, scenarioName); err != nil {
			return nil, fmt.Errorf("failed to complete execution: %w", err)
		}

		return &PhaseAdvanceResult{
			Success:   true,
			Completed: true,
			Message:   "All phases completed successfully",
		}, nil
	}

	// Evaluate quality gates before advancing
	gateResults, err := e.evaluateQualityGates(profile.QualityGates, state.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate quality gates: %w", err)
	}

	for _, result := range gateResults {
		if !result.Passed {
			if result.Action == ActionHalt {
				return &PhaseAdvanceResult{
					Success: false,
					Message: fmt.Sprintf("Quality gate '%s' failed: %s", result.GateName, result.Message),
				}, nil
			}
		}
	}

	// Advance to next phase
	state.CurrentPhaseIndex++
	state.CurrentPhaseIteration = 0
	state.PhaseStartMetrics = state.Metrics // Capture metrics at start of new phase
	state.LastUpdated = time.Now()

	// Start next phase
	if err := e.startPhase(state, profile); err != nil {
		return nil, fmt.Errorf("failed to start next phase: %w", err)
	}

	return &PhaseAdvanceResult{
		Success:        true,
		NextPhaseIndex: state.CurrentPhaseIndex,
		Completed:      false,
		Message:        fmt.Sprintf("Advanced to phase %d: %s", state.CurrentPhaseIndex+1, profile.Phases[state.CurrentPhaseIndex].Mode),
	}, nil
}

// EvaluateQualityGates evaluates all quality gates
func (e *ExecutionEngine) evaluateQualityGates(gates []QualityGate, metrics MetricsSnapshot) ([]QualityGateResult, error) {
	results := make([]QualityGateResult, len(gates))

	for i, gate := range gates {
		passed, err := e.conditionEvaluator.Evaluate(gate.Condition, metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate gate '%s': %w", gate.Name, err)
		}

		results[i] = QualityGateResult{
			GateName: gate.Name,
			Passed:   passed,
			Message:  gate.Message,
			Action:   gate.FailureAction,
		}
	}

	return results, nil
}

// startPhase records the start of a phase
func (e *ExecutionEngine) startPhase(state *ProfileExecutionState, profile *AutoSteerProfile) error {
	// The phase is already set in the state
	// Just save the updated state
	return e.saveExecutionState(state)
}

// completePhase records the completion of the current phase
func (e *ExecutionEngine) completePhase(state *ProfileExecutionState, profile *AutoSteerProfile) error {
	if state.CurrentPhaseIndex >= len(profile.Phases) {
		return fmt.Errorf("invalid phase index: %d", state.CurrentPhaseIndex)
	}

	currentPhase := profile.Phases[state.CurrentPhaseIndex]

	// Use the phase start metrics that were captured when the phase began
	startMetrics := state.PhaseStartMetrics

	now := time.Now()

	// Determine stop reason based on current state
	stopReason := "max_iterations"
	if state.CurrentPhaseIteration < currentPhase.MaxIterations {
		stopReason = "condition_met"
	}

	phaseExecution := PhaseExecution{
		PhaseID:      currentPhase.ID,
		Mode:         currentPhase.Mode,
		Iterations:   state.CurrentPhaseIteration,
		StartMetrics: startMetrics,
		EndMetrics:   state.Metrics,
		Commits:      []string{}, // Git commits collection deferred to Phase 2
		StartedAt:    state.StartedAt,
		CompletedAt:  &now,
		StopReason:   stopReason,
	}

	state.PhaseHistory = append(state.PhaseHistory, phaseExecution)

	return e.saveExecutionState(state)
}

// CompleteExecution finalizes the execution and archives it to history
func (e *ExecutionEngine) completeExecution(state *ProfileExecutionState, scenarioName string) error {
	// Get start metrics (from beginning of first phase)
	var startMetrics MetricsSnapshot
	if len(state.PhaseHistory) > 0 {
		startMetrics = state.PhaseHistory[0].StartMetrics
	} else {
		startMetrics = state.Metrics
	}

	// Calculate total duration and iterations
	totalDuration := time.Since(state.StartedAt).Milliseconds()
	totalIterations := 0
	for _, phase := range state.PhaseHistory {
		totalIterations += phase.Iterations
	}

	// Create phase breakdown
	phaseBreakdown := make([]PhasePerformance, len(state.PhaseHistory))
	for i, phase := range state.PhaseHistory {
		var duration int64
		if phase.CompletedAt != nil {
			duration = phase.CompletedAt.Sub(phase.StartedAt).Milliseconds()
		}

		phaseBreakdown[i] = PhasePerformance{
			Mode:         phase.Mode,
			Iterations:   phase.Iterations,
			MetricDeltas: calculateMetricDeltas(phase.StartMetrics, phase.EndMetrics),
			Duration:     duration,
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

	_, err = e.db.Exec(query,
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
	deleteQuery := `DELETE FROM profile_execution_state WHERE task_id = $1`
	_, err = e.db.Exec(deleteQuery, state.TaskID)
	if err != nil {
		return fmt.Errorf("failed to delete execution state: %w", err)
	}

	return nil
}

// saveExecutionState saves the execution state to the database
func (e *ExecutionEngine) saveExecutionState(state *ProfileExecutionState) error {
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
			task_id, profile_id, current_phase_index, current_phase_iteration,
			phase_history, metrics, phase_start_metrics, started_at, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (task_id) DO UPDATE SET
			profile_id = EXCLUDED.profile_id,
			current_phase_index = EXCLUDED.current_phase_index,
			current_phase_iteration = EXCLUDED.current_phase_iteration,
			phase_history = EXCLUDED.phase_history,
			metrics = EXCLUDED.metrics,
			phase_start_metrics = EXCLUDED.phase_start_metrics,
			last_updated = EXCLUDED.last_updated
	`

	_, err = e.db.Exec(query,
		state.TaskID,
		state.ProfileID,
		state.CurrentPhaseIndex,
		state.CurrentPhaseIteration,
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

// Helper functions

func calculateMetricDeltas(start, end MetricsSnapshot) map[string]float64 {
	deltas := make(map[string]float64)

	// Universal metrics
	deltas["operational_targets_percentage"] = end.OperationalTargetsPercentage - start.OperationalTargetsPercentage

	// UX metrics
	if start.UX != nil && end.UX != nil {
		deltas["accessibility_score"] = end.UX.AccessibilityScore - start.UX.AccessibilityScore
		deltas["ui_test_coverage"] = end.UX.UITestCoverage - start.UX.UITestCoverage
	}

	// Test metrics
	if start.Test != nil && end.Test != nil {
		deltas["unit_test_coverage"] = end.Test.UnitTestCoverage - start.Test.UnitTestCoverage
		deltas["flaky_tests"] = float64(end.Test.FlakyTests - start.Test.FlakyTests)
	}

	// Refactor metrics
	if start.Refactor != nil && end.Refactor != nil {
		deltas["tidiness_score"] = end.Refactor.TidinessScore - start.Refactor.TidinessScore
		deltas["cyclomatic_complexity_avg"] = start.Refactor.CyclomaticComplexityAvg - end.Refactor.CyclomaticComplexityAvg // Negative is good
	}

	return deltas
}

func calculateEffectiveness(phase PhaseExecution) float64 {
	// Simple effectiveness calculation: improvement per iteration
	// This can be made more sophisticated based on mode
	deltas := calculateMetricDeltas(phase.StartMetrics, phase.EndMetrics)

	totalImprovement := 0.0
	for _, delta := range deltas {
		if delta > 0 {
			totalImprovement += delta
		}
	}

	if phase.Iterations == 0 {
		return 0
	}

	return totalImprovement / float64(phase.Iterations)
}

// GetEnhancedPrompt generates the Auto Steer prompt section for a task
// Returns empty string if the task doesn't have an Auto Steer profile
func (e *ExecutionEngine) GetEnhancedPrompt(taskID string) (string, error) {
	// Get execution state
	state, err := e.GetExecutionState(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
		// No Auto Steer profile for this task
		return "", nil
	}

	// Get profile
	profile, err := e.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	// Generate prompt section
	promptSection := e.promptEnhancer.GenerateAutoSteerSection(state, profile, e.conditionEvaluator)

	return promptSection, nil
}

// GetCurrentMode returns the current steering mode for a task
// Returns empty string if the task doesn't have an Auto Steer profile
func (e *ExecutionEngine) GetCurrentMode(taskID string) (SteerMode, error) {
	state, err := e.GetExecutionState(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
		return "", nil
	}

	profile, err := e.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	if state.CurrentPhaseIndex >= len(profile.Phases) {
		return "", nil
	}

	return profile.Phases[state.CurrentPhaseIndex].Mode, nil
}

// IsAutoSteerActive checks if a task has an active Auto Steer profile
func (e *ExecutionEngine) IsAutoSteerActive(taskID string) (bool, error) {
	state, err := e.GetExecutionState(taskID)
	if err != nil {
		return false, fmt.Errorf("failed to get execution state: %w", err)
	}

	return state != nil, nil
}
