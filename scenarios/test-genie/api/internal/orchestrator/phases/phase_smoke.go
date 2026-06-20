package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
	"test-genie/internal/smoke"
	"test-genie/internal/smoke/smokeconfig"
)

var smokeRunForPhase = smoke.RunForPhase

// smokeRunResult adapts the UI smoke outcome to StandardRunResult so the smoke
// phase flows through the single RunNativePhase chokepoint (and is measured
// there) instead of hand-rolling a RunReport. SummaryText is empty so no extra
// summary observation is appended — the phase's own observations are the output.
type smokeRunResult struct {
	success      bool
	err          error
	failureClass shared.FailureClass
	remediation  string
	observations []shared.Observation
}

func (r *smokeRunResult) Succeeded() bool                       { return r.success }
func (r *smokeRunResult) Err() error                            { return r.err }
func (r *smokeRunResult) Failure() shared.FailureClass          { return r.failureClass }
func (r *smokeRunResult) RemediationText() string               { return r.remediation }
func (r *smokeRunResult) ObservationList() []shared.Observation { return r.observations }
func (r *smokeRunResult) SummaryText() string                   { return "" }

// runSmokePhase executes the UI smoke test as its own validation phase.
// This validates that a scenario's UI loads correctly, establishes communication
// with the host via iframe-bridge, and produces no critical errors.
func runSmokePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunNativePhase[struct{}](ctx, env, logWriter, Smoke,
		nil,
		func(struct{}) (StandardRunResult, error) {
			return executeSmoke(ctx, env, logWriter), nil
		},
		// Smoke owns its observation output; suppress the generic summary line.
		WithNativePhaseSummaryMessage(func(Name, string) string { return "" }),
	)
}

// executeSmoke runs the UI smoke harness and maps every outcome (disabled,
// execution error, skipped, blocked, stale bundle, failure, success) onto a
// StandardRunResult. It never returns an error to RunNativePhase: failures are
// encoded in the result so the phase keeps its rich, outcome-specific
// remediations instead of the generic "Check smoke configuration" message.
func executeSmoke(ctx context.Context, env workspace.Environment, logWriter io.Writer) StandardRunResult {
	// Check if smoke testing is disabled via configuration
	cfg := smokeconfig.LoadUISmokeConfig(env.ScenarioDir)
	if !cfg.Enabled {
		logPhaseStep(logWriter, "UI smoke testing disabled via .vrooli/testing.json")
		return &smokeRunResult{
			success: true,
			observations: []shared.Observation{
				shared.NewSkipObservation("UI smoke testing disabled via .vrooli/testing.json"),
			},
		}
	}

	logPhaseStep(logWriter, "running UI smoke test for %s", env.ScenarioName)

	// Run the smoke test
	phaseResult, err := smokeRunForPhase(ctx, env.ScenarioName, env.ScenarioDir, env.UIURL, env.RunID, env.CaptureProfile, logWriter)
	if err != nil {
		return &smokeRunResult{
			success:      false,
			err:          err,
			failureClass: shared.FailureClassSystem,
			remediation:  "Ensure the browser-automation-studio scenario is running and the scenario UI is configured (vrooli scenario start browser-automation-studio).",
			observations: []shared.Observation{
				shared.NewErrorObservation(fmt.Sprintf("UI smoke execution failed: %v", err)),
			},
		}
	}

	// Handle different outcomes
	if phaseResult.Skipped {
		return &smokeRunResult{
			success:      true,
			observations: []shared.Observation{shared.NewSkipObservation(phaseResult.Message)},
		}
	}

	if phaseResult.Blocked {
		return &smokeRunResult{
			success:      false,
			err:          phaseResult.ToError(),
			failureClass: shared.FailureClassMisconfiguration,
			remediation:  phaseResult.Message,
			observations: []shared.Observation{shared.NewErrorObservation(phaseResult.Message)},
		}
	}

	if !phaseResult.Success {
		// Check for bundle staleness
		if fresh, reason := phaseResult.GetBundleStatus(); !fresh {
			return &smokeRunResult{
				success:      false,
				err:          fmt.Errorf("ui bundle stale: %s", reason),
				failureClass: shared.FailureClassMisconfiguration,
				remediation:  "Rebuild or restart the UI so bundles are regenerated before re-running smoke tests.",
				observations: []shared.Observation{shared.NewErrorObservation(fmt.Sprintf("UI bundle stale: %s", reason))},
			}
		}

		return &smokeRunResult{
			success:      false,
			err:          phaseResult.ToError(),
			failureClass: shared.FailureClassSystem,
			remediation:  "Investigate the UI smoke failure (see artifacts under coverage/runs/<runID>/ui-smoke/) and fix the underlying issue.",
			observations: []shared.Observation{shared.NewErrorObservation(phaseResult.Message)},
		}
	}

	// Success case
	observations := []shared.Observation{shared.NewSuccessObservation(phaseResult.FormatObservation())}
	if res := phaseResult.Result; res != nil {
		logPhaseStep(logWriter,
			"UI smoke details: url=%s duration=%dms handshake(signaled=%t timeout=%t %dms err=%s) network_failures=%d page_errors=%d console_errors=%d",
			res.UIURL, res.DurationMs, res.Handshake.Signaled, res.Handshake.TimedOut, res.Handshake.DurationMs, res.Handshake.Error,
			res.NetworkFailureCount, res.PageErrorCount, res.ConsoleErrorCount,
		)
		if res.Artifacts != (smoke.ArtifactPaths{}) {
			logPhaseStep(logWriter, "UI smoke artifacts: screenshot=%s console=%s network=%s raw=%s readme=%s",
				res.Artifacts.Screenshot, res.Artifacts.Console, res.Artifacts.Network, res.Artifacts.Raw, res.Artifacts.Readme)
		}
		if res.ConsoleWarningCount > 0 {
			message := fmt.Sprintf("UI smoke captured %d browser console warning(s)", res.ConsoleWarningCount)
			if res.Artifacts.Console != "" {
				message += fmt.Sprintf("; see %s", res.Artifacts.Console)
			}
			observations = append(observations, shared.NewWarningObservation(message))
		}
	}
	logPhaseSuccess(logWriter, "UI smoke test passed")

	return &smokeRunResult{success: true, observations: observations}
}
