package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"backdrop-studio/internal/capabilities"
	internalcatalog "backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/modules"
	internalrelease "backdrop-studio/internal/release"
	internalrender "backdrop-studio/internal/render"
	"backdrop-studio/internal/server"

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

	capsH "backdrop-studio/handlers/capabilities"
	catalogH "backdrop-studio/handlers/catalog"
	composeH "backdrop-studio/handlers/compose"
	generatorsH "backdrop-studio/handlers/generators"
	healthH "backdrop-studio/handlers/health"
	legibilityH "backdrop-studio/handlers/legibility"
	releaseH "backdrop-studio/handlers/release"
	renderH "backdrop-studio/handlers/render"
	scaffoldH "backdrop-studio/handlers/scaffold"
	surfacesH "backdrop-studio/handlers/surfaces"
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
	scenarioID, err := storage.ScenarioNamespace("backdrop-studio")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve backdrop-studio storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "backdrop-studio"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "backdrop-studio",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := internalcatalog.NewStore(db.Primary()).Seed(context.Background()); err != nil {
		log.Fatalf("catalog seed failed: %v", err)
	}
	imageClient := imageengine.NewClient()
	// The render store reaches authored generators through the catalog, which
	// is the single authority on what a generator is. A style bound to one that
	// is absent or unvalidated is refused by name rather than substituted.
	renderStore := internalrender.NewStoreWithGenerator(imageClient, imageClient).
		WithGeneratorStore(internalcatalog.NewStore(db.Primary()))
	// The release path's asset-studio capability. It resolves lazily so a
	// scenario that is not running becomes a named missing capability at the
	// moment of release rather than a startup failure — and so a model-backed
	// release refuses with that name instead of quietly shipping an
	// undisclosed synthetic image.
	assetPublisher := &internalrelease.AssetStudioPublisher{
		Resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "asset-studio")
		},
	}
	// The render store is the provenance source: a candidate's model, prompt
	// and seed are read from the render that produced them, never from the
	// caller asking for the release.
	releaseStore := internalrelease.NewStoreWithPublisher(assetPublisher, renderStore)
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "backdrop-studio-api", "1.0.0", internalcatalog.NewStore(db.Primary()).AppliedSeedVersion),
		capsH.Module(capabilities.NewRegistry()),
		catalogH.Module(db.Primary()),
		composeH.Module(db.Primary()),
		renderH.Module(db.Primary(), renderStore),
		legibilityH.Module(),
		releaseH.Module(releaseStore),
		scaffoldH.Module(db.Primary()),
		generatorsH.Module(db.Primary()),
		surfacesH.Module(db.Primary()),
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
