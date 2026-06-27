package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"architecture-cartographer/internal/app"
	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/config"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/modules"
	"architecture-cartographer/internal/observability"
	"architecture-cartographer/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "architecture-cartographer"}) {
		return
	}

	dsn, err := app.SQLiteDSN()
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

	if err := graph.MigrateSchema(context.Background(), db.Primary()); err != nil {
		log.Fatalf("schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	clk := clock.System{}
	repoRoot, repoErr := repocontract.FindRepoRootFromEnvOrCWD()
	if repoErr != nil {
		log.Printf("cartographer: repo root resolution failed: %v", repoErr)
	}

	// Cartographer-global control surface (tunable levers; no per-scenario
	// config). Misconfigured levers degrade to defaults with a logged
	// diagnostic rather than failing startup.
	cfg, cfgDiags := config.Load(os.Getenv)
	for _, d := range cfgDiags {
		log.Printf("cartographer config: %s: %s", d.Key, d.Message)
	}

	srv := server.New(
		server.Deps{Clock: clk, Logger: log.Default()},
		app.Modules(db, repoRoot, cfg)...,
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	observability.RegisterPprof(rootMux, cfg.PprofEnabled)
	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler:      handler,
		WriteTimeout: 5 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
