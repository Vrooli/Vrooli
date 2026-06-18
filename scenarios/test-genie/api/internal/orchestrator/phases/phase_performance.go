package phases

import (
	"context"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/performance"
)

// runPerformancePhase benchmarks build times using the performance package.
// This includes Go API builds and Node.js UI builds.
func runPerformancePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunNativePhase(ctx, env, logWriter, Performance,
		performance.LoadExpectations,
		func(expectations *performance.Expectations) (StandardRunResult, error) {
			runner := performance.New(performance.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				RunID:        env.RunID,
				Expectations: expectations,
				UIURL:        env.UIURL,
			}, performance.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
	)
}
