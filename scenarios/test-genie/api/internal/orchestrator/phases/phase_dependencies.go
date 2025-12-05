package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/dependencies"
	"test-genie/internal/dependencies/resources"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

// runDependenciesPhase validates runtime/tool requirements using the dependencies package.
// This includes baseline commands, language runtimes, package managers, and resources.
func runDependenciesPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunPhase(ctx, logWriter, "dependencies",
		func() (*dependencies.RunResult, error) {
			config := dependencies.Config{
				ScenarioDir:   env.ScenarioDir,
				ScenarioName:  env.ScenarioName,
				AppRoot:       env.AppRoot,
				CommandLookup: commandLookup,
			}

			opts := []dependencies.Option{
				dependencies.WithLogger(logWriter),
			}

			// Try to set up resource health checking if vrooli CLI is available
			resourceChecker := createResourceChecker(ctx, env, logWriter)
			if resourceChecker != nil {
				opts = append(opts, dependencies.WithResourceChecker(resourceChecker))
			}

			runner := dependencies.New(config, opts...)
			return runner.Run(ctx), nil
		},
		func(r *dependencies.RunResult) PhaseResult[dependencies.Observation] {
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"",
				fmt.Sprintf("Dependency validation completed (%d checks)", r.Summary.TotalChecks()),
			)
		},
	)
}

// createResourceChecker creates a resource health checker if the vrooli CLI is available.
func createResourceChecker(ctx context.Context, env workspace.Environment, logWriter io.Writer) resources.HealthChecker {
	// Check if vrooli CLI is available
	if err := EnsureCommandAvailable("vrooli"); err != nil {
		shared.LogWarn(logWriter, "vrooli CLI unavailable, skipping resource health checks: %v", err)
		return nil
	}

	// Determine app root
	appRoot := env.AppRoot
	if appRoot == "" {
		appRoot = workspace.AppRootFromScenario(env.ScenarioDir)
	}

	// Create CLI-based status fetcher
	fetcher := &cliStatusFetcher{
		scenarioName: env.ScenarioName,
		appRoot:      appRoot,
		logWriter:    logWriter,
	}

	return resources.NewChecker(fetcher, logWriter)
}

// cliStatusFetcher implements resources.StatusFetcher using the vrooli CLI.
type cliStatusFetcher struct {
	scenarioName string
	appRoot      string
	logWriter    io.Writer
}

// Fetch implements resources.StatusFetcher.
func (f *cliStatusFetcher) Fetch(ctx context.Context) (*resources.ScenarioStatus, error) {
	return fetchScenarioStatusForResources(ctx, f.scenarioName, f.appRoot, f.logWriter)
}

// fetchScenarioStatusForResources fetches scenario status and converts to resources types.
func fetchScenarioStatusForResources(ctx context.Context, scenarioName, appRoot string, logWriter io.Writer) (*resources.ScenarioStatus, error) {
	shared.LogStep(logWriter, "collecting scenario status via 'vrooli scenario status %s --json'", scenarioName)
	output, err := phaseCommandCapture(ctx, appRoot, nil, "vrooli", "scenario", "status", scenarioName, "--json")
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario status failed: %w", err)
	}

	// Parse directly into resources.ScenarioStatus using shared helper
	var status resources.ScenarioStatus
	if err := ParseJSON(output, &status); err != nil {
		return nil, fmt.Errorf("failed to parse scenario status JSON: %w", err)
	}

	return &status, nil
}
