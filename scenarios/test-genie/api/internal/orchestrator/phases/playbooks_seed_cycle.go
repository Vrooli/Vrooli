package phases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
	"test-genie/internal/storage/sqlfiles"

	"github.com/vrooli/api-core/database"
	// Register modernc.org/sqlite as the pure-Go "sqlite" driver.
	_ "modernc.org/sqlite"

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

type resourceNeeds struct {
	RequirePostgres bool
	RequireRedis    bool
	RequireSQLite   bool
	SQLiteEnvVars   []string
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

	needs := detectResourceNeeds(env, logWriter)
	isoManager := isolationManagerFactory(isolation.Config{
		ScenarioName:    env.ScenarioName,
		RequirePostgres: needs.RequirePostgres,
		RequireRedis:    needs.RequireRedis,
		RequireSQLite:   needs.RequireSQLite,
		SQLiteEnvVars:   needs.SQLiteEnvVars,
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

	if err := applyPlaybooksMigrations(ctx, env, needs, logWriter); err != nil {
		if envApplied {
			restoreEnv()
		}
		_ = isoResult.Cleanup(context.Background())
		return nil, fmt.Errorf("failed to apply playbooks migrations: %w", err)
	}

	if err := RestartScenario(ctx, env.ScenarioName, logWriter); err != nil {
		if envApplied {
			restoreEnv()
		}
		_ = isoResult.Cleanup(context.Background())
		return nil, fmt.Errorf("failed to restart scenario with playbooks isolation: %w", err)
	}

	if envApplied {
		restoreEnv()
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
// against the isolated database backend. Files execute in lexicographic order.
func applyPlaybooksMigrations(ctx context.Context, env workspace.Environment, needs resourceNeeds, logWriter io.Writer) error {
	if !needs.RequirePostgres && !needs.RequireSQLite {
		return nil
	}

	migrationsDir := filepath.Join(env.ScenarioDir, "bas", "seeds", "migrations")
	commonFiles, err := collectMigrationFiles(migrationsDir, "common")
	if err != nil {
		return err
	}
	postgresFiles, err := collectScopedMigrationFiles(migrationsDir, "postgres")
	if err != nil {
		return err
	}
	sqliteFiles, err := collectScopedMigrationFiles(migrationsDir, "sqlite")
	if err != nil {
		return err
	}
	if len(commonFiles) == 0 && len(postgresFiles) == 0 && len(sqliteFiles) == 0 {
		return nil
	}

	if needs.RequirePostgres {
		files := append([]string(nil), commonFiles...)
		files = append(files, postgresFiles...)
		if err := applyPostgresMigrations(ctx, env, files, logWriter); err != nil {
			return err
		}
	}
	if needs.RequireSQLite {
		files := append([]string(nil), commonFiles...)
		files = append(files, sqliteFiles...)
		if err := applySQLiteMigrations(ctx, env, files, logWriter); err != nil {
			return err
		}
	}
	return nil
}

func applyPostgresMigrations(ctx context.Context, env workspace.Environment, files []string, logWriter io.Writer) error {
	if err := EnsureCommandAvailable("psql"); err != nil {
		return fmt.Errorf("psql not available for playbooks migrations: %w", err)
	}
	connURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if connURL == "" {
		return fmt.Errorf("DATABASE_URL is not set for playbooks migrations")
	}
	shared.LogStep(logWriter, "applying playbooks migrations (%d file(s))", len(files))
	for _, file := range files {
		shared.LogInfo(logWriter, "  psql -f %s", file)
		if err := phaseCommandExecutor(ctx, env.ScenarioDir, logWriter, "psql", "-d", connURL, "-v", "ON_ERROR_STOP=1", "-f", file); err != nil {
			return fmt.Errorf("psql apply %s: %w", file, err)
		}
	}
	return nil
}

func applySQLiteMigrations(ctx context.Context, env workspace.Environment, files []string, logWriter io.Writer) error {
	sqliteDSN := strings.TrimSpace(firstNonEmpty(
		os.Getenv("PLAYBOOKS_SQLITE_DSN"),
		os.Getenv("PLAYBOOKS_SQLITE_PATH"),
		os.Getenv("SQLITE_PATH"),
		os.Getenv("SQLITE_DB"),
	))
	if sqliteDSN == "" {
		return fmt.Errorf("SQLITE_PATH is not set for playbooks migrations")
	}
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          sqliteDSN,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return fmt.Errorf("open sqlite migrations database: %w", err)
	}
	defer db.Close()

	shared.LogStep(logWriter, "applying playbooks sqlite migrations (%d file(s))", len(files))
	for _, file := range files {
		shared.LogInfo(logWriter, "  sqlite migrate %s", file)
		if err := sqlfiles.ExecFile(db, file); err != nil {
			return fmt.Errorf("sqlite apply %s: %w", file, err)
		}
	}
	return nil
}

func collectMigrationFiles(root, scopedDir string) ([]string, error) {
	legacyFiles, err := collectScopedMigrationFiles(root, "")
	if err != nil {
		return nil, err
	}
	scopedFiles, err := collectScopedMigrationFiles(root, scopedDir)
	if err != nil {
		return nil, err
	}
	merged := append([]string(nil), legacyFiles...)
	merged = append(merged, scopedFiles...)
	return merged, nil
}

func collectScopedMigrationFiles(root, scopedDir string) ([]string, error) {
	targetDir := root
	if scopedDir != "" {
		targetDir = filepath.Join(root, scopedDir)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read migrations dir %s: %w", targetDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, filepath.Join(targetDir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var sqliteSignalTokens = []string{
	"modernc.org/sqlite",
	"github.com/mattn/go-sqlite3",
	`sql.Open("sqlite"`,
	`sqlx.Connect("sqlite"`,
	"DriverSQLite",
	"sqliteDSN(",
	"BAS_SQLITE_PATH",
	"SQLITE_PATH",
}

// detectResourceNeeds inspects stable scenario signals and returns which
// isolated resources should be provisioned for Playbooks. SQLite is inferred
// from scenario files/code instead of resource declarations because SQLite is
// now a local runtime detail rather than a first-class Vrooli resource.
func detectResourceNeeds(env workspace.Environment, logWriter io.Writer) resourceNeeds {
	needs := resourceNeeds{}

	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	if manifest, err := workspace.LoadServiceManifest(manifestPath); err == nil {
		needs.SQLiteEnvVars = manifest.SQLitePathEnvVars()
		for _, res := range manifest.Dependencies.Resources {
			if !res.Enabled && !res.Required {
				continue
			}
			switch strings.ToLower(res.Type) {
			case "postgres":
				needs.RequirePostgres = true
			case "redis":
				needs.RequireRedis = true
			}
		}
	} else {
		shared.LogWarn(logWriter, "unable to read service manifest (%v); sqlite detection will rely on scenario files", err)
	}

	sqliteSignals, err := detectSQLiteSignals(env.ScenarioDir)
	if err != nil {
		shared.LogWarn(logWriter, "unable to inspect sqlite signals (%v)", err)
	} else if len(sqliteSignals) > 0 {
		needs.RequireSQLite = true
		shared.LogInfo(logWriter, "sqlite detected via scenario signals: %s", strings.Join(sqliteSignals, ", "))
	}

	// If nothing matched, assume both legacy backing services to avoid false negatives.
	if !needs.RequirePostgres && !needs.RequireRedis && !needs.RequireSQLite {
		shared.LogWarn(logWriter, "scenario declares no postgres/redis resources and emitted no sqlite signals; defaulting to provision Postgres + Redis for playbooks isolation")
		return resourceNeeds{RequirePostgres: true, RequireRedis: true}
	}

	return needs
}

func detectSQLiteSignals(scenarioDir string) ([]string, error) {
	var signals []string

	schemaPath := filepath.Join(scenarioDir, "initialization", "storage", "sqlite", "schema.sql")
	if info, err := os.Stat(schemaPath); err == nil && !info.IsDir() {
		signals = append(signals, "initialization/storage/sqlite/schema.sql")
	}

	err := filepath.WalkDir(scenarioDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "coverage", "dist", "data", ".next":
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)
		if base != "go.mod" && filepath.Ext(path) != ".go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, token := range sqliteSignalTokens {
			if strings.Contains(content, token) {
				rel, relErr := filepath.Rel(scenarioDir, path)
				if relErr != nil {
					rel = path
				}
				signals = append(signals, rel)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(signals)
	return compactStrings(signals), nil
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	var prev string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == prev {
			continue
		}
		out = append(out, value)
		prev = value
	}
	return out
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
