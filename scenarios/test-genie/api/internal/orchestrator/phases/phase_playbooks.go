package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks"
	"test-genie/internal/playbooks/config"
	"test-genie/internal/playbooks/isolation"
	"test-genie/internal/shared"
)

// isolationProvider lets tests stub isolation without requiring Docker.
type isolationProvider interface {
	Prepare(ctx context.Context) (*isolation.Result, error)
}

// isolationManagerFactory creates the default isolation manager.
var isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
	return isolation.NewManager(cfg)
}

// runPlaybooksPhase executes BAS playbook workflows using the playbooks package.
// This includes loading the registry, executing workflows via BAS API, and
// writing results for requirements coverage tracking.
func runPlaybooksPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
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

	// Determine which resources are actually needed from service manifest.
	requirePG, requireRedis := detectResourceNeeds(env, logWriter)

	// Provision isolated resources for the playbooks run (Postgres + Redis if required).
	isoManager := isolationManagerFactory(isolation.Config{
		ScenarioName:    env.ScenarioName,
		RequirePostgres: requirePG,
		RequireRedis:    requireRedis,
		Retain:          retainIsolation,
		LogWriter:       logWriter,
		Timeout:         2 * time.Minute,
	})

	isoResult, err := isoManager.Prepare(ctx)
	if err != nil {
		return RunReport{
			Err:                   fmt.Errorf("failed to prepare playbooks isolation: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure Docker is available for testcontainers or provide access to start temporary Postgres/Redis instances.",
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
	if err := applyPlaybooksMigrations(ctx, env, requirePG, logWriter); err != nil {
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

	// Restart the target scenario so it picks up the temporary resources.
	if err := RestartScenario(ctx, env.ScenarioName, logWriter); err != nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("failed to restart scenario with playbooks isolation: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Check lifecycle logs for restart failures and ensure the scenario can connect to the temporary Postgres/Redis instances.",
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
		if err := RestartScenario(context.Background(), env.ScenarioName, logWriter); err != nil {
			shared.LogWarn(logWriter, "failed to restart scenario back to normal resources: %v", err)
		}
		if err := isoResult.Cleanup(context.Background()); err != nil {
			shared.LogWarn(logWriter, "failed to clean up playbooks isolation resources: %v", err)
		}
	}()

	report := RunPhase(ctx, logWriter, "playbooks",
		func() (*playbooks.RunResult, error) {
			runner := playbooks.New(playbooks.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				TestDir:      env.TestDir,
				AppRoot:      env.AppRoot,
			},
				playbooks.WithLogger(logWriter),
				playbooks.WithPlaybooksConfig(playbooksCfg),
				playbooks.WithSeedEnv(isoResult.Env),
				playbooks.WithPortResolver(func(ctx context.Context, scenarioName, portName string) (string, error) {
					return ResolveScenarioPort(ctx, logWriter, scenarioName, portName)
				}),
				playbooks.WithScenarioStarter(func(ctx context.Context, scenario string) error {
					return StartScenario(ctx, scenario, logWriter)
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

// applyPlaybooksMigrations applies optional .sql files under bas/seeds/migrations
// against the current DATABASE_URL (already set via isolation env). Files execute in lexicographic order.
func applyPlaybooksMigrations(ctx context.Context, env workspace.Environment, requirePostgres bool, logWriter io.Writer) error {
	if !requirePostgres {
		return nil
	}

	migrationsDir := filepath.Join(env.ScenarioDir, "bas", "seeds", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read migrations dir: %w", err)
	}

	if err := EnsureCommandAvailable("psql"); err != nil {
		return fmt.Errorf("psql not available for playbooks migrations: %w", err)
	}

	connURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if connURL == "" {
		return fmt.Errorf("DATABASE_URL is not set for playbooks migrations")
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, filepath.Join(migrationsDir, entry.Name()))
	}
	if len(files) == 0 {
		return nil
	}
	sort.Strings(files)

	shared.LogStep(logWriter, "applying playbooks migrations (%d file(s))", len(files))
	for _, file := range files {
		shared.LogInfo(logWriter, "  psql -f %s", file)
		if err := phaseCommandExecutor(ctx, env.ScenarioDir, logWriter, "psql", "-d", connURL, "-v", "ON_ERROR_STOP=1", "-f", file); err != nil {
			return fmt.Errorf("psql apply %s: %w", file, err)
		}
	}
	return nil
}

// detectResourceNeeds inspects the scenario service manifest and returns whether
// Postgres and Redis should be provisioned for Playbooks isolation. Defaults to
// provisioning both when the manifest cannot be read or does not declare resources.
func detectResourceNeeds(env workspace.Environment, logWriter io.Writer) (requirePostgres bool, requireRedis bool) {
	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	manifest, err := workspace.LoadServiceManifest(manifestPath)
	if err != nil {
		shared.LogWarn(logWriter, "unable to read service manifest (%v); defaulting to Postgres + Redis isolation", err)
		return true, true
	}

	if manifest.Dependencies.Resources == nil || len(manifest.Dependencies.Resources) == 0 {
		return true, true
	}

	for _, res := range manifest.Dependencies.Resources {
		if !res.Enabled && !res.Required {
			continue
		}
		switch strings.ToLower(res.Type) {
		case "postgres":
			requirePostgres = true
		case "redis":
			requireRedis = true
		}
	}

	// If nothing matched, assume both to avoid false negatives.
	if !requirePostgres && !requireRedis {
		shared.LogWarn(logWriter, "service manifest declares no postgres/redis resources; defaulting to provision both for playbooks isolation")
		return true, true
	}

	return requirePostgres, requireRedis
}
