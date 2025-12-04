package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/smoke"
	"test-genie/internal/smoke/smokeconfig"
)

// runSmokePhase executes the UI smoke test as its own validation phase.
// This validates that a scenario's UI loads correctly, establishes communication
// with the host via iframe-bridge, and produces no critical errors.
func runSmokePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	// Check if smoke testing is disabled via configuration
	cfg := smokeconfig.LoadUISmokeConfig(env.ScenarioDir)
	if !cfg.Enabled {
		logPhaseStep(logWriter, "UI smoke testing disabled via .vrooli/testing.json")
		return RunReport{
			Observations: []Observation{
				NewSkipObservation("UI smoke testing disabled via .vrooli/testing.json"),
			},
		}
	}

	logPhaseStep(logWriter, "running UI smoke test for %s", env.ScenarioName)

	// Run the smoke test
	phaseResult, err := smoke.RunForPhase(ctx, env.ScenarioName, env.ScenarioDir, logWriter)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
			Remediation:           "Check browserless availability and scenario UI configuration.",
			Observations: []Observation{
				NewErrorObservation(fmt.Sprintf("UI smoke execution failed: %v", err)),
			},
		}
	}

	var observations []Observation

	// Handle different outcomes
	if phaseResult.Skipped {
		observations = append(observations, NewSkipObservation(phaseResult.Message))
		return RunReport{Observations: observations}
	}

	if phaseResult.Blocked {
		observations = append(observations, NewErrorObservation(phaseResult.Message))
		return RunReport{
			Err:                   phaseResult.ToError(),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           phaseResult.Message,
			Observations:          observations,
		}
	}

	if !phaseResult.Success {
		// Check for bundle staleness
		if fresh, reason := phaseResult.GetBundleStatus(); !fresh {
			observations = append(observations, NewErrorObservation(fmt.Sprintf("UI bundle stale: %s", reason)))
			return RunReport{
				Err:                   fmt.Errorf("ui bundle stale: %s", reason),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Rebuild or restart the UI so bundles are regenerated before re-running smoke tests.",
				Observations:          observations,
			}
		}

		observations = append(observations, NewErrorObservation(phaseResult.Message))
		return RunReport{
			Err:                   phaseResult.ToError(),
			FailureClassification: FailureClassSystem,
			Remediation:           "Investigate the UI smoke failure (see artifacts in coverage/ui-smoke/) and fix the underlying issue.",
			Observations:          observations,
		}
	}

	// Success case
	observations = append(observations, NewSuccessObservation(phaseResult.FormatObservation()))
	logPhaseSuccess(logWriter, "UI smoke test passed")

	return RunReport{Observations: observations}
}
