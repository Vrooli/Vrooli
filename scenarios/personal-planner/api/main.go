package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"personal-planner/internal/modules"
	"personal-planner/internal/server"

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

	calendarH "personal-planner/handlers/calendar"
	focusH "personal-planner/handlers/focus"
	goalsH "personal-planner/handlers/goals"
	healthH "personal-planner/handlers/health"
	integrationsH "personal-planner/handlers/integrations"
	reviewH "personal-planner/handlers/review"
	workH "personal-planner/handlers/work"
	workspaceH "personal-planner/handlers/workspace"
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
	scenarioID, err := storage.ScenarioNamespace("personal-planner")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve personal-planner storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "personal-planner"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "personal-planner",
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

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "personal-planner-api", "1.0.0"),
		focusH.Module(db, schedule.System(), log.Default()),
		calendarH.Module(db, schedule.System(), log.Default()),
		goalsH.Module(db, schedule.System(), log.Default()),
		integrationsH.Module(db, schedule.System(), log.Default()),
		reviewH.Module(db, schedule.System(), log.Default()),
		workH.Module(db, schedule.System(), log.Default()),
		workspaceH.Module(db, schedule.System(), log.Default()),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())
	authConfig, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		log.Fatalf("authentication configuration failed: %v", err)
	}
	var sharedAuth *authn.Config
	if authConfig.Enabled() {
		sharedAuth = &authConfig
	}

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler:        handler,
		Authentication: sharedAuth,
		Cleanup:        func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
