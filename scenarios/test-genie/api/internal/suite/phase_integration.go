package suite

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var expectedPhaseListing = []string{"structure", "dependencies", "unit", "integration", "business", "performance"}

// runIntegrationPhase validates CLI flows and acceptance tests without delegating to bash orchestrators.
func runIntegrationPhase(ctx context.Context, env PhaseEnvironment, logWriter io.Writer) PhaseRunReport {
	if err := ctx.Err(); err != nil {
		return PhaseRunReport{Err: err, FailureClassification: failureClassSystem}
	}

	var observations []string
	cliPath := filepath.Join(env.ScenarioDir, "cli", "test-genie")
	if err := ensureExecutable(cliPath); err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Ensure cli/test-genie exists and is executable so operators can invoke the scenario CLI.",
		}
	}
	logPhaseStep(logWriter, "cli binary verified: %s", cliPath)
	observations = append(observations, "cli binary executable")

	if err := phaseCommandExecutor(ctx, "", logWriter, cliPath, "help"); err != nil {
		return PhaseRunReport{
			Err:                   fmt.Errorf("cli help command failed: %w", err),
			FailureClassification: failureClassSystem,
			Remediation:           "Run `cli/test-genie help` manually to inspect the error output.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "cli help command succeeded")
	observations = append(observations, "cli help verified")

	versionOutput, err := phaseCommandCapture(ctx, "", logWriter, cliPath, "version")
	if err != nil {
		return PhaseRunReport{
			Err:                   fmt.Errorf("cli version command failed: %w", err),
			FailureClassification: failureClassSystem,
			Remediation:           "Ensure cli/test-genie version works without interactive prompts.",
			Observations:          observations,
		}
	}
	if !strings.Contains(strings.ToLower(versionOutput), "version") {
		return PhaseRunReport{
			Err:                   fmt.Errorf("cli version output malformed: %s", strings.TrimSpace(versionOutput)),
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Update cli/test-genie to print the version string used by operator tooling.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "cli version output: %s", strings.TrimSpace(versionOutput))
	observations = append(observations, "cli version reported")

	runnerPath := filepath.Join(env.TestDir, "run-tests.sh")
	if err := ensureExecutable(runnerPath); err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Restore test/run-tests.sh with execute permissions so phase listings work.",
			Observations:          observations,
		}
	}
	listing, err := phaseCommandCapture(ctx, env.TestDir, logWriter, runnerPath, "--list")
	if err != nil {
		return PhaseRunReport{
			Err:                   fmt.Errorf("failed to list phases: %w", err),
			FailureClassification: failureClassSystem,
			Remediation:           "Run test/run-tests.sh --list to confirm the orchestrator works locally.",
			Observations:          observations,
		}
	}
	loweredListing := strings.ToLower(listing)
	for _, phase := range expectedPhaseListing {
		if !strings.Contains(loweredListing, phase) {
			return PhaseRunReport{
				Err:                   fmt.Errorf("phase '%s' missing from orchestrator --list output", phase),
				FailureClassification: failureClassMisconfiguration,
				Remediation:           "Ensure test/phases contains scripts for all expected phases.",
				Observations:          observations,
			}
		}
	}
	logPhaseStep(logWriter, "phase listing validated via run-tests.sh --list")
	observations = append(observations, "phase listing verified")

	if err := ensureCommandAvailable("bats"); err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMissingDependency,
			Remediation:           "Install the Bats test runner to execute CLI acceptance suites.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "bats command detected")
	observations = append(observations, "bats runtime available")

	cliDir := filepath.Join(env.ScenarioDir, "cli")
	baseSuite := filepath.Join(cliDir, "test-genie.bats")
	if err := ensureFile(baseSuite); err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassMisconfiguration,
			Remediation:           "Restore cli/test-genie.bats so CLI acceptance coverage exists.",
			Observations:          observations,
		}
	}
	if err := phaseCommandExecutor(ctx, cliDir, logWriter, "bats", "--tap", filepath.Base(baseSuite)); err != nil {
		return PhaseRunReport{
			Err:                   fmt.Errorf("cli/test-genie.bats failed: %w", err),
			FailureClassification: failureClassSystem,
			Remediation:           "Inspect cli/test-genie.bats to resolve acceptance test failures.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "cli/test-genie.bats executed successfully")
	observations = append(observations, "primary Bats suite passed")

	testSuitesRun, err := runAdditionalBatsSuites(ctx, cliDir, logWriter)
	if err != nil {
		return PhaseRunReport{
			Err:                   err,
			FailureClassification: failureClassSystem,
			Remediation:           "Ensure cli/test contains valid .bats files with readable permissions.",
			Observations:          observations,
		}
	}
	if testSuitesRun > 0 {
		observations = append(observations, fmt.Sprintf("additional Bats suites: %d", testSuitesRun))
	}

	logPhaseStep(logWriter, "integration validation complete")
	return PhaseRunReport{Observations: observations}
}

func runAdditionalBatsSuites(ctx context.Context, cliDir string, logWriter io.Writer) (int, error) {
	testDir := filepath.Join(cliDir, "test")
	info, err := os.Stat(testDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read cli/test directory: %w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("cli/test must be a directory, found file at %s", testDir)
	}

	entries, err := os.ReadDir(testDir)
	if err != nil {
		return 0, fmt.Errorf("failed to list cli/test suite files: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bats") {
			continue
		}
		relPath := filepath.Join("test", entry.Name())
		logPhaseStep(logWriter, "executing %s", relPath)
		if err := phaseCommandExecutor(ctx, cliDir, logWriter, "bats", "--tap", relPath); err != nil {
			return count, fmt.Errorf("%s failed: %w", relPath, err)
		}
		count++
	}
	return count, nil
}
