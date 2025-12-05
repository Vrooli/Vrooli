package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/lint"
	"test-genie/internal/orchestrator/workspace"
)

// runLintPhase performs static analysis (linting and type checking) for Go, Node.js, and Python.
func runLintPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunPhase(ctx, logWriter, "lint",
		func() (*lint.RunResult, error) {
			// Load lint settings from testing.json
			settings, err := lint.LoadSettings(env.ScenarioDir)
			if err != nil {
				return nil, fmt.Errorf("failed to load lint settings: %w", err)
			}

			config := lint.Config{
				ScenarioDir:   env.ScenarioDir,
				ScenarioName:  env.ScenarioName,
				CommandLookup: commandLookup,
				Settings:      settings,
			}

			runner := lint.New(config, lint.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		func(r *lint.RunResult) PhaseResult[lint.Observation] {
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"",
				fmt.Sprintf("Lint validation completed (%d languages, %d issues)", r.Summary.TotalChecks(), r.Summary.TotalIssues()),
			)
		},
	)
}
