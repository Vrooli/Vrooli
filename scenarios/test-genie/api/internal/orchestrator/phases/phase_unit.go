package phases

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"test-genie/internal/orchestrator/workspace"
)

// runUnitPhase executes Go unit tests and shell syntax checks without relying on bash orchestration.
func runUnitPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	var observations []string
	apiDir := filepath.Join(env.ScenarioDir, "api")
	if err := ensureDir(apiDir); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Ensure the api/ directory exists so Go unit tests can run.",
		}
	}

	if err := EnsureCommandAvailable("go"); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install the Go toolchain to execute API unit tests.",
		}
	}

	logPhaseStep(logWriter, "executing go test ./... inside %s", apiDir)
	if err := phaseCommandExecutor(ctx, apiDir, logWriter, "go", "test", "./..."); err != nil {
		return RunReport{
			Err:                   fmt.Errorf("go test ./... failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Fix failing Go tests under api/ before re-running the suite.",
			Observations:          observations,
		}
	}
	observations = append(observations, "go test ./... passed")

	shellTargets := []string{
		filepath.Join(env.ScenarioDir, "cli", "test-genie"),
		filepath.Join(env.ScenarioDir, "test", "lib", "runtime.sh"),
		filepath.Join(env.ScenarioDir, "test", "lib", "orchestrator.sh"),
	}

	if err := EnsureCommandAvailable("bash"); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install bash so shell entrypoints can be linted.",
			Observations:          observations,
		}
	}

	for _, target := range shellTargets {
		if err := ensureFile(target); err != nil {
			return RunReport{
				Err:                   err,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Restore the CLI binary and test/lib scripts so syntax checks can run.",
				Observations:          observations,
			}
		}
		logPhaseStep(logWriter, "running bash -n %s", target)
		if err := phaseCommandExecutor(ctx, "", logWriter, "bash", "-n", target); err != nil {
			return RunReport{
				Err:                   fmt.Errorf("bash -n %s failed: %w", target, err),
				FailureClassification: FailureClassSystem,
				Remediation:           fmt.Sprintf("Fix syntax errors in %s and re-run the suite.", target),
				Observations:          observations,
			}
		}
		observations = append(observations, fmt.Sprintf("bash -n verified: %s", target))
	}

	logPhaseStep(logWriter, "unit validation complete")
	return RunReport{Observations: observations}
}
