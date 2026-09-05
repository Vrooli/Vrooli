package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"

	"signal-inbox/internal/categories"
	"signal-inbox/internal/inference"
	"signal-inbox/internal/modules"
	"signal-inbox/internal/retrieval"
	"signal-inbox/internal/searchregistry"
	"signal-inbox/internal/server"
	"signal-inbox/internal/sources"
	"signal-inbox/internal/triage"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	_ "modernc.org/sqlite"

	categoriesH "signal-inbox/handlers/categories"
	healthH "signal-inbox/handlers/health"
	retrievalH "signal-inbox/handlers/retrieval"
	signalsH "signal-inbox/handlers/signals"
	sourcesH "signal-inbox/handlers/sources"
	triageH "signal-inbox/handlers/triage"
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
	scenarioID, err := storage.ScenarioNamespace("signal-inbox")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve signal-inbox storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "signal-inbox"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "signal-inbox",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	var inferenceClient inference.Client
	if gatewayURL, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "ai-gateway"); resolveErr != nil {
		slog.Warn("ai-gateway unavailable; capture will record uncategorized signals", "error", resolveErr)
	} else {
		inferenceClient = inference.NewGatewayClient(routingconnect.NewRoutingServiceClient(http.DefaultClient, gatewayURL))
	}
	categoryService := categories.NewService(categories.NewSQLiteRepository(db), schedule.System(), inferenceClient)
	triageService := triage.NewService(triage.NewSQLiteRepository(db), schedule.System())
	retrievalService := retrieval.NewService(retrieval.NewSQLiteRepository(db), schedule.System(), retrieval.NewQdrantSemanticSearch(inferenceClient))
	if _, err := categoryService.Bootstrap(context.Background()); err != nil {
		log.Fatalf("seed reserved category: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	signalsRuntime, err := signalsH.NewRuntimeWithRoutedRoots(db, schedule.System(), fileRoots, categoryService)
	if err != nil {
		log.Fatalf("signals runtime: %v", err)
	}
	sourcesService, err := sources.NewService(sources.NewSQLiteRepository(db), signalsRuntime.Service, schedule.System(), sources.ChromeBookmarksAdapter{}, sources.RedditSavedArchiveAdapter{}, sources.XAuthoredArchiveAdapter{}, sources.XLikesArchiveAdapter{})
	if err != nil {
		log.Fatalf("sources runtime: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "signal-inbox-api", "1.0.0"),
		categoriesH.Module(categoryService),
		signalsH.ModuleWithRuntime(signalsRuntime, log.Default()),
		sourcesH.Module(sourcesService),
		retrievalH.Module(retrievalService),
		triageH.Module(triageService),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// Search Hub is an optional federation layer. Register after this scenario's
	// own routes exist, in the background, so an unavailable hub never delays or
	// fails Signal Inbox startup. The lifecycle launches this binary from api/.
	go searchregistry.Register(context.Background(), filepath.Join("..", ".vrooli", "search.json"), log.Default())

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
