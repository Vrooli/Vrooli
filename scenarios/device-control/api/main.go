package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"device-control/internal/capabilities"
	internalflows "device-control/internal/flows"
	"device-control/internal/modules"
	"device-control/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	capsH "device-control/handlers/capabilities"
	controlH "device-control/handlers/control"
	flowsH "device-control/handlers/flows"
	healthH "device-control/handlers/health"
	"device-control/internal/control"
	strategyregistry "device-control/strategy/registry"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("device-control")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve device-control storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "device-control"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "device-control",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	gateway := internalflows.NewGateway(http.DefaultClient, discovery.ResolveScenarioURLDefault)
	controlService, err := control.NewWithDBAndAttached(strategyregistry.Default(), db, control.NewBridgeAttachedReader(http.DefaultClient, nil), fileRoots)
	if err != nil {
		log.Fatalf("device-control state initialization failed: %v", err)
	}
	controlService.SetAgentPlanner(internalflows.NewGatewayPlanner(gateway))

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "device-control-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		controlH.Module(controlService),
		flowsH.Module(internalflows.NewResolver(gateway)),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 2 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
