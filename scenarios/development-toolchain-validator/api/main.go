package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"development-toolchain-validator/internal/modules"
	"development-toolchain-validator/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	goldenH "development-toolchain-validator/handlers/golden"
	healthH "development-toolchain-validator/handlers/health"
	manifestH "development-toolchain-validator/handlers/manifest"
	reportH "development-toolchain-validator/handlers/report"
	skillCatalogH "development-toolchain-validator/handlers/skill_catalog"
	stalenessH "development-toolchain-validator/handlers/staleness"
	validationRecordH "development-toolchain-validator/handlers/validation_record"
	validationRunH "development-toolchain-validator/handlers/validation_run"

	agentmanager "development-toolchain-validator/integrations/agent_manager"
	devtools "development-toolchain-validator/integrations/dev_tools"
	promptmanager "development-toolchain-validator/integrations/prompt_manager"
	workspacesandbox "development-toolchain-validator/integrations/workspace_sandbox"

	golden "development-toolchain-validator/internal/golden"
	manifest "development-toolchain-validator/internal/manifest"
	vr "development-toolchain-validator/internal/validation_record"
	vrun "development-toolchain-validator/internal/validation_run"
)

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "development-toolchain-validator"},
		storage.ClassData,
		"development-toolchain-validator.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve development-toolchain-validator db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "development-toolchain-validator"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// Additive column migrations for tables that gained columns after their
	// initial schema shipped (sqlite has no ADD COLUMN IF NOT EXISTS, so
	// these are introspection-guarded and idempotent).
	if err := vr.EnsureColumns(context.Background(), db); err != nil {
		log.Fatalf("validation_record migration failed: %v", err)
	}
	if err := golden.EnsureColumns(context.Background(), db); err != nil {
		log.Fatalf("golden migration failed: %v", err)
	}

	skillCatalogSource := promptmanager.NewSkillCatalogRESTAdapter(promptmanager.Options{})

	// agent-manager profile reconciliation. DTV declares its sandboxed
	// runner profile in .vrooli/agent-profiles/default.json and lists
	// the source in service.json; this call asks agent-manager to upsert
	// the profile keyed on "development-toolchain-validator/default".
	// Failure here is non-fatal because the validation_run worker fails
	// individual skill runs visibly (verdict=run_failure) when the
	// profile is missing — preferable to refusing to boot the whole
	// API. A 10s ceiling keeps a misconfigured/slow agent-manager from
	// stalling startup indefinitely.
	agentClient := agentmanager.New(agentmanager.Options{})
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if resp, err := agentClient.Initialize(initCtx); err != nil {
		log.Printf("agent-manager profile reconciliation failed (skill validation will fail until resolved): %v", err)
	} else {
		log.Printf("agent-manager profiles reconciled: scenario=%s created=%d updated=%d unchanged=%d failed=%d",
			resp.Scenario, resp.Created, resp.Updated, resp.Unchanged, resp.Failed)
	}
	initCancel()

	// validation_run worker. Constructed before the server module so the
	// module's Service.Start can use the worker's Notify hook to wake
	// the loop immediately on new queued runs.
	vrunRepo := vrun.NewSQLiteRepository(db)
	vrRepo := vr.NewSQLiteRepository(db)
	vrService := vr.NewService(vrRepo, schedule.System())
	worker := vrun.NewWorker(vrun.WorkerDeps{
		Repo:     vrunRepo,
		Records:  vrService,
		AgentMgr: agentClient,
		Tools:    devtools.New(devtools.Options{Clock: schedule.System()}),
		Sandbox:  workspacesandbox.New(workspacesandbox.Options{}),
		Goldens: vrun.GoldenMaterializerFromRepo{
			Repo:   golden.NewSQLiteRepository(db, schedule.System()),
			Runner: golden.NewSubprocessRunner("vrooli", ""),
		},
		Manifests: vrun.ManifestSourceFromRepo{Repo: manifest.NewSQLiteRepository(db, schedule.System())},
		Skills:    promptmanager.NewSkillContentRESTAdapter(promptmanager.Options{}),
		Clock:     schedule.System(),
		Logger:    log.Default(),
	}, vrun.WorkerConfig{})
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go worker.Run(workerCtx)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "development-toolchain-validator-api", "1.0.0"),
		goldenH.Module(db, schedule.System(), log.Default()),
		manifestH.Module(db, schedule.System(), log.Default()),
		reportH.Module(db, schedule.System(), skillCatalogSource, log.Default()),
		skillCatalogH.Module(db, schedule.System(), skillCatalogSource, log.Default()),
		stalenessH.Module(db, schedule.System(), log.Default()),
		validationRecordH.Module(db, schedule.System(), log.Default()),
		validationRunH.Module(validationRunH.ModuleDeps{
			DB: db, Clock: schedule.System(), Logger: log.Default(),
			Notify: worker.Notify,
		}),
	)

	// Top-level mux mounts the API plus, in development mode, the dev-only
	// RoutingService used by test-genie to install a runtime test DB pool
	// without restarting this scenario. devrouting.Register is a no-op in
	// production mode.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// context so *database.RoutedDB routes the call to the installed test
	// pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			workerCancel()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
