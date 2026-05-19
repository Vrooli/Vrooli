package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"development-toolchain-validator/internal/clock"
	"development-toolchain-validator/internal/modules"
	"development-toolchain-validator/internal/server"

	"github.com/vrooli/api-core/database"
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

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	skillCatalogSource := promptmanager.NewSkillCatalogRESTAdapter(promptmanager.Options{})

	// validation_run worker. Constructed before the server module so the
	// module's Service.Start can use the worker's Notify hook to wake
	// the loop immediately on new queued runs.
	vrunRepo := vrun.NewSQLiteRepository(db)
	vrRepo := vr.NewSQLiteRepository(db)
	vrService := vr.NewService(vrRepo, clock.System{})
	worker := vrun.NewWorker(vrun.WorkerDeps{
		Repo:      vrunRepo,
		Records:   vrService,
		AgentMgr:  agentmanager.New(agentmanager.Options{}),
		Tools:     devtools.New(devtools.Options{Clock: clock.System{}}),
		Sandbox:   workspacesandbox.New(workspacesandbox.Options{}),
		Goldens:   vrun.GoldenSourceFromRepo{Repo: golden.NewSQLiteRepository(db, clock.System{})},
		Manifests: vrun.ManifestSourceFromRepo{Repo: manifest.NewSQLiteRepository(db, clock.System{})},
		Clock:     clock.System{},
		Logger:    log.Default(),
	}, vrun.WorkerConfig{})
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go worker.Run(workerCtx)

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "development-toolchain-validator-api", "1.0.0"),
		goldenH.Module(db, clock.System{}, log.Default()),
		manifestH.Module(db, clock.System{}, log.Default()),
		reportH.Module(db, clock.System{}, skillCatalogSource, log.Default()),
		skillCatalogH.Module(db, clock.System{}, skillCatalogSource, log.Default()),
		stalenessH.Module(db, clock.System{}, log.Default()),
		validationRecordH.Module(db, clock.System{}, log.Default()),
		validationRunH.Module(validationRunH.ModuleDeps{
			DB: db, Clock: clock.System{}, Logger: log.Default(),
			Notify: worker.Notify,
		}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error {
			workerCancel()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
