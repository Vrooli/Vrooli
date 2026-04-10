package autosteer

import (
	"errors"
	"log"
)

// PhaseCoordinator handles phase transition logic and quality gate evaluation.
// It contains pure business logic with no I/O dependencies.
type PhaseCoordinator struct {
	evaluator ConditionEvaluatorAPI
}

// NewPhaseCoordinator creates a new PhaseCoordinator
func NewPhaseCoordinator(evaluator ConditionEvaluatorAPI) *PhaseCoordinator {
	return &PhaseCoordinator{
		evaluator: evaluator,
	}
}

// Evaluator returns the condition evaluator for use by other components.
func (c *PhaseCoordinator) Evaluator() ConditionEvaluatorAPI {
	return c.evaluator
}

// PhaseAdvanceDecision represents the outcome of evaluating whether to advance phases
type PhaseAdvanceDecision struct {
	ShouldStop bool
	Reason     string // "max_iterations", "condition_met", "continue"
}

// ShouldAdvancePhase evaluates whether the current phase should stop based on iteration count and stop conditions.
// Returns decision with reason.
func (c *PhaseCoordinator) ShouldAdvancePhase(phase SteerPhase, metrics MetricsSnapshot, currentIteration int) PhaseAdvanceDecision {
	// Check if max iterations reached
	if currentIteration >= phase.MaxIterations {
		return PhaseAdvanceDecision{
			ShouldStop: true,
			Reason:     "max_iterations",
		}
	}

	// Evaluate stop conditions
	for i, condition := range phase.StopConditions {
		result, err := c.evaluator.Evaluate(condition, metrics)
		if err != nil {
			var unavailable *MetricUnavailableError
			if errors.As(err, &unavailable) {
				// Non-fatal: metric is unavailable. Log and continue evaluating the next condition.
				log.Printf("PhaseCoordinator: condition %d skipped due to unavailable metric '%s': %v", i, unavailable.Metric, unavailable.Error())
				continue
			}
			// For other errors, log and continue (don't block advancement)
			log.Printf("PhaseCoordinator: condition %d evaluation error: %v", i, err)
			continue
		}

		if result {
			return PhaseAdvanceDecision{
				ShouldStop: true,
				Reason:     "condition_met",
			}
		}
	}

	// Continue in current phase
	return PhaseAdvanceDecision{
		ShouldStop: false,
		Reason:     "continue",
	}
}

// QualityGateEvaluation represents the result of evaluating a single quality gate
type QualityGateEvaluation struct {
	GateName string
	Passed   bool
	Message  string
	Action   QualityGateAction
}

// EvaluateQualityGates evaluates all quality gates against current metrics.
// Returns results for each gate. Does not fail on individual gate evaluation errors.
func (c *PhaseCoordinator) EvaluateQualityGates(gates []QualityGate, metrics MetricsSnapshot) []QualityGateEvaluation {
	results := make([]QualityGateEvaluation, 0, len(gates))

	for _, gate := range gates {
		passed, err := c.evaluator.Evaluate(gate.Condition, metrics)
		if err != nil {
			// Gate evaluation failed - treat as not passed with warning action
			log.Printf("PhaseCoordinator: quality gate '%s' evaluation error: %v", gate.Name, err)
			results = append(results, QualityGateEvaluation{
				GateName: gate.Name,
				Passed:   false,
				Message:  "Gate evaluation failed: " + err.Error(),
				Action:   ActionWarn, // Don't halt on evaluation errors
			})
			continue
		}

		results = append(results, QualityGateEvaluation{
			GateName: gate.Name,
			Passed:   passed,
			Message:  gate.Message,
			Action:   gate.FailureAction,
		})
	}

	return results
}

// ShouldHaltOnQualityGates checks if any quality gate failures should halt progression
func (c *PhaseCoordinator) ShouldHaltOnQualityGates(evaluations []QualityGateEvaluation) (halt bool, failedGate string, message string) {
	for _, eval := range evaluations {
		if !eval.Passed && eval.Action == ActionHalt {
			return true, eval.GateName, eval.Message
		}
	}
	return false, "", ""
}

// DetermineStopReason determines the stop reason for phase completion based on iteration and max
func (c *PhaseCoordinator) DetermineStopReason(currentIteration, maxIterations int) string {
	if currentIteration < maxIterations {
		return "condition_met"
	}
	return "max_iterations"
}
