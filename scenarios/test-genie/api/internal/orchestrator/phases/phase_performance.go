package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/performance"
)

// runPerformancePhase benchmarks build times using the performance package.
// This includes Go API builds and Node.js UI builds.
func runPerformancePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Load expectations from testing.json
	expectations, err := performance.LoadExpectations(env.ScenarioDir)
	if err != nil {
		logPhaseError(logWriter, "Failed to load performance expectations: %v", err)
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix .vrooli/testing.json so performance settings can be parsed.",
		}
	}

	// Run performance validation using the dedicated package
	// Note: Lighthouse now uses CLI directly, not Browserless
	runner := performance.New(performance.Config{
		ScenarioDir:  env.ScenarioDir,
		ScenarioName: env.ScenarioName,
		Expectations: expectations,
		UIURL:        env.UIURL,
	}, performance.WithLogger(logWriter))

	result := runner.Run(ctx)

	// Convert observations from performance package to phases package
	observations := convertPerformanceObservations(result.Observations)

	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: convertPerformanceFailureClass(result.FailureClass),
			Remediation:           result.Remediation,
			Observations:          observations,
		}
	}

	// Final summary
	observations = append(observations, Observation{
		Icon: "✅",
		Text: fmt.Sprintf("Performance validation completed (%s)", result.Summary.String()),
	})

	logPhaseSuccess(logWriter, "Performance validation complete")
	return RunReport{Observations: observations}
}

// convertPerformanceObservations converts performance.Observation slice to phases.Observation slice.
func convertPerformanceObservations(obs []performance.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertPerformanceObservation(o)
	}
	return result
}

// convertPerformanceObservation converts a single performance.Observation to phases.Observation.
func convertPerformanceObservation(o performance.Observation) Observation {
	switch o.Type {
	case performance.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case performance.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case performance.ObservationWarning:
		return NewWarningObservation(o.Message)
	case performance.ObservationError:
		return NewErrorObservation(o.Message)
	case performance.ObservationInfo:
		return NewInfoObservation(o.Message)
	case performance.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// convertPerformanceFailureClass converts performance.FailureClass to phases failure classification string.
func convertPerformanceFailureClass(fc performance.FailureClass) string {
	switch fc {
	case performance.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case performance.FailureClassMissingDependency:
		return FailureClassMissingDependency
	case performance.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}
