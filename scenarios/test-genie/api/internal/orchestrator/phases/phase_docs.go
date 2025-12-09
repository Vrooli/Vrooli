package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/docs"
	"test-genie/internal/orchestrator/workspace"
)

// runDocsPhase validates Markdown files, mermaid diagrams, links, and absolute path usage.
func runDocsPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary docs.Summary

	report := RunPhase(ctx, logWriter, "docs",
		func() (*docs.RunResult, error) {
			settings, err := docs.LoadSettings(env.ScenarioDir)
			if err != nil {
				return nil, fmt.Errorf("failed to load docs settings: %w", err)
			}

			runner := docs.New(docs.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				Settings:     settings,
			}, docs.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		func(r *docs.RunResult) PhaseResult[docs.Observation] {
			var result docs.RunResult
			summaryText := ""
			if r != nil {
				result = *r
				summary = r.Summary
				summaryText = r.Summary.String()
			}

			return ExtractWithSummary(
				result.Success,
				result.Error,
				result.FailureClass,
				result.Remediation,
				result.Observations,
				"📄",
				fmt.Sprintf("Docs validation completed (%s)", summaryText),
			)
		},
	)

	writePhasePointer(env, "docs", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Docs summary: %s (abs_hits=%d, abs_blocked=%d)", summary.String(), summary.AbsolutePathHits, summary.AbsoluteFailures)
	return report
}
