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

	coverageH "data-backup-manager/handlers/coverage"
	destinationsH "data-backup-manager/handlers/destinations"
	discoveryH "data-backup-manager/handlers/discovery"
	healthH "data-backup-manager/handlers/health"
	plansH "data-backup-manager/handlers/plans"
	restoresH "data-backup-manager/handlers/restores"
	runsH "data-backup-manager/handlers/runs"
	targetsH "data-backup-manager/handlers/targets"

	coverageint "data-backup-manager/internal/coverage"
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

	// Additive column migration for the runs table (error, updated_at) so the
	// async-execution columns land on a database that predates them without
	// losing run history.
	if err := runsint.EnsureColumns(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return fmt.Errorf("runs column migration failed: %w", err)
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

	// Discovery: read-only onboarding suggestions. Scans well-known runtime
	// state (~/.vrooli) for targets and mounted volumes for destinations,
	// filtering against the live catalog + a dismissals table. Suggestions are
	// derived; only dismissals persist. The protected-path set is computed here
	// (runtime root + registered destinations + registered target locators) —
	// wider than the destinations service's own protectedRoot (D4). Built before
	// plans because the plan coverage guard reads its recommendations.
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

	// Plan coverage guard: a suggestions-only coverage instance backs the plans
	// service so create/update can block on incomplete non-sensitive default
	// coverage. It depends only on discovery (not plans), which keeps the
	// construction graph acyclic — the full coverage service below reads plans.
	guardCoverage := coverageint.NewService(coverageint.Deps{
		Suggestions: coverageSuggestions{svc: discoverySvc},
	})
	plansSvc := plansint.NewService(plansint.NewSQLiteRepository(db, clk), planCoverageGuard{svc: guardCoverage})

	// The run orchestration: capture (sources) + snapshot/retention (engine),
	// cap-block via destinations, reading plans/targets through adapters. Runs
	// execute on a background worker bound to execCtx (server lifetime), so a
	// client/proxy disconnect cannot cancel an in-flight backup. execCtx is
	// cancelled on graceful shutdown to interrupt and drain in-flight work.
	execCtx, cancelExec := context.WithCancel(ctx)
	defer cancelExec()
	concurrency, err := runConcurrency()
	if err != nil {
		cancelExec()
		_ = db.Close()
		return err
	}
	overdueAfterDur, err := overdueAfter()
	if err != nil {
		cancelExec()
		_ = db.Close()
		return err
	}
	sourceRegistry := sources.NewProductionRegistry(sources.ExecRunner{})
	runsSvc := runsint.NewService(runsint.Deps{
		Repo:              runsint.NewSQLiteRepository(db, clk),
		Plans:             planLookup{svc: plansSvc},
		Targets:           targetLookup{svc: targetsSvc},
		ActiveTargets:     targetLookup{svc: targetsSvc},
		Destinations:      destinationLookup{svc: destSvc},
		Engine:            kopia,
		Sources:           sourceRegistry,
		Events:            logEventSink{logger: logger},
		Clock:             clk,
		Logger:            logger,
		BaseContext:       execCtx,
		TargetConcurrency: concurrency,
		OverdueAfter:      overdueAfterDur,
	})

	// Startup reconciliation: close any run left non-terminal by a prior
	// crash/restart/disconnect so no run can wedge in pending/capturing forever.
	if err := runsSvc.Reconcile(ctx); err != nil {
		cancelExec()
		_ = db.Close()
		return fmt.Errorf("run reconciliation failed: %w", err)
	}

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

	// Coverage: the first-real-backup readiness surface. Composes discovery
	// suggestions with the targets/plans/runs/restores catalogs into one report
	// plus bulk default acceptance. Owns no scanner logic; reads no file
	// contents; persists nothing.
	coverageSvc := coverageint.NewService(coverageint.Deps{
		Suggestions: coverageSuggestions{svc: discoverySvc},
		Targets:     coverageTargetCatalog{svc: targetsSvc},
		Plans:       coveragePlanCatalog{svc: plansSvc},
		Runs:        coverageRunStatus{svc: runsSvc},
		Restores:    coverageVerified{svc: restoresSvc},
	})

	// In-process scheduler: fires due plans on their cadence (OT-P0-005).
	scheduler := schedint.New(clk, planSource{svc: plansSvc}, runTrigger{svc: runsSvc})
	if err := startScheduler(ctx, scheduler, logger); err != nil {
		_ = db.Close()
		return err
	}

	// Health reports backup posture (degraded when targets are overdue/failed)
	// in addition to liveness, and emits a posture event for monitoring. It
	// reads the same per-target Overdue the runs service computes from
	// overdueAfterDur, so /health and `runs status` never disagree.
	posture := backupPosture{runs: runsSvc}

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.ModuleWithPosture(db, "data-backup-manager-api", "1.0.0", posture, logEventSink{logger: logger}),
		coverageH.Module(coverageSvc, logger),
		destinationsH.Module(db, clk, kopia, protectedRoot, logger),
		discoveryH.Module(discoverySvc, logger),
		plansH.Module(plansSvc, logger),
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
		// Backup runs are now async (TriggerRun returns immediately), so they no
		// longer need a long write timeout. But restores verify/restore are still
		// synchronous and exec kopia for the whole restore+checksum — minutes on
		// large targets / slow external drives. The api-core default WriteTimeout
		// (30s) severs those mid-operation ("unexpected EOF" client-side) even
		// though the handler keeps running. A generous ceiling lets them return
		// cleanly; the CLI pairs it with an unlimited-timeout client for the same
		// RPCs. (ReadTimeout stays default — request bodies here are tiny.) When
		// restores also become async this can drop to the api-core default.
		WriteTimeout: 6 * time.Hour,
		Cleanup: func(cctx context.Context) error {
			// Interrupt and drain in-flight runs before closing the DB so a
			// worker cannot write to a closed handle.
			cancelExec()
			_ = runsSvc.Shutdown(cctx)
			return db.Close()
		},
	}); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
