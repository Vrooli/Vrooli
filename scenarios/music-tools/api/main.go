package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"music-tools/internal/capabilities"
	"music-tools/internal/capacity"
	cleanupdomain "music-tools/internal/cleanup"
	"music-tools/internal/jobs"
	"music-tools/internal/librarysync"
	"music-tools/internal/models"
	"music-tools/internal/modules"
	"music-tools/internal/server"
	musicstorage "music-tools/internal/storage"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	analysisH "music-tools/handlers/analysis"
	capsH "music-tools/handlers/capabilities"
	compositionH "music-tools/handlers/composition"
	healthH "music-tools/handlers/health"
	modelsH "music-tools/handlers/models"
	sourcesH "music-tools/handlers/sources"
	transformationH "music-tools/handlers/transformation"
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
	scenarioID, err := storage.ScenarioNamespace("music-tools")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve music-tools storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

// fileRootPath is the template's mandatory file-store seam. Domain stores
// compose their relative paths from it rather than retaining startup root
// strings, so X-Vrooli-Test-Mode is honored independently per request.
func fileRootPath(ctx context.Context, roots *filerouting.RoutedRoots, class storage.Class, rel string) (string, error) {
	root, err := roots.Pick(ctx, class)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "music-tools"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "music-tools",
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
	audioNamespace, err := storage.ScenarioNamespace("music-tools")
	if err != nil {
		log.Fatalf("music blob namespace: %v", err)
	}
	musicBlobs, err := musicstorage.New(audioNamespace)
	if err != nil {
		log.Fatalf("music blob storage configuration failed: %v", err)
	}
	musicBlobs.SetBudgetBytes(32 << 30)
	// The shared model-rights evaluator is an explicit constructor dependency.
	// Scenario Dependency Analyzer admission is currently below the host disk
	// reserve, so nil deliberately keeps model-backed requests fail-closed until
	// the canonical policy adapter can be injected here.
	musicCapacityBroker := capacity.CLICapacityBroker{}
	compositionState, err := compositionH.NewStateWithCapacityStorageAndRightsEvaluatorAndJobSetup(db.Primary(), musicCapacityBroker, musicBlobs, nil, func(manager *jobs.Manager) error {
		return transformationH.RegisterOperations(manager, musicBlobs)
	})
	if err != nil {
		log.Fatalf("composition job manager initialization failed: %v", err)
	}
	modelRegistry, err := models.LoadSeed(os.Getenv("MUSIC_LICENSE_LANE") == "permissive")
	if err != nil {
		log.Fatalf("model registry initialization failed: %v", err)
	}
	librarySync, err := librarysync.New(db.Primary(), musicBlobs, log.Default())
	if err != nil {
		log.Fatalf("music-library outbox initialization failed: %v", err)
	}
	if err := librarySync.Start(context.Background()); err != nil {
		log.Fatalf("music-library outbox start failed: %v", err)
	}
	analysisModule, err := analysisH.Module(compositionState.JobManager, musicBlobs, musicCapacityBroker, librarySync)
	if err != nil {
		log.Fatalf("analysis job registration failed: %v", err)
	}
	sourcesModule, err := sourcesH.Module(db.Primary(), musicBlobs)
	if err != nil {
		log.Fatalf("managed source initialization failed: %v", err)
	}
	transformationModule, err := transformationH.Module(compositionState.JobManager, musicBlobs)
	if err != nil {
		log.Fatalf("transformation job registration failed: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "music-tools-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		compositionH.Module(compositionState),
		analysisModule,
		transformationModule,
		sourcesModule,
		modelsH.Module(modelRegistry),
		cleanupdomain.Module(cleanupdomain.Deps{Store: musicBlobs}),
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
		Handler:        handler,
		LifecycleHooks: lifecycle.Hooks{Quiesce: librarySync.Quiesce, Resume: librarySync.Resume},
		Cleanup:        func(ctx context.Context) error { librarySync.Stop(); compositionState.Close(); return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
