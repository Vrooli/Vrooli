package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/integration"
	"test-genie/internal/orchestrator/workspace"
)

// runIntegrationPhase validates CLI flows and acceptance tests using the integration package.
// This includes CLI binary discovery, help/version command validation, and BATS suite execution.
func runIntegrationPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunPhase(ctx, logWriter, "integration",
		func() (*integration.RunResult, error) {
			runner := integration.New(
				integration.Config{
					ScenarioDir:  env.ScenarioDir,
					ScenarioName: env.ScenarioName,
				},
				integration.WithLogger(logWriter),
				integration.WithCommandExecutor(phaseCommandExecutor),
				integration.WithCommandCapture(phaseCommandCapture),
				integration.WithCommandLookup(commandLookup),
			)
			return runner.Run(ctx), nil
		},
		func(r *integration.RunResult) PhaseResult[integration.Observation] {
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"✅",
				fmt.Sprintf("Integration validation completed (%s)", r.Summary.String()),
			)
		},
	)
}
