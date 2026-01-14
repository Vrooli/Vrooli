package autosteer

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// NewExecutionOrchestratorFromDB creates a fully wired ExecutionOrchestrator from database connection.
// This is the primary factory function for production use.
func NewExecutionOrchestratorFromDB(db *sql.DB, projectRoot string, phasesDir string) *ExecutionOrchestrator {
	// Create sub-components
	profileService := NewProfileService(db)
	metricsCollector := NewMetricsCollector(projectRoot)
	conditionEvaluator := NewConditionEvaluator()
	promptEnhancer := NewPromptEnhancer(phasesDir)

	// Create SRP components
	stateManager := NewExecutionStateManager(db)
	phaseCoordinator := NewPhaseCoordinator(conditionEvaluator)
	iterationEval := NewIterationEvaluator(stateManager, phaseCoordinator, metricsCollector, profileService)

	return NewExecutionOrchestrator(
		stateManager,
		phaseCoordinator,
		iterationEval,
		profileService,
		metricsCollector,
		promptEnhancer,
	)
}

// ExecutionOrchestrator coordinates profile execution by delegating to specialized components.
// It implements ExecutionEngineAPI as a thin facade over the component collaborators.
type ExecutionOrchestrator struct {
	stateManager     ExecutionStateRepository // Interface for testability
	phaseCoordinator PhaseCoordinatorAPI
	iterationEval    IterationEvaluatorAPI // Interface for testability
	profileService   ProfileRepository
	metricsCollector MetricsProvider
	promptEnhancer   PromptEnhancerAPI // Interface for testability
}

// NewExecutionOrchestrator creates a new ExecutionOrchestrator with all its collaborators.
// All dependencies are interfaces, enabling easy mocking for unit tests.
func NewExecutionOrchestrator(
	stateManager ExecutionStateRepository,
	phaseCoordinator PhaseCoordinatorAPI,
	iterationEval IterationEvaluatorAPI,
	profileService ProfileRepository,
	metricsCollector MetricsProvider,
	promptEnhancer PromptEnhancerAPI,
) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		stateManager:     stateManager,
		phaseCoordinator: phaseCoordinator,
		iterationEval:    iterationEval,
		profileService:   profileService,
		metricsCollector: metricsCollector,
		promptEnhancer:   promptEnhancer,
	}
}

// StartExecution initializes a new profile execution for a task.
// Implements ExecutionEngineAPI.
func (o *ExecutionOrchestrator) StartExecution(taskID string, profileID string, scenarioName string) (*ProfileExecutionState, error) {
	// Get profile
	profile, err := o.profileService.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Collect initial metrics
	metrics, err := o.metricsCollector.CollectMetrics(scenarioName, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to collect initial metrics: %w", err)
	}

	// Initialize state
	state := o.stateManager.InitializeState(taskID, profileID, *metrics)

	// Save to database
	if err := o.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	// Log phase start
	if len(profile.Phases) > 0 {
		log.Printf("Auto Steer: Started execution for task %s with profile %s, beginning phase 0 (%s)",
			taskID, profileID, profile.Phases[0].Mode)
	}

	return state, nil
}

// GetExecutionState retrieves the current execution state for a task.
// Implements ExecutionEngineAPI.
func (o *ExecutionOrchestrator) GetExecutionState(taskID string) (*ProfileExecutionState, error) {
	return o.stateManager.Get(taskID)
}

// EvaluateIteration evaluates the current iteration and determines if the phase should stop.
// Implements ExecutionEngineAPI.
func (o *ExecutionOrchestrator) EvaluateIteration(taskID string, scenarioName string) (*IterationEvaluation, error) {
	return o.iterationEval.Evaluate(taskID, scenarioName)
}

// SeekExecution manually adjusts the execution cursor to a specific phase/iteration.
// Implements ExecutionEngineAPI.
func (o *ExecutionOrchestrator) SeekExecution(taskID, profileID, scenarioName string, phaseIndex int, phaseIteration int) (*ProfileExecutionState, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %w", err)
	}

	// If no state exists, try to initialize it first
	if state == nil {
		if profileID == "" || scenarioName == "" {
			return nil, fmt.Errorf("no execution state found for task %s; provide profile_id and scenario_name to initialize", taskID)
		}

		log.Printf("Auto Steer: Initializing execution state for task %s before seeking (profile: %s, scenario: %s)", taskID, profileID, scenarioName)
		state, err = o.StartExecution(taskID, profileID, scenarioName)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize execution state: %w", err)
		}
		log.Printf("Auto Steer: Initialized task %s at phase 0, now seeking to phase %d iteration %d", taskID, phaseIndex, phaseIteration)
	}

	profile, err := o.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if len(profile.Phases) == 0 {
		return nil, fmt.Errorf("profile has no phases")
	}

	// Clamp phase index
	if phaseIndex < 0 {
		phaseIndex = 0
	}
	if phaseIndex >= len(profile.Phases) {
		phaseIndex = len(profile.Phases) - 1
	}

	// Clamp iteration
	maxIterations := profile.Phases[phaseIndex].MaxIterations
	if maxIterations < 1 {
		maxIterations = 1
	}
	if phaseIteration < 0 {
		phaseIteration = 0
	}
	if phaseIteration > maxIterations {
		phaseIteration = maxIterations
	}

	// Derive total iteration count up to the target position
	totalIterations := 0
	for idx := 0; idx < phaseIndex; idx++ {
		iter := profile.Phases[idx].MaxIterations
		if idx < len(state.PhaseHistory) && state.PhaseHistory[idx].Iterations > 0 {
			iter = state.PhaseHistory[idx].Iterations
		}
		totalIterations += iter
	}
	totalIterations += phaseIteration

	// Update state
	state.CurrentPhaseIndex = phaseIndex
	state.CurrentPhaseIteration = phaseIteration
	state.AutoSteerIteration = totalIterations
	state.PhaseStartMetrics = state.Metrics
	state.PhaseStartedAt = time.Now()
	state.LastUpdated = time.Now()

	// Truncate history if seeking backwards
	if len(state.PhaseHistory) > phaseIndex {
		state.PhaseHistory = state.PhaseHistory[:phaseIndex]
	}

	if err := o.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	return state, nil
}

// AdvancePhase advances to the next phase or completes execution.
// Implements ExecutionEngineAPI.
func (o *ExecutionOrchestrator) AdvancePhase(taskID string, scenarioName string) (*PhaseAdvanceResult, error) {
	// Get execution state
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
		return nil, fmt.Errorf("no execution state found for task: %s", taskID)
	}

	// Get profile
	profile, err := o.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Complete current phase
	if state.CurrentPhaseIndex < len(profile.Phases) {
		currentPhase := profile.Phases[state.CurrentPhaseIndex]
		stopReason := o.phaseCoordinator.DetermineStopReason(state.CurrentPhaseIteration, currentPhase.MaxIterations)
		if err := o.stateManager.RecordPhaseCompletion(state, currentPhase, stopReason); err != nil {
			return nil, fmt.Errorf("failed to complete phase: %w", err)
		}
	}

	// Check if there are more phases
	if state.CurrentPhaseIndex+1 >= len(profile.Phases) {
		// All phases completed - finalize execution
		if err := o.stateManager.FinalizeExecution(state, scenarioName); err != nil {
			return nil, fmt.Errorf("failed to complete execution: %w", err)
		}

		return &PhaseAdvanceResult{
			Success:   true,
			Completed: true,
			Message:   "All phases completed successfully",
		}, nil
	}

	// Evaluate quality gates before advancing
	gateResults := o.phaseCoordinator.EvaluateQualityGates(profile.QualityGates, state.Metrics)
	halt, failedGate, message := o.phaseCoordinator.ShouldHaltOnQualityGates(gateResults)
	if halt {
		return &PhaseAdvanceResult{
			Success: false,
			Message: fmt.Sprintf("Quality gate '%s' failed: %s", failedGate, message),
		}, nil
	}

	// Advance to next phase
	o.stateManager.AdvanceToNextPhase(state)

	// Save updated state
	if err := o.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	return &PhaseAdvanceResult{
		Success:        true,
		NextPhaseIndex: state.CurrentPhaseIndex,
		Completed:      false,
		Message:        fmt.Sprintf("Advanced to phase %d: %s", state.CurrentPhaseIndex+1, profile.Phases[state.CurrentPhaseIndex].Mode),
	}, nil
}

// GetCurrentMode returns the current steering mode for a task.
// Implements ExecutionEngineAPI.
func (o *ExecutionOrchestrator) GetCurrentMode(taskID string) (SteerMode, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
		return "", nil
	}

	profile, err := o.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	if state.CurrentPhaseIndex >= len(profile.Phases) {
		return "", nil
	}

	return profile.Phases[state.CurrentPhaseIndex].Mode, nil
}

// DeleteExecutionState removes any active execution state for a task.
// Implements ExecutionEngineAPI.
func (o *ExecutionOrchestrator) DeleteExecutionState(taskID string) error {
	return o.stateManager.Delete(taskID)
}

// GetEnhancedPrompt generates the Auto Steer prompt section for a task.
// Returns empty string if the task doesn't have an Auto Steer profile.
func (o *ExecutionOrchestrator) GetEnhancedPrompt(taskID string) (string, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get execution state: %w", err)
	}

	if state == nil {
		return "", nil
	}

	profile, err := o.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	// Generate prompt section
	promptSection := o.promptEnhancer.GenerateAutoSteerSection(state, profile, o.phaseCoordinator.Evaluator())

	return promptSection, nil
}

// GenerateModeSection renders a standalone mode block.
func (o *ExecutionOrchestrator) GenerateModeSection(mode SteerMode) string {
	if o == nil || o.promptEnhancer == nil {
		return ""
	}
	return o.promptEnhancer.GenerateModeSection(mode)
}

// IsAutoSteerActive checks if a task has an active Auto Steer profile.
func (o *ExecutionOrchestrator) IsAutoSteerActive(taskID string) (bool, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return false, fmt.Errorf("failed to get execution state: %w", err)
	}
	return state != nil, nil
}
