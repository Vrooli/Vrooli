package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"plan-manager/internal/clock"
	internalexecution "plan-manager/internal/execution"
	"plan-manager/internal/modules"
	internalplanlog "plan-manager/internal/planlog"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	authoringH "plan-manager/handlers/authoring"
	executionH "plan-manager/handlers/execution"
	healthH "plan-manager/handlers/health"
	planlogH "plan-manager/handlers/planlog"
	plansH "plan-manager/handlers/plans"
	validationH "plan-manager/handlers/validation"
)

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The path scope is the variant-aware namespace (storage.ScenarioNamespace),
// not the bare slug: under a Baseline Modes shadow engagement the lifecycle
// injects VROOLI_STORAGE_NAMESPACE, so the shadow's SQLite file lands beside
// "<scenario>_shadow" and never shares live's database. Outside the lifecycle
// (local `go run`, tests) it falls back to the compile-time slug, so live paths
// are unchanged. This is why a generated scenario is shadow-safe with zero
// per-scenario work — see packages/api-core/storage/namespace.go.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path, ok := os.LookupEnv("SQLITE_PATH"); ok {
		path = strings.TrimSpace(path)
		if path == "" {
			return "", fmt.Errorf("SQLITE_PATH is set but empty")
		}
		return sqliteFileDSN(path)
	}
	if path, ok := os.LookupEnv("SQLITE_DB"); ok {
		path = strings.TrimSpace(path)
		if path == "" {
			return "", fmt.Errorf("SQLITE_DB is set but empty")
		}
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("plan-manager")
	if err != nil {
		return "", fmt.Errorf("resolve plan-manager storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"plan-manager.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve plan-manager db path: %w", err)
	}
	return sqliteFileDSN(path)
}

// sqlitePathFromDSN recovers the physical file path from a sqlite DSN so
// finalize can report the store identity it wrote to.
func sqlitePathFromDSN(dsn string) string {
	path := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	return path
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
	if preflight.Run(preflight.Config{ScenarioName: "plan-manager"}) {
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

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "plan-manager-api", "1.0.0"),
		plansH.Module(db, clock.System{}, log.Default()),
		validationH.Module(db, clock.System{}, log.Default()),
		authoringH.Module(db, clock.System{}, log.Default(), sqlitePathFromDSN(dsn)),
		executionH.Module(db, clock.System{}, log.Default()),
		planlogH.Module(db, clock.System{}, log.Default(), newPlanLogResolver(db, clock.System{})),
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

// planLogResolver maps a `plan_or_execution` handle to the canonical plan id
// and optional execution id. This is cross-domain composition, so it lives in
// main rather than inside the log transport module.
type planLogResolver struct {
	plans      internalplans.Service
	executions internalexecution.Repository
}

func newPlanLogResolver(db *database.RoutedDB, clk clock.Clock) planLogResolver {
	return planLogResolver{
		plans: internalplans.NewService(internalplans.Deps{
			Repo:  internalplans.NewSQLiteRepository(db, clk),
			Clock: clk,
		}),
		executions: internalexecution.NewSQLiteRepository(db, clk),
	}
}

func (r planLogResolver) Resolve(ctx context.Context, handle string) (internalplanlog.Scope, bool, error) {
	if e, found, gerr := r.executions.GetExecution(ctx, handle); gerr != nil {
		return internalplanlog.Scope{}, false, gerr
	} else if found {
		plan, perr := r.plans.Get(ctx, e.PlanID, internalplans.WorkspaceScope{})
		if perr != nil {
			return internalplanlog.Scope{}, false, perr
		}
		return internalplanlog.Scope{
			PlanID:         e.PlanID,
			ExecutionID:    e.ID,
			CurrentPhaseID: e.CurrentPhaseID,
			Phases:         logPhaseRefs(plan.Phases),
		}, true, nil
	}
	plan, gerr := r.plans.Get(ctx, handle, internalplans.WorkspaceScope{})
	if gerr != nil {
		return internalplanlog.Scope{}, false, nil
	}
	return internalplanlog.Scope{
		PlanID: plan.ID,
		Phases: logPhaseRefs(plan.Phases),
	}, true, nil
}

func logPhaseRefs(phases []internalplans.Phase) []internalplanlog.PhaseRef {
	out := make([]internalplanlog.PhaseRef, 0, len(phases))
	for _, ph := range phases {
		out = append(out, internalplanlog.PhaseRef{ID: ph.ID, Order: ph.Order, Title: ph.Title})
	}
	return out
}
