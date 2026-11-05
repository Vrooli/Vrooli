package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"meta-optimization-manager/internal/modules"
	"meta-optimization-manager/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	convergenceH "meta-optimization-manager/handlers/convergence"
	coverageH "meta-optimization-manager/handlers/coverage"
	focusH "meta-optimization-manager/handlers/focus"
	healthH "meta-optimization-manager/handlers/health"
	trialsH "meta-optimization-manager/handlers/trials"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "meta-optimization-manager"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "meta-optimization-manager",
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
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "meta-optimization-manager-api", "1.0.0"),
		coverageH.Module(db, schedule.System(), log.Default()),
		focusH.Module(db, schedule.System(), log.Default()),
		convergenceH.Module(db, schedule.System(), log.Default()),
		trialsH.Module(db, schedule.System(), log.Default()),
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
		// Focus can perform a bounded, concurrent fleet maturity read. Its
		// response must outlive the per-target validation budget rather than
		// being truncated by the server's short default write deadline.
		WriteTimeout: 2 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
