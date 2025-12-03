package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/structure"
	"test-genie/internal/structure/existence"
	"test-genie/internal/uismoke"
)

// UISmokeModeNative enables the native Go implementation of UI smoke tests.
// Defaults to true. Set TEST_GENIE_UI_SMOKE_NATIVE=false to use legacy cached results.
var UISmokeModeNative = os.Getenv("TEST_GENIE_UI_SMOKE_NATIVE") != "false"

// runStructurePhase validates the essential scenario layout using the structure package.
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

	// Run structure validation using the new screaming-architecture package
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

	// Section: UI Smoke Test
	observations = append(observations, NewSectionObservation("🌐", "Running UI smoke test..."))
	logPhaseInfo(logWriter, "Checking UI smoke telemetry...")

	smokeResult := enforceUISmokeTelemetry(ctx, env, expectations, logWriter)
	if smokeResult.failure != nil {
		smokeResult.failure.Observations = append(observations, smokeResult.failure.Observations...)
		return *smokeResult.failure
	} else if smokeResult.skipped {
		logPhaseStep(logWriter, "⏭️  %s", smokeResult.observation)
		observations = append(observations, NewSkipObservation(smokeResult.observation))
	} else if smokeResult.observation != "" {
		logPhaseSuccess(logWriter, "%s", smokeResult.observation)
		observations = append(observations, NewSuccessObservation(smokeResult.observation))
	} else {
		logPhaseStep(logWriter, "⏭️  UI smoke telemetry not configured")
		observations = append(observations, NewSkipObservation("UI smoke telemetry not configured"))
	}

	// Final summary
	observations = append(observations, Observation{Icon: "✅", Text: fmt.Sprintf("Structure validation completed (%s)", result.Summary.String())})

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

// uiSmokeOutcome holds the result of a UI smoke telemetry check.
type uiSmokeOutcome struct {
	observation string     // Human-readable result message
	skipped     bool       // True if the test was skipped (not a failure)
	failure     *RunReport // Non-nil if the test failed or was blocked
}

func enforceUISmokeTelemetry(ctx context.Context, env workspace.Environment, expectations *structure.Expectations, logWriter io.Writer) uiSmokeOutcome {
	// Check if UI smoke is disabled
	if expectations != nil && !expectations.UISmoke.Enabled {
		return uiSmokeOutcome{skipped: true, observation: "UI smoke harness disabled via .vrooli/testing.json"}
	}

	// Use native Go implementation if enabled
	if UISmokeModeNative {
		return runNativeUISmoke(ctx, env, logWriter)
	}

	// Fall back to reading cached results
	return readCachedUISmokeTelemetry(ctx, env, logWriter)
}

// runNativeUISmoke executes the UI smoke test using the native Go implementation.
func runNativeUISmoke(ctx context.Context, env workspace.Environment, logWriter io.Writer) uiSmokeOutcome {
	logPhaseStep(logWriter, "running native UI smoke test...")

	result, err := uismoke.RunForPhase(ctx, env.ScenarioName, env.ScenarioDir, logWriter)
	if err != nil {
		logPhaseWarn(logWriter, "ui smoke execution failed: %v", err)
		return uiSmokeOutcome{failure: &RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
			Remediation:           "Check browserless availability and scenario UI configuration.",
		}}
	}

	if result.Skipped {
		return uiSmokeOutcome{observation: result.Message, skipped: true}
	}

	if result.Blocked {
		return uiSmokeOutcome{failure: &RunReport{
			Err:                   result.ToError(),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           result.Message,
		}}
	}

	if !result.Success {
		// Check if it's a bundle staleness issue
		if fresh, reason := result.GetBundleStatus(); !fresh {
			return uiSmokeOutcome{failure: &RunReport{
				Err:                   fmt.Errorf("ui bundle stale: %s", reason),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Rebuild or restart the UI so bundles are regenerated before re-running structure tests.",
			}}
		}

		return uiSmokeOutcome{failure: &RunReport{
			Err:                   result.ToError(),
			FailureClassification: FailureClassSystem,
			Remediation:           "Investigate the UI smoke failure (see artifacts in coverage/<scenario>/ui-smoke/) and fix the underlying issue.",
		}}
	}

	return uiSmokeOutcome{observation: result.FormatObservation()}
}

// readCachedUISmokeTelemetry reads UI smoke results from cached scenario status.
func readCachedUISmokeTelemetry(ctx context.Context, env workspace.Environment, logWriter io.Writer) uiSmokeOutcome {
	status, err := fetchScenarioStatus(ctx, env, logWriter)
	if err != nil {
		logPhaseWarn(logWriter, "ui smoke telemetry unavailable: %v", err)
		return uiSmokeOutcome{skipped: true, observation: "ui smoke telemetry unavailable"}
	}
	if status.Diagnostics.UISmoke == nil {
		return uiSmokeOutcome{skipped: true, observation: "ui smoke telemetry not reported"}
	}
	smoke := status.Diagnostics.UISmoke
	if strings.EqualFold(smoke.Status, "passed") {
		if smoke.Bundle != nil && !smoke.Bundle.Fresh {
			reason := strings.TrimSpace(smoke.Bundle.Reason)
			if reason == "" {
				reason = "bundle marked stale"
			}
			return uiSmokeOutcome{failure: &RunReport{
				Err:                   fmt.Errorf("ui bundle stale: %s", reason),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Rebuild or restart the UI so bundles are regenerated before re-running structure tests.",
			}}
		}
		duration := int(smoke.DurationMs)
		if duration < 0 {
			duration = 0
		}
		return uiSmokeOutcome{observation: fmt.Sprintf("ui smoke passed (%dms)", duration)}
	}
	if strings.EqualFold(smoke.Status, "skipped") {
		return uiSmokeOutcome{skipped: true, observation: smoke.Message}
	}
	message := strings.TrimSpace(smoke.Message)
	if message == "" {
		message = "UI smoke reported a failure"
	}
	return uiSmokeOutcome{failure: &RunReport{
		Err:                   fmt.Errorf("ui smoke status '%s': %s", smoke.Status, message),
		FailureClassification: FailureClassSystem,
		Remediation:           "Investigate the UI smoke failure (see scenario status diagnostics) and restart the scenario before retrying.",
	}}
}

// GetCLIApproach returns the CLI approach for the given scenario.
// This is exposed for other phases (like integration) that may need to know.
func GetCLIApproach(scenarioDir, scenarioName string) existence.CLIApproach {
	return existence.DetectCLIApproach(scenarioDir, scenarioName)
}
