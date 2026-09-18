package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"nutrition-planner/internal/capabilities"
	"nutrition-planner/internal/modules"
	"nutrition-planner/internal/server"

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

	capsH "nutrition-planner/handlers/capabilities"
	catalogH "nutrition-planner/handlers/catalog"
	costH "nutrition-planner/handlers/cost"
	diagnosticsH "nutrition-planner/handlers/diagnostics"
	eligibilityH "nutrition-planner/handlers/eligibility"
	healthH "nutrition-planner/handlers/health"
	inventoryH "nutrition-planner/handlers/inventory"
	jobsH "nutrition-planner/handlers/jobs"
	nutritionH "nutrition-planner/handlers/nutrition"
	planningH "nutrition-planner/handlers/planning"
	portabilityH "nutrition-planner/handlers/portability"
	profileH "nutrition-planner/handlers/profile"
	recipeH "nutrition-planner/handlers/recipe"
	routineH "nutrition-planner/handlers/routine"
	supplementH "nutrition-planner/handlers/supplement"
	workspaceH "nutrition-planner/handlers/workspace"
	internalEntitlements "nutrition-planner/internal/entitlements"
	internalFeedback "nutrition-planner/internal/feedback"
	internalJobs "nutrition-planner/internal/jobs"
	internalPlanning "nutrition-planner/internal/planning"
	internalPortability "nutrition-planner/internal/portability"
	internalProfile "nutrition-planner/internal/profile"
	internalRecipe "nutrition-planner/internal/recipe"
	internalShopping "nutrition-planner/internal/shopping"
	internalWorkspace "nutrition-planner/internal/workspace"
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
	scenarioID, err := storage.ScenarioNamespace("nutrition-planner")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve nutrition-planner storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "nutrition-planner"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "nutrition-planner",
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

	workspaceService := internalWorkspace.NewService(internalWorkspace.NewSQLiteRepository(db, schedule.System()))
	recipeService := internalRecipe.NewService(internalRecipe.NewSQLiteRepository(db, schedule.System()))
	profileService := internalProfile.NewService(internalProfile.NewSQLiteRepository(db, schedule.System()))
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "nutrition-planner-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		diagnosticsH.Module(db, workspaceService),
		catalogH.Module(db, schedule.System(), log.Default()),
		costH.Module(db, schedule.System(), log.Default()),
		inventoryH.Module(db, schedule.System(), log.Default()),
		nutritionH.Module(db, schedule.System(), log.Default()),
		supplementH.Module(db, schedule.System(), log.Default()),
		routineH.Module(db, schedule.System(), log.Default()),
		workspaceH.ModuleWithService(workspaceService, log.Default()),
		recipeH.ModuleWithService(recipeService, workspaceService, log.Default()),
		portabilityH.ModuleWithRestorer(recipeService, workspaceService, internalPlanning.NewSQLiteRepository(db, schedule.System()), internalShopping.NewSQLiteRepository(db, schedule.System()), internalPortability.NewSQLiteRestorer(db, schedule.System()), log.Default()),
		profileH.ModuleWithServices(profileService, workspaceService, recipeService, log.Default()),
		eligibilityH.ModuleWithWorkspace(workspaceService, log.Default()),
		jobsH.Module(internalJobs.NewSQLiteRepository(db, schedule.System()), workspaceService, log.Default(), internalEntitlements.NewSQLiteRepository(db)),
		planningH.ModuleWithServices(workspaceService, recipeService, profileService, internalPlanning.NewSQLiteRepository(db, schedule.System()), internalShopping.NewSQLiteRepository(db, schedule.System()), internalFeedback.NewSQLiteRepository(db, schedule.System()), log.Default()),
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
