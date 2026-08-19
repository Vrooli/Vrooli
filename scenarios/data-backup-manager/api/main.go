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

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/modules"
	"data-backup-manager/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	auditsH "data-backup-manager/handlers/audits"
	coverageH "data-backup-manager/handlers/coverage"
	destinationsH "data-backup-manager/handlers/destinations"
	discoveryH "data-backup-manager/handlers/discovery"
	drillsH "data-backup-manager/handlers/drills"
	healthH "data-backup-manager/handlers/health"
	plansH "data-backup-manager/handlers/plans"
	restoresH "data-backup-manager/handlers/restores"
	runsH "data-backup-manager/handlers/runs"
	safetyH "data-backup-manager/handlers/safety"
	targetsH "data-backup-manager/handlers/targets"

	auditsint "data-backup-manager/internal/audits"
	coverageint "data-backup-manager/internal/coverage"
	destint "data-backup-manager/internal/destinations"
	discoveryint "data-backup-manager/internal/discovery"
	drillsint "data-backup-manager/internal/drills"
	plansint "data-backup-manager/internal/plans"
	restoresint "data-backup-manager/internal/restores"
	runsint "data-backup-manager/internal/runs"
	safetyint "data-backup-manager/internal/safety"
	scenariospecint "data-backup-manager/internal/scenariospec"
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

func sqlitePath() (string, error) {
	if path, ok := lookupEnvTrimmed("SQLITE_PATH"); ok {
		return path, nil
	}
	if path, ok := lookupEnvTrimmed("SQLITE_DB"); ok {
		return path, nil
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
	return path, nil
}

// databasePreflight prevents a previously initialized catalog from silently
// becoming a fresh empty catalog after a cleanup, retention, or mount mistake.
// A genuinely new installation is still allowed to create its first database;
// the marker makes subsequent absence a loud recovery condition.
func databasePreflight(path string) error {
	if strings.HasPrefix(path, "file:") {
		return nil
	}
	info, err := os.Stat(path)
	if err == nil {
		if info.Size() == 0 {
			return fmt.Errorf("data-backup-manager database %q is empty; refusing to initialize over a possible loss", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect data-backup-manager database %q: %w", path, err)
	}
	if _, markerErr := os.Stat(path + ".initialized"); markerErr == nil {
		return fmt.Errorf("data-backup-manager database %q is absent after prior initialization; restore it before starting", path)
	}
	return nil
}

func markDatabaseInitialized(path string) error {
	if strings.HasPrefix(path, "file:") {
		return nil
	}
	marker := path + ".initialized"
	if _, err := os.Stat(marker); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(marker, []byte("data-backup-manager database initialized\n"), 0o600)
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
	databasePath, err := sqlitePath()
	if err != nil {
		return fmt.Errorf("sqlite configuration failed: %w", err)
	}
	if err := databasePreflight(databasePath); err != nil {
		return fmt.Errorf("database preflight failed: %w", err)
	}
	dsn, err := sqliteFileDSN(databasePath)
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
	if err := markDatabaseInitialized(databasePath); err != nil {
		_ = db.Close()
		return fmt.Errorf("record database initialization marker: %w", err)
	}

	// Additive column migration for the destinations table (SQLite has no
	// ADD COLUMN IF NOT EXISTS); idempotent on fresh and existing databases.
	if err := destint.EnsureColumns(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return fmt.Errorf("destinations column migration failed: %w", err)
	}
	if err := plansint.EnsureColumns(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return fmt.Errorf("plans column migration failed: %w", err)
	}
	if err := targetsint.EnsureColumns(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return fmt.Errorf("targets column migration failed: %w", err)
	}

	// Additive column migration for the runs table (error, updated_at) so the
	// async-execution columns land on a database that predates them without
	// losing run history.
	if err := runsint.EnsureColumns(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return fmt.Errorf("runs column migration failed: %w", err)
	}

	// Additive column migration for the restores table (updated_at heartbeat) so
	// async-restore columns land on a database that predates them.
	if err := restoresint.EnsureColumns(ctx, db.Primary()); err != nil {
		_ = db.Close()
		return fmt.Errorf("restores column migration failed: %w", err)
	}

	// The backup engine (resource-kopia) wrapped behind the KopiaEngine seam,
	// and the storage root the manager protects (a destination must not point
	// under it — the separate-root rule). SCENARIO_DATA_DIR is set by the
	// lifecycle; empty in bare dev runs, which makes the rule permissive.
	clk := schedule.System()
	logger := log.Default()
	kopia := engine.NewKopiaCLI()
	protectedRoot, _ := lookupEnvTrimmed("SCENARIO_DATA_DIR")
	fileResolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return fmt.Errorf("create file storage resolver: %w", err)
	}
	filePaths, err := fileResolver.Resolve(storage.Options{ScenarioID: "data-backup-manager"})
	if err != nil {
		return fmt.Errorf("resolve file storage roots: %w", err)
	}
	fileRoots := filerouting.New(filePaths)

	// Concrete domain services used both for mounting (via each module) and as
	// the backing for the cross-domain adapters the run orchestration needs.
	targetsSvc := targetsint.NewService(targetsint.NewSQLiteRepository(db, clk))
	destSvc := destint.NewService(destint.NewSQLiteRepository(db, clk), kopia, &destint.FSBundleWriter{}, protectedRoot)
	if err := destSvc.ReconcileCredentialReferences(ctx); err != nil {
		// A destination may be mounted read-only while the manager is still
		// able to serve catalog/readiness requests. Keep the service available,
		// surface the exact pending migration, and retry on the next boot.
		logger.Printf("destination credential-reference reconciliation pending: %v", err)
	}
	planReadiness := destinationreadiness.NewService(
		destinationreadiness.NewReadOnlyInspector(sysmounts.New()),
		destinationreadiness.NewLocalPreparer(),
	)

	// Discovery: read-only onboarding suggestions. Scans well-known runtime
	// state (~/.vrooli) for targets and mounted volumes for destinations,
	// filtering against the live catalog + a dismissals table. Suggestions are
	// derived; only dismissals persist. The protected-path set is computed here
	// (runtime root + registered destinations + registered target locators) —
	// wider than the destinations service's own protectedRoot (D4). Built before
	// plans because the plan coverage guard reads its recommendations.
	sourceScanner := discoveryint.NewCachedTargetSourceScanner(
		discoveryint.NewCompositeScanner(
			discoveryint.NewWellKnownScanner(),
			discoveryint.NewResourceDataScanner(),
		),
		30*time.Second,
	)
	// Warm the bounded read-only inventory before advertising API health. The
	// first overview request must not pay the full repository walk, and later
	// concurrent coverage/suggestions requests reuse this snapshot.
	if _, err := sourceScanner.Scan(ctx); err != nil {
		logger.Printf("discovery inventory warm-up unavailable: %v", err)
	}

	discoverySvc := discoveryint.NewService(discoveryint.Deps{
		Volumes: sysmounts.New(),
		// Two source scanners behind one seam: Vrooli's own runtime home
		// (~/.vrooli) plus every declared non-regenerable owner storage entry.
		Sources:      sourceScanner,
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
	plansSvc := plansint.NewServiceWithPolicies(
		plansint.NewSQLiteRepository(db, clk),
		planCoverageGuard{svc: guardCoverage},
		planCriticalTargetPolicy{svc: targetsSvc},
		planCriticalDestinationPolicy{targets: targetsSvc, destinations: destSvc, readiness: planReadiness},
	)

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
	// Late-bound: the scheduler is constructed below (it needs runsSvc as its
	// trigger), then assigned into this adapter before any request runs.
	nextSched := &nextScheduleAdapter{plans: plansSvc}
	runsSvc := runsint.NewService(runsint.Deps{
		Repo:                 runsint.NewSQLiteRepository(db, clk),
		Plans:                planLookup{svc: plansSvc},
		Targets:              targetLookup{svc: targetsSvc},
		ActiveTargets:        targetLookup{svc: targetsSvc},
		Destinations:         destinationLookup{svc: destSvc},
		Engine:               kopia,
		Sources:              sourceRegistry,
		Events:               logEventSink{logger: logger},
		Clock:                clk,
		Logger:               logger,
		BaseContext:          execCtx,
		TargetConcurrency:    concurrency,
		OverdueAfter:         overdueAfterDur,
		NextSchedule:         nextSched,
		PreflightSourcePaths: true,
		Readiness:            planReadiness,
		RoutedRoots:          fileRoots,
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
		// Restores run async on a background worker bound to the server-lifetime
		// context, so the request RPC returns immediately and a client disconnect
		// cannot sever an in-flight restore (no more 6h WriteTimeout).
		BaseContext: execCtx,
		Workers:     concurrency,
		Logger:      logger,
	})

	// Startup reconciliation: close any restore left non-terminal by a prior
	// crash/restart/disconnect, fail-not-resume (never falsely "verified").
	if err := restoresSvc.Reconcile(ctx); err != nil {
		cancelExec()
		_ = db.Close()
		return fmt.Errorf("restore reconciliation failed: %w", err)
	}

	// Generic snapshot audit: restore a snapshot to scratch, capture the live
	// target to scratch (read-only on live), walk both, and compare by generic
	// signals only. Scenario-agnostic — DBM never learns a domain's objects.
	// Runs async on a background worker bound to the server-lifetime context.
	auditsSvc := auditsint.NewService(auditsint.Deps{
		Repo:         auditsint.NewSQLiteRepository(db, clk),
		Targets:      auditTargetLookup{svc: targetsSvc},
		Destinations: auditDestinationLookup{svc: destSvc},
		Engine:       kopia,
		Sources:      sourceRegistry,
		Clock:        clk,
		BaseContext:  execCtx,
		Workers:      concurrency,
		Logger:       logger,
	})

	// Startup reconciliation: close any audit left non-terminal by a prior
	// crash/restart/disconnect, fail-not-resume.
	if err := auditsSvc.Reconcile(ctx); err != nil {
		cancelExec()
		_ = db.Close()
		return fmt.Errorf("audit reconciliation failed: %w", err)
	}

	// Recovery drills reuse the verified-restore gate but add durable policy,
	// idempotency, latest-successful-snapshot selection, and scheduled operator
	// evidence. They never write to a live target.
	drillsSvc := drillsint.NewService(drillsint.Deps{
		Repo:        drillsint.NewSQLiteRepository(db),
		Plans:       drillPlanLookup{svc: plansSvc},
		Snapshots:   drillSnapshotLookup{svc: runsSvc},
		Restores:    drillRestoreRunner{svc: restoresSvc},
		Clock:       clk,
		BaseContext: execCtx,
		Workers:     concurrency,
		Logger:      logger,
	})
	if err := drillsSvc.Reconcile(ctx); err != nil {
		cancelExec()
		_ = db.Close()
		return fmt.Errorf("recovery-drill reconciliation failed: %w", err)
	}
	if err := startDrillScheduler(ctx, drillsSvc, logger); err != nil {
		_ = db.Close()
		return err
	}

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
	nextSched.sched = scheduler // late bind: see nextScheduleAdapter
	if err := startScheduler(ctx, scheduler, logger); err != nil {
		_ = db.Close()
		return err
	}

	// Health reports backup posture (degraded when targets are overdue/failed)
	// in addition to liveness, and emits a posture event for monitoring. It
	// reads the same per-target Overdue the runs service computes from
	// overdueAfterDur, so /health and `runs status` never disagree.
	posture := backupPosture{runs: runsSvc}

	// Baseline Modes safety substrate: pure orchestration over the
	// destinations/targets/plans/runs services (no tables of its own) that the
	// platform recovery floor shells out to for pre-promote scenario snapshots.
	safetySvc := safetyint.NewService(safetyint.Deps{
		Destinations: safetyDestinations{svc: destSvc},
		Targets:      safetyTargets{svc: targetsSvc},
		Plans:        safetyPlans{svc: plansSvc},
		Runs:         safetyRuns{svc: runsSvc},
		Restores:     safetyRestores{svc: restoresSvc},
		Inspector:    safetyScenarioInspector{insp: scenariospecint.NewInspector()},
		RuntimeRoot:  discoveryint.RuntimeRoot,
	})

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.ModuleWithPosture(db, "data-backup-manager-api", "1.0.0", posture, logEventSink{logger: logger}),
		auditsH.Module(auditsSvc, logger),
		coverageH.Module(coverageSvc, logger),
		destinationsH.Module(db, clk, kopia, protectedRoot, logger),
		discoveryH.Module(discoverySvc, logger),
		plansH.Module(plansSvc, logger),
		restoresH.Module(restoresSvc, logger),
		drillsH.Module(drillsSvc, logger),
		runsH.Module(runsSvc, verifiedLookup{svc: restoresSvc}, logger),
		safetyH.Module(safetySvc, logger),
		targetsH.Module(db, clk, logger),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.HandleFunc("/health", healthHandler(srv.Handler()))
	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// Both backup runs AND restores/verifies are now async: their RPCs persist
		// a record, schedule the kopia work on a server-lifetime background worker,
		// and return immediately. Nothing blocks the HTTP handler for the duration
		// of a kopia operation anymore, so the api-core default WriteTimeout is
		// sufficient — the 6h override (and the CLI's unlimited-timeout client)
		// that the synchronous restore path needed are gone.
		Cleanup: func(cctx context.Context) error {
			// Interrupt and drain in-flight runs + restores before closing the DB
			// so no worker can write to a closed handle.
			cancelExec()
			_ = runsSvc.Shutdown(cctx)
			_ = restoresSvc.Shutdown(cctx)
			_ = auditsSvc.Shutdown(cctx)
			_ = drillsSvc.Shutdown(cctx)
			return db.Close()
		},
	}); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
