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
	"test-genie/internal/playbooks/dbdetect"
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
	// PrimaryDriver identifies the scenario's primary database driver
	// ("postgres" or "sqlite") when detection has strong evidence
	// (e.g. a Go driver import). Empty when no single driver dominates.
	// Used by the routed path to pick a DSN that matches the scenario's
	// actual driver, avoiding hangs from cross-driver DSN injection.
	PrimaryDriver string
	SQLiteEnvVars []string
}

// isolationProvider lets tests stub seed isolation without requiring Docker.
type isolationProvider interface {
	Prepare(ctx context.Context) (*isolation.Result, error)
}

var isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
	return isolation.NewManager(cfg)
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

	needs := resolveDBNeeds(ctx, env, logWriter)
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

	if env.TargetRuntime == nil {
		if envApplied {
			restoreEnv()
		}
		_ = isoResult.Cleanup(context.Background())
		return nil, fmt.Errorf("target runtime manager is not configured")
	}

	if err := env.TargetRuntime.RestartWithEnv(ctx, isoResult.Env, logWriter); err != nil {
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
		if err := env.TargetRuntime.Restore(cleanupCtx, logWriter); err != nil {
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

// resolveDBNeeds runs the dbdetect resolver against the
// scenario, writes the evidence chain to logWriter, and projects the report
// to the booleans the isolation manager consumes. There is no fallback: a
// scenario with no evidence is provisioned with nothing, which is the
// correct signal that detection should be fixed at the source.
func resolveDBNeeds(ctx context.Context, env workspace.Environment, logWriter io.Writer) resourceNeeds {
	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	rawManifest, err := workspace.LoadServiceManifest(manifestPath)
	if err != nil {
		shared.LogWarn(logWriter, "unable to read service manifest (%v); proceeding with file-based detection only", err)
	}
	manifest := dbdetect.WrapManifest(rawManifest)

	resolver, err := dbdetect.NewResolver(dbdetect.DefaultCollectors(), dbdetect.DefaultProfiles())
	if err != nil {
		shared.LogWarn(logWriter, "db-detect resolver construction failed: %v", err)
		return resourceNeeds{SQLiteEnvVars: manifest.SQLitePathEnvVars()}
	}

	report := resolver.Resolve(ctx, dbdetect.ScenarioInputs{
		ScenarioDir: env.ScenarioDir,
		Manifest:    manifest,
		Filesystem:  dbdetect.OSFilesystem{},
	})
	if logWriter != nil {
		_, _ = logWriter.Write([]byte(report.FormatHuman()))
	}
	needs := resourceNeeds{
		RequirePostgres: report.Required("postgres"),
		RequireRedis:    report.Required("redis"),
		RequireSQLite:   report.Required("sqlite"),
		PrimaryDriver:   primaryDriver(report),
		SQLiteEnvVars:   manifest.SQLitePathEnvVars(),
	}
	if needs.PrimaryDriver == "" {
		shared.LogWarn(logWriter, "db-detect did not pick a primary driver — routed path will not be used")
	} else {
		shared.LogInfo(logWriter, "db-detect primary driver = %s", needs.PrimaryDriver)
	}
	return needs
}

// primaryDriver returns "postgres" or "sqlite" when one has strictly
// stronger evidence than the other (per Evidence.Priority), and empty
// otherwise. The empty case means "no winner — fall back to the caller's
// default DSN order" (caller-defined behavior).
func primaryDriver(report dbdetect.DetectionReport) string {
	pgPriority := decisionPriority(report, "postgres")
	sqlitePriority := decisionPriority(report, "sqlite")
	if pgPriority > sqlitePriority {
		return "postgres"
	}
	if sqlitePriority > pgPriority {
		return "sqlite"
	}
	return ""
}

func decisionPriority(report dbdetect.DetectionReport, db string) dbdetect.Priority {
	res, ok := report.Results[db]
	if !ok || !res.Required || res.Decision == nil {
		return 0
	}
	return res.Decision.Priority
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
