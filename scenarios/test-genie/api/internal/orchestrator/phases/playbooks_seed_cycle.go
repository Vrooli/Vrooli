package phases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks/config"
	"test-genie/internal/playbooks/isolation"
	"test-genie/internal/playbooks/seeds"
	"test-genie/internal/shared"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// PlaybooksSeedSession holds state for a seed lifecycle run.
type PlaybooksSeedSession struct {
	RunID      string
	Env        map[string]string
	Resources  []isolation.ResourceInfo
	SeedState  map[string]any
	CleanupRef string
	cleanup    func(ctx context.Context) error
}

// Cleanup tears down isolation resources and restarts the scenario to normal resources.
func (s *PlaybooksSeedSession) Cleanup(ctx context.Context) error {
	if s == nil || s.cleanup == nil {
		return nil
	}
	return s.cleanup(ctx)
}

// ApplyPlaybooksSeed provisions isolated resources, restarts the scenario, runs seed scripts,
// and returns seed state for BAS workflow execution.
func ApplyPlaybooksSeed(ctx context.Context, env workspace.Environment, logWriter io.Writer, retain bool) (*PlaybooksSeedSession, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	playbooksCfg, err := config.Load(env.ScenarioDir)
	if err != nil {
		playbooksCfg = config.Default()
	}
	if playbooksCfg != nil && !playbooksCfg.Seeds.Enabled {
		return nil, fmt.Errorf("playbooks seeds disabled via .vrooli/testing.json")
	}

	requirePG, requireRedis := detectResourceNeeds(env, logWriter)
	isoManager := isolationManagerFactory(isolation.Config{
		ScenarioName:    env.ScenarioName,
		RequirePostgres: requirePG,
		RequireRedis:    requireRedis,
		Retain:          retain,
		LogWriter:       logWriter,
		Timeout:         2 * time.Minute,
	})

	isoResult, err := isoManager.Prepare(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare playbooks isolation: %w", err)
	}

	restoreEnv := isolation.ApplyEnv(isoResult.Env)
	envApplied := true

	if err := applyPlaybooksMigrations(ctx, env, requirePG, logWriter); err != nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return nil, fmt.Errorf("failed to apply playbooks migrations: %w", err)
	}

	if err := RestartScenario(ctx, env.ScenarioName, logWriter); err != nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return nil, fmt.Errorf("failed to restart scenario with playbooks isolation: %w", err)
	}

	if envApplied {
		restoreEnv()
		envApplied = false
	}

	seedCtx, cancel := context.WithTimeout(ctx, playbooksCfg.Seeds.SeedTimeout())
	defer cancel()

	seedManager := seeds.NewManager(env.ScenarioDir, env.AppRoot, env.TestDir, logWriter)
	restoreSeedEnv := applyEnv(isoResult.Env)
	_, seedErr := seedManager.Apply(seedCtx)
	restoreSeedEnv()
	if seedErr != nil {
		_ = isoResult.Cleanup(context.Background())
		return nil, fmt.Errorf("seed execution failed: %w", seedErr)
	}

	seedState, err := loadSeedState(env.ScenarioDir)
	if err != nil {
		_ = isoResult.Cleanup(context.Background())
		return nil, err
	}

	session := &PlaybooksSeedSession{
		RunID:     isoResult.RunID,
		Env:       isoResult.Env,
		Resources: isoResult.Resources,
		SeedState: seedState,
	}
	session.cleanup = func(cleanupCtx context.Context) error {
		if err := RestartScenario(cleanupCtx, env.ScenarioName, logWriter); err != nil {
			shared.LogWarn(logWriter, "failed to restart scenario back to normal resources: %v", err)
		}
		if err := isoResult.Cleanup(cleanupCtx); err != nil {
			return fmt.Errorf("failed to clean up playbooks isolation resources: %w", err)
		}
		return nil
	}

	return session, nil
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

func applyEnv(env map[string]string) func() {
	if len(env) == 0 {
		return func() {}
	}
	prev := make(map[string]*string, len(env))
	for k, v := range env {
		if existing, ok := os.LookupEnv(k); ok {
			val := existing
			prev[k] = &val
		} else {
			prev[k] = nil
		}
		_ = os.Setenv(k, v)
	}
	return func() {
		for k, v := range prev {
			if v == nil {
				_ = os.Unsetenv(k)
				continue
			}
			_ = os.Setenv(k, *v)
		}
	}
}

func loadSeedState(scenarioDir string) (map[string]any, error) {
	seedPath := sharedartifacts.SeedStatePath(scenarioDir)
	data, err := os.ReadFile(seedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse seed state JSON: %w", err)
	}
	return state, nil
}
