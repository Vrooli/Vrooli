package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/business"
	"test-genie/internal/orchestrator/workspace"
)

// runBusinessPhase validates the requirements registry using the business package.
// This includes discovery, parsing, and structural validation of requirement modules.
func runBusinessPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Load expectations from testing.json
	expectations, err := business.LoadExpectations(env.ScenarioDir)
	if err != nil {
		logPhaseError(logWriter, "Failed to load business expectations: %v", err)
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix .vrooli/testing.json so business requirements can be parsed.",
		}
	}

	// Run business validation using the dedicated package
	runner := business.New(business.Config{
		ScenarioDir:  env.ScenarioDir,
		ScenarioName: env.ScenarioName,
		Expectations: expectations,
	}, business.WithLogger(logWriter))

	result := runner.Run(ctx)

	// Convert observations from business package to phases package
	observations := convertBusinessObservations(result.Observations)

	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: convertBusinessFailureClass(result.FailureClass),
			Remediation:           result.Remediation,
			Observations:          observations,
		}
	}

	// Final summary
	observations = append(observations, Observation{
		Icon: "✅",
		Text: fmt.Sprintf("Business validation completed (%s)", result.Summary.String()),
	})

	logPhaseSuccess(logWriter, "Business validation complete")
	return RunReport{Observations: observations}
}

// convertBusinessObservations converts business.Observation slice to phases.Observation slice.
func convertBusinessObservations(obs []business.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertBusinessObservation(o)
	}
	return result
}

// convertBusinessObservation converts a single business.Observation to phases.Observation.
func convertBusinessObservation(o business.Observation) Observation {
	switch o.Type {
	case business.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case business.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case business.ObservationWarning:
		return NewWarningObservation(o.Message)
	case business.ObservationError:
		return NewErrorObservation(o.Message)
	case business.ObservationInfo:
		return NewInfoObservation(o.Message)
	case business.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// convertBusinessFailureClass converts business.FailureClass to phases failure classification string.
func convertBusinessFailureClass(fc business.FailureClass) string {
	switch fc {
	case business.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case business.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}
