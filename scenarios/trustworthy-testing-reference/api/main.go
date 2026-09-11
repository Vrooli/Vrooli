package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"trustworthy-testing-reference/internal/capabilities"
	"trustworthy-testing-reference/internal/modules"
	"trustworthy-testing-reference/internal/server"

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

	capsH "trustworthy-testing-reference/handlers/capabilities"
	healthH "trustworthy-testing-reference/handlers/health"
	notesH "trustworthy-testing-reference/handlers/notes" // EXAMPLE-DOMAIN:notes
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
	scenarioID, err := storage.ScenarioNamespace("trustworthy-testing-reference")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve trustworthy-testing-reference storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "trustworthy-testing-reference"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "trustworthy-testing-reference",
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
		healthH.Module(db, "trustworthy-testing-reference-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		notesH.Module(db, schedule.System(), log.Default()), // EXAMPLE-DOMAIN:notes
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	// EXAMPLE-DOMAIN:notes START
	// /measures is the measures-go serve substrate: the central measures
	// index (measures-health) harvests <prefix>/declarations and the
	// auto-execution path POSTs <prefix>/execute. The notes domain owns the
	// one reference measure (notes.count); a real multi-domain scenario
	// registers each domain's measures on one shared registry here.
	notesMeasures, err := notesH.MeasuresHandler(db, schedule.System())
	if err != nil {
		log.Fatalf("measures registry: %v", err)
	}
	rootMux.Handle("/measures/", http.StripPrefix("/measures", notesMeasures))
	// EXAMPLE-DOMAIN:notes END

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
