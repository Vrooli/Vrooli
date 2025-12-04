package phases

import (
	"context"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/unit"
)

// runUnitPhase executes language-specific unit tests using the unit package.
// This is a thin orchestrator that delegates to specialized sub-packages
// for each language (Go, Node.js, Python, Shell).
func runUnitPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Create unit test runner with configuration
	runner := unit.New(unit.Config{
		ScenarioDir:  env.ScenarioDir,
		ScenarioName: env.ScenarioName,
	}, unit.WithLogger(logWriter))

	// Execute all unit tests
	result := runner.Run(ctx)

	// Convert unit.RunResult to phases.RunReport
	observations := convertUnitObservations(result.Observations)

	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: convertUnitFailureClass(result.FailureClass),
			Remediation:           result.Remediation,
			Observations:          observations,
		}
	}

	logPhaseStep(logWriter, "unit validation complete")
	return RunReport{Observations: observations}
}

// convertUnitObservations converts unit.Observation slice to phases.Observation slice.
func convertUnitObservations(obs []unit.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertUnitObservation(o)
	}
	return result
}

// convertUnitObservation converts a single unit.Observation to phases.Observation.
func convertUnitObservation(o unit.Observation) Observation {
	switch o.Type {
	case unit.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case unit.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case unit.ObservationWarning:
		return NewWarningObservation(o.Message)
	case unit.ObservationError:
		return NewErrorObservation(o.Message)
	case unit.ObservationInfo:
		return NewInfoObservation(o.Message)
	case unit.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// convertUnitFailureClass converts unit.FailureClass to phases failure classification string.
func convertUnitFailureClass(fc unit.FailureClass) string {
	switch fc {
	case unit.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case unit.FailureClassMissingDependency:
		return FailureClassMissingDependency
	case unit.FailureClassTestFailure:
		return FailureClassSystem // Map test failures to system for now
	case unit.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}
