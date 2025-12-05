package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/performance"
)

// runPerformancePhase benchmarks build times using the performance package.
// This includes Go API builds and Node.js UI builds.
func runPerformancePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunPhaseWithExpectations(ctx, env, logWriter, "performance",
		performance.LoadExpectations,
		func(expectations *performance.Expectations) (*performance.RunResult, error) {
			runner := performance.New(performance.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				Expectations: expectations,
				UIURL:        env.UIURL,
			}, performance.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		func(r *performance.RunResult) PhaseResult[performance.Observation] {
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"✅",
				fmt.Sprintf("Performance validation completed (%s)", r.Summary.String()),
			)
		},
	)
}
