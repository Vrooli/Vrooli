package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/integration"
	"test-genie/internal/orchestrator/workspace"
)

// runIntegrationPhase validates CLI flows and acceptance tests using the integration package.
// This includes CLI binary discovery, help/version command validation, and BATS suite execution.
func runIntegrationPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Create the integration runner with command functions
	runner := integration.New(
		integration.Config{
			ScenarioDir:  env.ScenarioDir,
			ScenarioName: env.ScenarioName,
		},
		integration.WithLogger(logWriter),
		integration.WithCommandExecutor(phaseCommandExecutor),
		integration.WithCommandCapture(phaseCommandCapture),
		integration.WithCommandLookup(commandLookup),
	)

	result := runner.Run(ctx)

	// Convert observations from integration package to phases package
	observations := convertIntegrationObservations(result.Observations)

	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: convertIntegrationFailureClass(result.FailureClass),
			Remediation:           result.Remediation,
			Observations:          observations,
		}
	}

	// Final summary
	observations = append(observations, Observation{
		Icon: "✅",
		Text: fmt.Sprintf("Integration validation completed (%s)", result.Summary.String()),
	})

	logPhaseSuccess(logWriter, "Integration validation complete")
	return RunReport{Observations: observations}
}

// convertIntegrationObservations converts integration.Observation slice to phases.Observation slice.
func convertIntegrationObservations(obs []integration.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertIntegrationObservation(o)
	}
	return result
}

// convertIntegrationObservation converts a single integration.Observation to phases.Observation.
func convertIntegrationObservation(o integration.Observation) Observation {
	switch o.Type {
	case integration.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case integration.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case integration.ObservationWarning:
		return NewWarningObservation(o.Message)
	case integration.ObservationError:
		return NewErrorObservation(o.Message)
	case integration.ObservationInfo:
		return NewInfoObservation(o.Message)
	case integration.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// convertIntegrationFailureClass converts integration.FailureClass to phases failure classification string.
func convertIntegrationFailureClass(fc integration.FailureClass) string {
	switch fc {
	case integration.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case integration.FailureClassMissingDependency:
		return FailureClassMissingDependency
	case integration.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}
