package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"

	"measures-health/internal/modules"
	"measures-health/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	healthH "measures-health/handlers/health"
	measuresH "measures-health/handlers/measures"
	searchH "measures-health/handlers/search"
	validationH "measures-health/handlers/validation"
	"measures-health/internal/runhistory"
	validationcore "measures-health/internal/validation"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "measures-health"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "measures-health",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	repoRoot := validationcore.ResolveRepoRoot()

	// runhistory persists measures-health's own validation_run entity — the
	// substrate the `validation_run` measures aggregate over (the gold-star
	// dogfood). The validation handler writes a row per top-level ValidateScenario;
	// the measures handler reads counts over it.
	runs := runhistory.New(db, nil)
	measuresModule, err := measuresH.Module(runs, nil, log.Default())
	if err != nil {
		log.Fatalf("measures module init failed: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "measures-health-api", "1.0.0"),
		validationH.Module(repoRoot, runs, log.Default()),
		searchH.Module(repoRoot, log.Default()),
		measuresModule,
	)

	// Self-register the central measures provider with search-hub from the
	// scenario-owned `.vrooli/search.json` SSOT. search-hub is an OPTIONAL
	// dependency, so this runs in the background with bounded retry and degrades
	// gracefully — measures-health serves search whether or not the hub is up,
	// and the registry upsert is idempotent so re-registering each boot is safe.
	registerCtx, cancelRegister := context.WithCancel(context.Background())
	defer cancelRegister()
	searchJSONPath := filepath.Join(repoRoot, "scenarios", "measures-health", ".vrooli", "search.json")
	go searchregister.Register(registerCtx, searchregister.Config{
		ScenarioID:     "measures-health",
		SearchFilePath: searchJSONPath,
		Logger:         log.Default(),
	})

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
