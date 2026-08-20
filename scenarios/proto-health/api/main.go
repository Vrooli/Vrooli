package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"proto-health/internal/modules"
	"proto-health/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	_ "modernc.org/sqlite"

	healthH "proto-health/handlers/health"
	impactH "proto-health/handlers/impact"
	validationH "proto-health/handlers/validation"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "proto-health"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "proto-health",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		log.Fatalf("repo root discovery failed: %v", err)
	}
	descriptorSource, err := descriptorimage.NewForRepo(repoRoot)
	if err != nil {
		log.Fatalf("proto descriptor source: %v", err)
	}
	if _, err := descriptorSource.LoadWithRetry(5, 100*time.Millisecond); err != nil {
		log.Fatalf("proto descriptor initial load: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "proto-health-api", "1.0.0", descriptorSource),
		impactH.Module(log.Default(), repoRoot, descriptorSource),
		validationH.Module(log.Default(), repoRoot, descriptorSource),
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
