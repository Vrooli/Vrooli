package autosteer

import (
	"fmt"
)

// IterationEvaluator evaluates a single iteration and determines the next action.
// It coordinates metrics collection and phase advancement decisions.
// Implements IterationEvaluatorAPI (assertion in interfaces.go).
type IterationEvaluator struct {
	stateManager     ExecutionStateRepository // Interface for testability
	phaseCoordinator PhaseCoordinatorAPI
	metricsCollector MetricsProvider
	profileService   ProfileRepository
}

// NewIterationEvaluator creates a new IterationEvaluator.
// All dependencies are interfaces, enabling easy mocking for unit tests.
func NewIterationEvaluator(
	stateManager ExecutionStateRepository,
	phaseCoordinator PhaseCoordinatorAPI,
	metricsCollector MetricsProvider,
	profileService ProfileRepository,
) *IterationEvaluator {
	return &IterationEvaluator{
		stateManager:     stateManager,
		phaseCoordinator: phaseCoordinator,
		metricsCollector: metricsCollector,
		profileService:   profileService,
	}
}

// Evaluate evaluates the current iteration for a task.
// It collects metrics, checks stop conditions, and returns the evaluation result.
func (e *IterationEvaluator) Evaluate(taskID string, scenarioName string) (*IterationEvaluation, error) {
	// Get execution state
	state, err := e.stateManager.Get(taskID)
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

	// Increment iteration counters
	newIteration := state.CurrentPhaseIteration + 1
	newAutoSteerIteration := state.AutoSteerIteration + 1

	// Collect current metrics
	metrics, err := e.metricsCollector.CollectMetrics(scenarioName, newIteration, newAutoSteerIteration)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Update state with new metrics and counters
	e.stateManager.IncrementIteration(state, *metrics)

	// Evaluate phase advancement using PhaseCoordinator
	decision := e.phaseCoordinator.ShouldAdvancePhase(currentPhase, *metrics, state.CurrentPhaseIteration)

	// Save updated state
	if err := e.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	return &IterationEvaluation{
		ShouldStop: decision.ShouldStop,
		Reason:     decision.Reason,
	}, nil
}

// EvaluateWithoutMetricsCollection evaluates using existing metrics in the state.
// Useful for re-evaluation without collecting new metrics.
func (e *IterationEvaluator) EvaluateWithoutMetricsCollection(taskID string) (*IterationEvaluation, error) {
	// Get execution state
	state, err := e.stateManager.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
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
		return &IterationEvaluation{
			ShouldStop: true,
			Reason:     "all_phases_completed",
		}, nil
	}

	currentPhase := profile.Phases[state.CurrentPhaseIndex]

	// Evaluate using existing metrics
	decision := e.phaseCoordinator.ShouldAdvancePhase(currentPhase, state.Metrics, state.CurrentPhaseIteration)

	return &IterationEvaluation{
		ShouldStop: decision.ShouldStop,
		Reason:     decision.Reason,
	}, nil
}
