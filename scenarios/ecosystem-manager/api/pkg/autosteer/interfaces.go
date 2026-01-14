package autosteer

// This file defines internal component interfaces for dependency injection and testing.
// These interfaces enable unit testing of orchestration components without database or I/O.

// ConditionEvaluatorAPI defines the contract for evaluating stop conditions against metrics.
// Implementations must be stateless and thread-safe.
type ConditionEvaluatorAPI interface {
	// Evaluate evaluates a stop condition against current metrics.
	// Returns true if the condition is satisfied, false otherwise.
	Evaluate(condition StopCondition, metrics MetricsSnapshot) (bool, error)

	// FormatCondition formats a condition as a human-readable string with current metric values.
	FormatCondition(condition StopCondition, metrics MetricsSnapshot) string
}

// Compile-time interface assertion
var _ ConditionEvaluatorAPI = (*ConditionEvaluator)(nil)

// PhaseCoordinatorAPI defines the contract for phase transition decisions.
// Contains pure business logic with no I/O dependencies.
type PhaseCoordinatorAPI interface {
	// ShouldAdvancePhase evaluates whether the current phase should stop
	// based on iteration count and stop conditions.
	ShouldAdvancePhase(phase SteerPhase, metrics MetricsSnapshot, currentIteration int) PhaseAdvanceDecision

	// EvaluateQualityGates evaluates all quality gates against current metrics.
	EvaluateQualityGates(gates []QualityGate, metrics MetricsSnapshot) []QualityGateEvaluation

	// ShouldHaltOnQualityGates checks if any quality gate failures should halt progression.
	ShouldHaltOnQualityGates(evaluations []QualityGateEvaluation) (halt bool, failedGate string, message string)

	// DetermineStopReason determines the stop reason for phase completion.
	DetermineStopReason(currentIteration, maxIterations int) string

	// Evaluator returns the condition evaluator for prompt generation.
	Evaluator() ConditionEvaluatorAPI
}

// Compile-time interface assertion
var _ PhaseCoordinatorAPI = (*PhaseCoordinator)(nil)

// Note: MetricsProvider is defined in repositories.go and provides:
//   CollectMetrics(scenarioName string, phaseLoops, totalLoops int) (*MetricsSnapshot, error)

// IterationEvaluatorAPI defines the contract for evaluating iterations.
// Enables unit testing of ExecutionOrchestrator without real iteration evaluation.
type IterationEvaluatorAPI interface {
	// Evaluate evaluates the current iteration for a task.
	// Collects metrics, increments counters, checks stop conditions, and returns the evaluation.
	Evaluate(taskID string, scenarioName string) (*IterationEvaluation, error)

	// EvaluateWithoutMetricsCollection evaluates using existing metrics in the state.
	// Useful for re-evaluation without triggering metrics collection.
	EvaluateWithoutMetricsCollection(taskID string) (*IterationEvaluation, error)
}

// Compile-time interface assertion
var _ IterationEvaluatorAPI = (*IterationEvaluator)(nil)

// PromptEnhancerAPI defines the contract for generating Auto Steer prompt sections.
// Enables unit testing of ExecutionOrchestrator without filesystem access.
type PromptEnhancerAPI interface {
	// GenerateModeSection renders a standalone section for a specific mode.
	GenerateModeSection(mode SteerMode) string

	// GenerateAutoSteerSection generates the full Auto Steer section for agent prompts.
	GenerateAutoSteerSection(state *ProfileExecutionState, profile *AutoSteerProfile, evaluator ConditionEvaluatorAPI) string

	// GeneratePhaseTransitionMessage generates a message for phase transitions.
	GeneratePhaseTransitionMessage(oldPhase, newPhase SteerPhase, phaseNumber, totalPhases int) string

	// GenerateCompletionMessage generates a message when all phases are complete.
	GenerateCompletionMessage(profile *AutoSteerProfile, state *ProfileExecutionState) string
}

// Compile-time interface assertion
var _ PromptEnhancerAPI = (*PromptEnhancer)(nil)
