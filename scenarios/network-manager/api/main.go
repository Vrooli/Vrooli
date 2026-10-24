package main

import (
	"context"
	"log"
	"net/http"

	"network-manager/internal/modules"
	"network-manager/internal/resolver"
	"network-manager/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	adaptersH "network-manager/handlers/adapters"
	healthH "network-manager/handlers/health"
	homeintegrationH "network-manager/handlers/homeintegration"
	inventoryH "network-manager/handlers/inventory"
	monitoringH "network-manager/handlers/monitoring"
	optimizationH "network-manager/handlers/optimization"
	policyH "network-manager/handlers/policy"
	privacyH "network-manager/handlers/privacy"
	resolverH "network-manager/handlers/resolver"
	snapshotH "network-manager/handlers/snapshot"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "network-manager"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "network-manager",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := resolver.Migrate(context.Background(), db.Primary()); err != nil {
		log.Fatalf("schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "network-manager-api", "1.0.0"),
		adaptersH.Module(db),
		homeintegrationH.Module(db),
		inventoryH.Module(db),
		monitoringH.Module(db),
		optimizationH.Module(db),
		policyH.Module(db),
		privacyH.Module(db),
		resolverH.Module(db),
		snapshotH.Module(db),
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
