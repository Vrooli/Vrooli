package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/structure"
	"test-genie/internal/structure/existence"
)

// runStructurePhase validates the essential scenario layout using the structure package.
// This includes existence checks (directories, files, CLI), content validation (JSON, manifest),
// and UI smoke testing.
func runStructurePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Load expectations from testing.json
	expectations, err := structure.LoadExpectations(env.ScenarioDir)
	if err != nil {
		logPhaseError(logWriter, "Failed to load structure expectations: %v", err)
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix .vrooli/testing.json so structure requirements can be parsed.",
		}
	}

	// Run structure validation using the screaming-architecture package
	// This now handles existence, content, AND smoke validation
	runner := structure.New(structure.Config{
		ScenarioDir:  env.ScenarioDir,
		ScenarioName: env.ScenarioName,
		Expectations: expectations,
	}, structure.WithLogger(logWriter))

	result := runner.Run(ctx)

	// Convert observations from structure package to phases package
	observations := convertObservations(result.Observations)

	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: convertFailureClass(result.FailureClass),
			Remediation:           result.Remediation,
			Observations:          observations,
		}
	}

	// Final summary (runner already adds one, but keep phase-level summary)
	observations = append(observations, Observation{
		Icon: "✅",
		Text: fmt.Sprintf("Structure validation completed (%s)", result.Summary.String()),
	})

	logPhaseSuccess(logWriter, "Structure validation complete")
	return RunReport{Observations: observations}
}

// convertObservations converts structure.Observation slice to phases.Observation slice.
func convertObservations(obs []structure.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertObservation(o)
	}
	return result
}

// convertObservation converts a single structure.Observation to phases.Observation.
func convertObservation(o structure.Observation) Observation {
	switch o.Type {
	case structure.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case structure.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case structure.ObservationWarning:
		return NewWarningObservation(o.Message)
	case structure.ObservationError:
		return NewErrorObservation(o.Message)
	case structure.ObservationInfo:
		return NewInfoObservation(o.Message)
	case structure.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// convertFailureClass converts structure.FailureClass to phases failure classification string.
func convertFailureClass(fc structure.FailureClass) string {
	switch fc {
	case structure.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case structure.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}

// GetCLIApproach returns the CLI approach for the given scenario.
// This is exposed for other phases (like integration) that may need to know.
func GetCLIApproach(scenarioDir, scenarioName string) existence.CLIApproach {
	return existence.DetectCLIApproach(scenarioDir, scenarioName)
}
