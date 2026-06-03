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
	discoveryH "data-backup-manager/handlers/discovery"
	healthH "data-backup-manager/handlers/health"
	plansH "data-backup-manager/handlers/plans"
	restoresH "data-backup-manager/handlers/restores"
	runsH "data-backup-manager/handlers/runs"
	targetsH "data-backup-manager/handlers/targets"

	destint "data-backup-manager/internal/destinations"
	discoveryint "data-backup-manager/internal/discovery"
	plansint "data-backup-manager/internal/plans"
	restoresint "data-backup-manager/internal/restores"
	runsint "data-backup-manager/internal/runs"
	schedint "data-backup-manager/internal/scheduler"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"
	targetsint "data-backup-manager/internal/targets"
)

func lookupEnvTrimmed(name string) (string, bool) {
	value, ok := os.LookupEnv(name)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

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
	if path, ok := lookupEnvTrimmed("SQLITE_PATH"); ok {
		return sqliteFileDSN(path)
	}
	if path, ok := lookupEnvTrimmed("SQLITE_DB"); ok {
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
	if err := run(context.Background()); err != nil {
		log.Fatalf("%v", err)
	}
}

func healthHandler(handler http.Handler) http.HandlerFunc {
	return handler.ServeHTTP
}

func run(ctx context.Context) error {
	dsn, err := sqliteDSN()
	if err != nil {
		return fmt.Errorf("sqlite configuration failed: %w", err)
	}

	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		_ = db.Close()
		return fmt.Errorf("schema initialization failed: %w", err)
	}

	// Additive column migration for the destinations table (SQLite has no
	// ADD COLUMN IF NOT EXISTS); idempotent on fresh and existing databases.
	if err := destint.EnsureColumns(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return fmt.Errorf("destinations column migration failed: %w", err)
	}

	// The backup engine (resource-kopia) wrapped behind the KopiaEngine seam,
	// and the storage root the manager protects (a destination must not point
	// under it — the separate-root rule). SCENARIO_DATA_DIR is set by the
	// lifecycle; empty in bare dev runs, which makes the rule permissive.
	clk := clock.System{}
	logger := log.Default()
	kopia := engine.NewKopiaCLI()
	protectedRoot, _ := lookupEnvTrimmed("SCENARIO_DATA_DIR")

	// Concrete domain services used both for mounting (via each module) and as
	// the backing for the cross-domain adapters the run orchestration needs.
	targetsSvc := targetsint.NewService(targetsint.NewSQLiteRepository(db, clk))
	destSvc := destint.NewService(destint.NewSQLiteRepository(db, clk), kopia, &destint.FSBundleWriter{}, protectedRoot)
	plansSvc := plansint.NewService(plansint.NewSQLiteRepository(db, clk))

	// The run orchestration: capture (sources) + snapshot/retention (engine),
	// cap-block via destinations, reading plans/targets through adapters.
	sourceRegistry := sources.NewProductionRegistry(sources.ExecRunner{})
	runsSvc := runsint.NewService(runsint.Deps{
		Repo:          runsint.NewSQLiteRepository(db, clk),
		Plans:         planLookup{svc: plansSvc},
		Targets:       targetLookup{svc: targetsSvc},
		ActiveTargets: targetLookup{svc: targetsSvc},
		Destinations:  destinationLookup{svc: destSvc},
		Engine:        kopia,
		Sources:       sourceRegistry,
		Events:        logEventSink{logger: logger},
		Clock:         clk,
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

	// Discovery: read-only onboarding suggestions. Scans well-known runtime
	// state (~/.vrooli) for targets and mounted volumes for destinations,
	// filtering against the live catalog + a dismissals table. Suggestions are
	// derived; only dismissals persist. The protected-path set is computed here
	// (runtime root + registered destinations + registered target locators) —
	// wider than the destinations service's own protectedRoot (D4).
	discoverySvc := discoveryint.NewService(discoveryint.Deps{
		Volumes: sysmounts.New(),
		// Two source scanners behind one seam: Vrooli's own runtime home
		// (~/.vrooli) plus each external-cli resource's declared durable host
		// state (coding-agent conversation history, via `vrooli resource list`).
		Sources: discoveryint.NewCompositeScanner(
			discoveryint.NewWellKnownScanner(),
			discoveryint.NewResourceDataScanner(discoveryint.NewResourceEnumerator()),
		),
		Targets:      discoveryTargetCatalog{svc: targetsSvc},
		Destinations: discoveryDestCatalog{svc: destSvc},
		Protected:    discoveryProtectedPaths{runtimeRoot: discoveryint.RuntimeRoot(), targets: targetsSvc, dests: destSvc},
		Dismissals:   discoveryint.NewSQLiteDismissalStore(db, clk),
	})

	// In-process scheduler: fires due plans on their cadence (OT-P0-005).
	scheduler := schedint.New(clk, planSource{svc: plansSvc}, runTrigger{svc: runsSvc})
	if err := startScheduler(ctx, scheduler, logger); err != nil {
		_ = db.Close()
		return err
	}

	// Health reports backup posture (degraded when targets are overdue/failed)
	// in addition to liveness, and emits a posture event for monitoring.
	overdue, err := overdueAfter()
	if err != nil {
		_ = db.Close()
		return err
	}
	posture := backupPosture{runs: runsSvc, clock: clk, overdueAfter: overdue}

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.ModuleWithPosture(db, "data-backup-manager-api", "1.0.0", posture, logEventSink{logger: logger}),
		destinationsH.Module(db, clk, kopia, protectedRoot, logger),
		discoveryH.Module(discoverySvc, logger),
		plansH.Module(db, clk, logger),
		restoresH.Module(restoresSvc, logger),
		runsH.Module(runsSvc, verifiedLookup{svc: restoresSvc}, logger),
		targetsH.Module(db, clk, logger),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	rootMux.HandleFunc("/health", healthHandler(srv.Handler()))
	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
