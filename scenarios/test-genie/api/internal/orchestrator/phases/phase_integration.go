package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/orchestrator/workspace"
)

// Note: expectedPhaseListing was removed - the Go orchestrator no longer validates
// bash phase scripts since it implements all phases natively.

// runIntegrationPhase validates CLI flows and acceptance tests.
// Note: This phase no longer validates bash orchestrator functionality since test-genie
// now implements all test phases natively in Go for portability and consistency.
func runIntegrationPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	var observations []string
	cliPath, err := discoverScenarioCLIBinary(env)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Add an executable CLI binary under cli/ so operators can invoke the scenario.",
		}
	}
	logPhaseStep(logWriter, "cli binary verified: %s", cliPath)
	observations = append(observations, "cli binary executable")

	if err := phaseCommandExecutor(ctx, "", logWriter, cliPath, "help"); err != nil {
		return RunReport{
			Err:                   fmt.Errorf("cli help command failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Run `cli/test-genie help` manually to inspect the error output.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "cli help command succeeded")
	observations = append(observations, "cli help verified")

	versionOutput, err := phaseCommandCapture(ctx, "", logWriter, cliPath, "version")
	if err != nil {
		return RunReport{
			Err:                   fmt.Errorf("cli version command failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure cli/test-genie version works without interactive prompts.",
			Observations:          observations,
		}
	}
	if !strings.Contains(strings.ToLower(versionOutput), "version") {
		return RunReport{
			Err:                   fmt.Errorf("cli version output malformed: %s", strings.TrimSpace(versionOutput)),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Update cli/test-genie to print the version string used by operator tooling.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "cli version output: %s", strings.TrimSpace(versionOutput))
	observations = append(observations, "cli version reported")

	// Note: We no longer validate test/run-tests.sh --list since the Go orchestrator
	// implements phases natively. The run-tests.sh script is kept as a thin compatibility
	// shim but is not required for test execution.
	logPhaseStep(logWriter, "skipping bash orchestrator validation (Go-native phases)")
	observations = append(observations, "go-native phase execution")

	if err := EnsureCommandAvailable("bats"); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install the Bats test runner to execute CLI acceptance suites.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "bats command detected")
	observations = append(observations, "bats runtime available")

	cliDir := filepath.Join(env.ScenarioDir, "cli")
	baseSuite, err := findPrimaryBatsSuite(cliDir, env.ScenarioName)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Add a .bats suite under cli/ to exercise CLI workflows.",
			Observations:          observations,
		}
	}
	baseName := filepath.Base(baseSuite)
	if err := phaseCommandExecutor(ctx, cliDir, logWriter, "bats", "--tap", baseName); err != nil {
		return RunReport{
			Err:                   fmt.Errorf("%s failed: %w", baseName, err),
			FailureClassification: FailureClassSystem,
			Remediation:           fmt.Sprintf("Inspect %s to resolve acceptance test failures.", baseName),
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "%s executed successfully", baseName)
	observations = append(observations, "primary Bats suite passed")

	testSuitesRun, err := runAdditionalBatsSuites(ctx, cliDir, logWriter)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure cli/test contains valid .bats files with readable permissions.",
			Observations:          observations,
		}
	}
	if testSuitesRun > 0 {
		observations = append(observations, fmt.Sprintf("additional Bats suites: %d", testSuitesRun))
	}

	logPhaseStep(logWriter, "integration validation complete")
	return RunReport{Observations: observations}
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
