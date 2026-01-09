package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/business"
	"test-genie/internal/orchestrator/workspace"
)

// runBusinessPhase validates the requirements registry using the business package.
// This includes discovery, parsing, and structural validation of requirement modules.
func runBusinessPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary string

	report := RunPhaseWithExpectations(ctx, env, logWriter, "business",
		business.LoadExpectations,
		func(expectations *business.Expectations) (*business.RunResult, error) {
			runner := business.New(business.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				Expectations: expectations,
			}, business.WithLogger(logWriter))
			return runner.Run(ctx), nil
		},
		func(r *business.RunResult) PhaseResult[business.Observation] {
			if r != nil {
				summary = r.Summary.String()
			}
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"✅",
				fmt.Sprintf("Business validation completed (%s)", r.Summary.String()),
			)
		},
	)

	writePhasePointer(env, "business", report, map[string]any{"summary": summary}, logWriter)
	return report
}
