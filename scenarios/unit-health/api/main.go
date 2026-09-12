package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"unit-health/internal/modules"
	"unit-health/internal/runhistory"
	"unit-health/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	healthH "unit-health/handlers/health"
	validationH "unit-health/handlers/validation"
)

func main() {
	if err := runApplication(); err != nil {
		log.Fatal(err)
	}
}

var (
	runApplication = startApplication
	preflightRun   = preflight.Run
	runAPIServer   = apiserver.Run
)

func startApplication() error {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflightRun(preflight.Config{ScenarioName: "unit-health"}) {
		return nil
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "unit-health",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return fmt.Errorf("Database connection failed: %w", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		return fmt.Errorf("schema initialization failed: %w", err)
	}

	logger := log.Default()
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: logger},
		healthH.Module(db, "unit-health-api", "1.0.0"),
		validationH.Module(logger, repoRoot, runhistory.NewRepository(db.Primary())),
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

	// Unit validation can execute target test suites before writing its
	// Connect response; the api-core 30s default write timeout is too short.
	if err := runAPIServer(apiserver.Config{
		Handler:      handler,
		WriteTimeout: 25 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		return fmt.Errorf("Server error: %w", err)
	}
	return nil
}
