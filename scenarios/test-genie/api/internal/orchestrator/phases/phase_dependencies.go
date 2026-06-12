package phases

import (
	"context"
	"fmt"
	"io"

	vroolicli "github.com/vrooli/vrooli-cli-go"

	"test-genie/internal/dependencies"
	"test-genie/internal/orchestrator/workspace"
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
				ScenarioStatusFetcher:            cliClient,
				ResourceStatusFetcher:            cliClient,
			}

			opts := []dependencies.Option{
				dependencies.WithLogger(logWriter),
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
