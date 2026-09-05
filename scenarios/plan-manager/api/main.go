package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	internalexecution "plan-manager/internal/execution"
	"plan-manager/internal/modules"
	internalplanlog "plan-manager/internal/planlog"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/server"
	internalvalidation "plan-manager/internal/validation"

	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/schedule"

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

// sqlitePathFromDSN recovers the physical file path from a sqlite DSN so
// finalize can report the store identity it wrote to.
func sqlitePathFromDSN(dsn string) string {
	path := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	return path
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "plan-manager"}) {
		return
	}

	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "plan-manager"})
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

	if err := internalvalidation.EnsureMigrations(context.Background(), db.Primary()); err != nil {
		log.Fatalf("validation storage migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := internalplans.EnsureMigrations(context.Background(), db.Primary()); err != nil {
		log.Fatalf("plan storage migration failed: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "plan-manager-api", "1.0.0"),
		plansH.Module(db, schedule.System(), log.Default()),
		validationH.Module(db, schedule.System(), log.Default()),
		authoringH.Module(db, schedule.System(), log.Default(), sqlitePathFromDSN(dsn)),
		executionH.Module(db, schedule.System(), log.Default()),
		planlogH.Module(db, schedule.System(), log.Default(), newPlanLogResolver(db, schedule.System())),
	)

	// Operational-table retention. Nothing in this API deleted anything before
	// now: a repository-wide search for DELETE statements found exactly one, and
	// it is a foreign-key cleanup, not retention. log_entries, validation
	// operations and candidate revisions therefore grew without bound.
	//
	// The budgets live in .vrooli/service.json and are enforced by the shared
	// engine; all three use the builtin age/bytes pruner, so no custom selection
	// rule is registered here. Deliberately NOT bounded: the plans table itself
	// (a plan is the durable product, not regenerable state), the authoring
	// sessions table (an age rule alone cannot tell a finished session from an
	// open one and would delete in-progress authoring), and the rendered mirror
	// tree at ~/.vrooli/plans, which the repo contract declares protected with
	// cleanup "never". Retention being unavailable is not fatal: a scenario that
	// cannot reclaim space still serves plans correctly.
	if retentionManager, retentionErr := retention.NewForScenario(retention.ScenarioConfig{
		Scenario:   "plan-manager",
		Registry:   retention.NewRegistry(),
		RunOnStart: true,
		Logger:     slog.Default(),
	}); retentionErr != nil {
		log.Printf("plan-manager: retention unavailable, operational tables will grow unbounded: %v", retentionErr)
	} else {
		retentionManager.Start(context.Background())
		defer retentionManager.Stop()
	}

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
		// One blocking validation attachment is budgeted at 15m. The operation
		// remains durable beyond this transport window; the extra margin lets the
		// handler return its terminal/current projection instead of being cut off
		// by api-core's 30s default write timeout.
		WriteTimeout: 17 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
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

func newPlanLogResolver(db *database.RoutedDB, clk schedule.Clock) planLogResolver {
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
