package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks"
	"test-genie/internal/playbooks/config"
	"test-genie/internal/playbooks/isolation"
	"test-genie/internal/shared"

	playbookregistry "test-genie/internal/playbooks/registry"
)

// isolationProvider lets tests stub isolation without requiring Docker.
type isolationProvider interface {
	Prepare(ctx context.Context) (*isolation.Result, error)
}

// isolationManagerFactory creates the default isolation manager.
var isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
	return isolation.NewManager(cfg)
}

type staticRegistryLoader struct {
	registry playbooks.Registry
}

func (s staticRegistryLoader) Load() (playbooks.Registry, error) {
	return s.registry, nil
}

// runPlaybooksPhase executes BAS playbook workflows using the playbooks package.
// This includes loading the registry, executing workflows via BAS API, and
// writing results for requirements coverage tracking.
func runPlaybooksPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
		}
	}

	// Load playbooks configuration from testing.json
	playbooksCfg, err := config.Load(env.ScenarioDir)
	if err != nil {
		logPhaseStep(logWriter, "failed to load playbooks config: %v", err)
		playbooksCfg = config.Default()
	}

	if playbooksCfg != nil && !playbooksCfg.Enabled {
		shared.LogWarn(logWriter, "playbooks phase disabled via .vrooli/testing.json (playbooks.enabled=false)")
		return RunReport{
			Observations: []Observation{
				NewSkipObservation("playbooks phase disabled via .vrooli/testing.json"),
			},
		}
	}

	retainIsolation := isolation.ShouldRetainFromEnv()

	// Honor skip flag before provisioning isolation or restarting the scenario.
	if os.Getenv("TEST_GENIE_SKIP_PLAYBOOKS") == "1" {
		shared.LogWarn(logWriter, "playbooks phase disabled via TEST_GENIE_SKIP_PLAYBOOKS (skipping isolation/restart)")
		return RunReport{
			Observations: []Observation{
				NewSkipObservation("playbooks phase disabled via TEST_GENIE_SKIP_PLAYBOOKS"),
			},
		}
	}

	registry, err := playbookregistry.NewLoader(env.ScenarioDir).Load()
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Regenerate bas/registry.json via playbook builder.",
		}
	}

	if len(registry.Playbooks) == 0 {
		shared.LogInfo(logWriter, "playbooks registry contains no workflows; skipping isolation/restart")
		return runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, nil)
	}

	if registry.UsesObserverMode() {
		shared.LogInfo(logWriter, "playbooks registry execution_mode=%s; skipping isolation/restart", registry.NormalizedExecutionMode())
		return runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, nil)
	}

	if env.ScenarioName == "test-genie" {
		return RunReport{
			Err:                   fmt.Errorf("playbooks for %s require isolation/restart, which would terminate the active test-genie API process", env.ScenarioName),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Set bas/registry.json metadata.execution_mode to \"observer\" for self-tests, or execute playbooks against a different target scenario.",
		}
	}

	// Determine which resources are actually needed from service manifest.
	needs := resolveDBNeeds(ctx, env, logWriter)

	// Provision isolated resources for the playbooks run based on the target
	// scenario manifest (Postgres, Redis, and/or SQLite).
	isoManager := isolationManagerFactory(isolation.Config{
		ScenarioName:    env.ScenarioName,
		RequirePostgres: needs.RequirePostgres,
		RequireRedis:    needs.RequireRedis,
		RequireSQLite:   needs.RequireSQLite,
		SQLiteEnvVars:   needs.SQLiteEnvVars,
		Retain:          retainIsolation,
		LogWriter:       logWriter,
		Timeout:         2 * time.Minute,
	})

	isoResult, err := isoManager.Prepare(ctx)
	if err != nil {
		return RunReport{
			Err:                   fmt.Errorf("failed to prepare playbooks isolation: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure Docker is available for testcontainers or provide access to start the temporary database and cache resources required by the target scenario.",
		}
	}

	restoreEnv := isolation.ApplyEnv(isoResult.Env)
	envApplied := true
	shared.LogStep(logWriter, "playbooks isolation ready (run=%s)", isoResult.RunID)
	for _, res := range isoResult.Resources {
		shared.LogInfo(logWriter, "  %s -> %s", res.Name, res.Endpoint)
		if retainIsolation && len(res.InspectCommands) > 0 {
			for _, cmd := range res.InspectCommands {
				shared.LogInfo(logWriter, "    inspect: %s", cmd)
			}
		}
	}

	// Apply optional SQL migrations for the temp database before restarting the scenario.
	if err := applyPlaybooksMigrations(ctx, env, needs, logWriter); err != nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("failed to apply playbooks migrations: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure psql is available and migrations under bas/seeds/migrations/ are valid.",
		}
	}

	if env.TargetRuntime == nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("target runtime manager is not configured"),
			FailureClassification: FailureClassSystem,
			Remediation:           "Run playbooks through test-genie execute so the target scenario lifecycle can be managed.",
		}
	}

	// Restart the target scenario so it picks up the temporary resources.
	if err := env.TargetRuntime.RestartWithEnv(ctx, isoResult.Env, logWriter); err != nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("failed to restart scenario with playbooks isolation: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Check lifecycle logs for restart failures and ensure the scenario can connect to the temporary resources provisioned for the playbooks run.",
		}
	}

	// Isolation env not needed for BAS; clear before running the phase.
	if envApplied {
		restoreEnv()
		envApplied = false
	}

	// Cleanup: ensure env restored, restart scenario normally, then tear down isolation resources.
	defer func() {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		if err := env.TargetRuntime.Restore(context.Background(), logWriter); err != nil {
			shared.LogWarn(logWriter, "failed to restart scenario back to normal resources: %v", err)
		}
		if err := isoResult.Cleanup(context.Background()); err != nil {
			shared.LogWarn(logWriter, "failed to clean up playbooks isolation resources: %v", err)
		}
	}()

	report := runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, isoResult.Env)

	// If retention is enabled, surface inspect commands as observations to aid debugging.
	if retainIsolation && len(isoResult.Resources) > 0 {
		for _, res := range isoResult.Resources {
			for _, cmd := range res.InspectCommands {
				report.Observations = append(report.Observations, NewInfoObservation(fmt.Sprintf("retain %s: %s", res.Name, cmd)))
			}
		}
	}

	return report
}

func runLoadedPlaybooksPhase(
	ctx context.Context,
	env workspace.Environment,
	logWriter io.Writer,
	playbooksCfg *config.Config,
	registry playbooks.Registry,
	seedEnv map[string]string,
) RunReport {
	return RunPhase(ctx, logWriter, "playbooks",
		func() (*playbooks.RunResult, error) {
			runner := playbooks.New(playbooks.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				TestDir:      env.TestDir,
				AppRoot:      env.AppRoot,
			},
				playbooks.WithLogger(logWriter),
				playbooks.WithPlaybooksConfig(playbooksCfg),
				playbooks.WithRegistryLoader(staticRegistryLoader{registry: registry}),
				playbooks.WithSeedEnv(seedEnv),
				playbooks.WithPortResolver(func(ctx context.Context, scenarioName, portName string) (string, error) {
					return ResolveScenarioPort(ctx, logWriter, scenarioName, portName)
				}),
				playbooks.WithScenarioStarter(func(ctx context.Context, scenario string) error {
					shared.LogStep(logWriter, "ensuring scenario %s is running", scenario)
					return phaseCommandExecutor(ctx, "", logWriter, "vrooli", "scenario", "start", scenario, "--clean-stale")
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
