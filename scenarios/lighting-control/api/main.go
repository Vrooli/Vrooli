package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"lighting-control/internal/capabilities"
	internalcontrol "lighting-control/internal/control"
	"lighting-control/internal/modules"
	internalpolicy "lighting-control/internal/policy"
	"lighting-control/internal/policy/sources"
	"lighting-control/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	"lighting-control/handlers/browsersession"
	capsH "lighting-control/handlers/capabilities"
	controlH "lighting-control/handlers/control"
	healthH "lighting-control/handlers/health"
	policyH "lighting-control/handlers/policy"
	runtimebindingH "lighting-control/handlers/runtimebinding"
	setupH "lighting-control/handlers/setup"
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
	scenarioID, err := storage.ScenarioNamespace("lighting-control")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve lighting-control storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

// initializeDatabase gives every admitted pool the same domain schema before
// requests can route to it. A leased database starts empty; primary bootstrap
// alone does not prepare it for the real control and policy repositories.
func initializeDatabase(ctx context.Context, db *database.RoutedDB) error {
	initialize := func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return fmt.Errorf("initialize lighting schemas: %w", err)
		}
		return internalcontrol.EnsureIdentitySource(ctx, pool)
	}
	if err := initialize(ctx, db.Primary()); err != nil {
		return err
	}
	db.SetTestPoolInitializer(initialize)
	return nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "lighting-control"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "lighting-control",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := initializeDatabase(context.Background(), db); err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	clock := schedule.System()
	backendBinding := internalcontrol.NewBackendBinding()
	controlService := controlH.NewDomainService(db, clock, backendBinding)
	policyService := policyH.NewService(internalpolicy.NewSQLiteStore(db), sources.New(), controlService)
	runtimeCtx, cancelRuntime := context.WithCancel(context.Background())
	stopNativeBinding, err := runtimebindingH.StartNativeFromEnv(runtimeCtx, controlService, os.Getenv)
	if err != nil {
		log.Printf("native lighting binding unavailable: %v", err)
		stopNativeBinding = func() {}
	}
	policyDone := make(chan struct{})
	go func() {
		defer close(policyDone)
		policyService.Run(runtimeCtx, 30*time.Second)
	}()
	srv := server.New(
		server.Deps{Clock: clock, Logger: log.Default()},
		healthH.Module(db, "lighting-control-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		controlH.ModuleWithDomainService(controlService, log.Default()),
		policyH.ModuleWithService(policyService, log.Default()),
		setupH.Module(),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	browsersession.Register(rootMux)
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())
	// Native lifecycle launches have one private backend owner. Standalone
	// launches retain the independently authenticated human maintenance route.
	if os.Getenv("VROOLI_RUNTIME_DIAGNOSTICS_DIR") == "" {
		runtimebindingH.Register(rootMux, controlService, os.Getenv("VROOLI_COMPONENT_GENERATION"))
	}
	authConfig, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		log.Fatalf("authentication configuration failed: %v", err)
	}

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler:        handler,
		Authentication: &authConfig,
		Cleanup: func(ctx context.Context) error {
			cancelRuntime()
			<-policyDone
			stopNativeBinding()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
