package phases

import (
	"context"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks"
)

// runPlaybooksPhase executes BAS playbook workflows using the playbooks package.
// This includes loading the registry, executing workflows via BAS API, and
// writing results for requirements coverage tracking.
func runPlaybooksPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return RunPhase(ctx, logWriter, "playbooks",
		func() (*playbooks.RunResult, error) {
			runner := playbooks.New(playbooks.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				TestDir:      env.TestDir,
				AppRoot:      env.AppRoot,
			},
				playbooks.WithLogger(logWriter),
				playbooks.WithPortResolver(func(ctx context.Context, scenarioName, portName string) (string, error) {
					return ResolveScenarioPort(ctx, logWriter, scenarioName, portName)
				}),
				playbooks.WithScenarioStarter(func(ctx context.Context, scenario string) error {
					return StartScenario(ctx, scenario, logWriter)
				}),
				playbooks.WithUIBaseURLResolver(func(ctx context.Context, scenarioName string) (string, error) {
					return ResolveScenarioBaseURL(ctx, logWriter, scenarioName)
				}),
			)
			return runner.Run(ctx), nil
		},
		func(r *playbooks.RunResult) PhaseResult[playbooks.Observation] {
			return ExtractSimple(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
			)
		},
	)
}
