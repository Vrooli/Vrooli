package phases

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	vroolicli "github.com/vrooli/vrooli-cli-go"

	"test-genie/internal/dependencies"
	"test-genie/internal/dependencies/resources"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

// cliClient is the shared typed Vrooli CLI client used to read live resource
// health for the dependency phase.
var cliClient = vroolicli.New()

// runDependenciesPhase validates runtime/tool requirements using the dependencies package.
// This includes baseline commands, language runtimes, package managers, and resources.
func runDependenciesPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary string

	report := RunPhase(ctx, logWriter, "dependencies",
		func() (*dependencies.RunResult, error) {
			config := dependencies.Config{
				ScenarioDir:                      env.ScenarioDir,
				ScenarioName:                     env.ScenarioName,
				AppRoot:                          env.AppRoot,
				CommandLookup:                    commandLookup,
				SkipResourceHealthWhenNoRequired: env.Mapping.HasLogicalPlacement(),
			}

			opts := []dependencies.Option{
				dependencies.WithLogger(logWriter),
			}

			// Try to set up resource health checking if vrooli CLI is available
			resourceChecker := createResourceChecker(env, logWriter)
			if resourceChecker != nil {
				opts = append(opts, dependencies.WithResourceChecker(resourceChecker))
			}

			runner := dependencies.New(config, opts...)
			return runner.Run(ctx), nil
		},
		func(r *dependencies.RunResult) PhaseResult[dependencies.Observation] {
			if r != nil {
				summary = fmt.Sprintf("%d checks", r.Summary.TotalChecks())
			}
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

	writePhasePointer(env, "dependencies", report, map[string]any{"summary": summary}, logWriter)
	return report
}

// createResourceChecker builds a resource health checker when the vrooli CLI is
// available. The scenario's required resources come from its service manifest;
// live health comes from `vrooli resource status --json` via the shared client.
// Returns nil (health check skipped) when the CLI or manifest is unavailable.
func createResourceChecker(env workspace.Environment, logWriter io.Writer) resources.HealthChecker {
	if err := EnsureCommandAvailable("vrooli"); err != nil {
		shared.LogWarn(logWriter, "vrooli CLI unavailable, skipping resource health checks: %v", err)
		return nil
	}

	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	manifest, err := workspace.LoadServiceManifest(manifestPath)
	if err != nil {
		shared.LogWarn(logWriter, "could not load service manifest, skipping resource health checks: %v", err)
		return nil
	}

	return resources.NewChecker(manifest.RequiredResources(), cliClient, logWriter)
}
