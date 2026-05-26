package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"data-backup-manager/internal/clock"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/modules"
	"data-backup-manager/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	destinationsH "data-backup-manager/handlers/destinations"
	healthH "data-backup-manager/handlers/health"
	plansH "data-backup-manager/handlers/plans"
	restoresH "data-backup-manager/handlers/restores"
	runsH "data-backup-manager/handlers/runs"
	targetsH "data-backup-manager/handlers/targets"

	destint "data-backup-manager/internal/destinations"
	plansint "data-backup-manager/internal/plans"
	restoresint "data-backup-manager/internal/restores"
	runsint "data-backup-manager/internal/runs"
	schedint "data-backup-manager/internal/scheduler"
	"data-backup-manager/internal/sources"
	targetsint "data-backup-manager/internal/targets"
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
		storage.Options{ScenarioID: "data-backup-manager"},
		storage.ClassData,
		"data-backup-manager.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve data-backup-manager db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "data-backup-manager"}) {
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

	// The backup engine (resource-kopia) wrapped behind the KopiaEngine seam,
	// and the storage root the manager protects (a destination must not point
	// under it — the separate-root rule). SCENARIO_DATA_DIR is set by the
	// lifecycle; empty in bare dev runs, which makes the rule permissive.
	clk := clock.System{}
	logger := log.Default()
	kopia := engine.NewKopiaCLI()
	protectedRoot := strings.TrimSpace(os.Getenv("SCENARIO_DATA_DIR"))

	// Concrete domain services used both for mounting (via each module) and as
	// the backing for the cross-domain adapters the run orchestration needs.
	targetsSvc := targetsint.NewService(targetsint.NewSQLiteRepository(db, clk))
	destSvc := destint.NewService(destint.NewSQLiteRepository(db, clk), kopia, protectedRoot)
	plansSvc := plansint.NewService(plansint.NewSQLiteRepository(db, clk))

	// The run orchestration: capture (sources) + snapshot/retention (engine),
	// cap-block via destinations, reading plans/targets through adapters.
	sourceRegistry := sources.NewProductionRegistry(sources.ExecRunner{})
	runsSvc := runsint.NewService(runsint.Deps{
		Repo:         runsint.NewSQLiteRepository(db, clk),
		Plans:        planLookup{svc: plansSvc},
		Targets:      targetLookup{svc: targetsSvc},
		Destinations: destinationLookup{svc: destSvc},
		Engine:       kopia,
		Sources:      sourceRegistry,
		Events:       logEventSink{logger: logger},
		Clock:        clk,
	})

	// Restore + verify gate: restore a target to a location, or test-restore to
	// scratch + checksum (verify), recording last-verified (OT-P0-006).
	restoresSvc := restoresint.NewService(restoresint.Deps{
		Repo:         restoresint.NewSQLiteRepository(db, clk),
		Targets:      restoreTargetLookup{svc: targetsSvc},
		Destinations: restoreDestinationLookup{svc: destSvc},
		Engine:       kopia,
		Sources:      sourceRegistry,
		Clock:        clk,
	})

	// In-process scheduler: fires due plans on their cadence (OT-P0-005).
	scheduler := schedint.New(clk, planSource{svc: plansSvc}, runTrigger{svc: runsSvc})
	startScheduler(context.Background(), scheduler, logger)

	// Health reports backup posture (degraded when targets are overdue/failed)
	// in addition to liveness, and emits a posture event for monitoring.
	posture := backupPosture{runs: runsSvc, clock: clk, overdueAfter: overdueAfter()}

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.ModuleWithPosture(db, "data-backup-manager-api", "1.0.0", posture, logEventSink{logger: logger}),
		destinationsH.Module(db, clk, kopia, protectedRoot, logger),
		plansH.Module(db, clk, logger),
		restoresH.Module(restoresSvc, logger),
		runsH.Module(runsSvc, logger),
		targetsH.Module(db, clk, logger),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
