package phases

import (
	"context"
	"io"

	"test-genie/internal/business"
	"test-genie/internal/orchestrator/workspace"
)

// runBusinessPhase validates the requirements registry using the business package.
// This includes discovery, parsing, and structural validation of requirement modules.
func runBusinessPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunNativePhase(ctx, env, logWriter, Business,
		business.LoadExpectations,
		func(expectations *business.Expectations) (StandardRunResult, error) {
			runner := business.New(business.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				Expectations: expectations,
			}, business.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		WithNativePhaseReportHook(func(report *RunReport, result StandardRunResult) {
			runResult, _ := result.(*business.RunResult)
			// Typed findings (source=BUSINESS) ride alongside the human-readable
			// observations: structural issues plus registry-drift checks. Findings
			// never change the phase's pass/fail — that stays with the validators.
			report.Findings = businessFindings(env.ScenarioName, env.ScenarioDir, runResult)
		}),
	)
}
