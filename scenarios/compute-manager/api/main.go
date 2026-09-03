package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"compute-manager/internal/capabilities"
	appconfig "compute-manager/internal/config"
	"compute-manager/internal/modules"
	"compute-manager/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	capsH "compute-manager/handlers/capabilities"
	healthH "compute-manager/handlers/health"
	instanceH "compute-manager/handlers/instance"
	intentH "compute-manager/handlers/intent"
	meterH "compute-manager/handlers/meter"
	reconcileH "compute-manager/handlers/reconcile"
	internalinstance "compute-manager/internal/instance"
	"compute-manager/internal/provider"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. Domain file stores receive the routed roots and select the
// request-appropriate class at their own storage seam.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("compute-manager")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve compute-manager storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "compute-manager"}) {
		return
	}
	if _, err := appconfig.Load(); err != nil {
		log.Fatalf("configuration invalid: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "compute-manager",
		MaxOpenConns: 8,
		MaxIdleConns: 8,
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

	instanceModule, err := instanceH.NewModuleWithRepository(&provider.Fake{}, time.Now, &internalinstance.Repository{DB: db.Primary()})
	if err != nil {
		log.Fatalf("instance module configuration failed: %v", err)
	}
	intentModule, err := intentH.NewModule(db.Primary())
	if err != nil {
		log.Fatalf("intent module configuration failed: %v", err)
	}
	meterModule, err := meterH.NewModule(db.Primary())
	if err != nil {
		log.Fatalf("meter module configuration failed: %v", err)
	}
	reconcileModule, err := reconcileH.NewModule(db.Primary())
	if err != nil {
		log.Fatalf("reconcile module configuration failed: %v", err)
	}
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "compute-manager-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		instanceModule,
		intentModule,
		meterModule,
		reconcileModule,
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
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
