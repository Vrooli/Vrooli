package phases

import (
	"context"
	"fmt"
	"io"
	"test-genie/internal/lint"
	"test-genie/internal/orchestrator/workspace"
)

// runLintPhase performs static analysis for discovered top-level components.
func runLintPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary string

	report := RunPhase(ctx, logWriter, "lint",
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
				CommandRunner: lintCommandRunner,
				Settings:      settings,
			}

			runner := lint.New(config, lint.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		func(r *lint.RunResult) PhaseResult[lint.Observation] {
			if r != nil {
				summary = fmt.Sprintf("%d components, %d issues", r.Summary.TotalChecks(), r.Summary.TotalIssues())
			}
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"",
				fmt.Sprintf("Lint validation completed (%d components, %d issues)", r.Summary.TotalChecks(), r.Summary.TotalIssues()),
			)
		},
	)

	writePhasePointer(env, "lint", report, map[string]any{"summary": summary}, logWriter)
	return report
}
