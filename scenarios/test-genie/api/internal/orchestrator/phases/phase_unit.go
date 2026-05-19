package phases

import (
	"context"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/unit"
)

// runUnitPhase executes language-specific unit tests using the unit package.
// This is a thin orchestrator that delegates to specialized sub-packages
// for each language (Go, Node.js, Python, Shell).
func runUnitPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary string

	report := RunPhase(ctx, logWriter, "unit",
		func() (*unit.RunResult, error) {
			runner := unit.New(unit.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
			}, unit.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		func(r *unit.RunResult) PhaseResult[unit.Observation] {
			if r != nil {
				summary = r.Summary.String()
			}
			return ExtractSimple(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
			)
		},
	)

	writePhasePointer(env, "unit", report, map[string]any{"summary": summary}, logWriter)
	return report
}
