package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"device-studio/internal/capabilities"
	"device-studio/internal/jobs"
	librarydomain "device-studio/internal/library"
	"device-studio/internal/modules"
	photoowner "device-studio/internal/photographic"
	"device-studio/internal/physicalruntime"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	capsH "device-studio/handlers/capabilities"
	creationH "device-studio/handlers/creation"
	devicesH "device-studio/handlers/devices"
	healthH "device-studio/handlers/health"
	libraryH "device-studio/handlers/library"
	motionsH "device-studio/handlers/motions"
	photographicH "device-studio/handlers/photographic"
	physicalruntimeH "device-studio/handlers/physicalruntime"
	projectsH "device-studio/handlers/projects"
	renderjobsH "device-studio/handlers/renderjobs"
	scenesH "device-studio/handlers/scenes"
	shotsH "device-studio/handlers/shots"
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
	scenarioID, err := storage.ScenarioNamespace("device-studio")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve device-studio storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

// dispatchOwnedWorker routes exact private worker subcommands before normal
// lifecycle preflight. API invocations continue through preflight unchanged.
func dispatchOwnedWorker(args []string, render, inspect, video func([]string) error, physicalWorker ...func([]string) error) (bool, error) {
	if rendering.IsWorkerInvocation(args) {
		return true, render(args)
	}
	if rendering.IsInspectionWorkerInvocation(args) {
		return true, inspect(args)
	}
	if rendering.IsVideoWorkerInvocation(args) {
		return true, video(args)
	}
	if rendering.IsPhysicalWorkerInvocation(args) {
		if len(physicalWorker) != 1 || physicalWorker[0] == nil {
			return true, fmt.Errorf("physical worker dispatch is not configured")
		}
		return true, physicalWorker[0](args)
	}
	if rendering.IsGeometryWorkerInvocation(args) {
		return true, rendering.RunGeometryWorkerCommand(args)
	}
	if rendering.IsGeometryCompileWorkerInvocation(args) {
		return true, rendering.RunGeometryCompileWorkerCommand(args)
	}
	return false, nil
}

func main() {
	// Owned workers run the pinned binary directly and must not enter
	// development rebuild preflight or initialize the API database/listener.
	if handled, err := dispatchOwnedWorker(os.Args[1:], rendering.RunWorkerCommand, rendering.RunInspectionWorkerCommand, rendering.RunVideoWorkerCommand, rendering.RunPhysicalWorkerCommand); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "device-studio"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "device-studio",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	configureTestPoolSchemas(db)
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	mediaStore, err := routedblobstore.New(fileRoots, "media")
	if err != nil {
		log.Fatalf("project media storage failed: %v", err)
	}
	mediaService := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: projectRepo.(projects.SourceMediaRepository), Store: mediaStore})
	physicalOwners, err := physicalruntime.New(db, fileRoots, mediaService)
	if err != nil {
		log.Fatalf("physical candidate owners failed: %v", err)
	}
	if _, err := physicalOwners.InitializeReviewedStarter(context.Background()); err != nil {
		log.Fatalf("reviewed physical starter initialization failed: %v", err)
	}

	libraryBlobs, err := routedblobstore.New(fileRoots, "library")
	if err != nil {
		log.Fatalf("library intake blob storage failed: %v", err)
	}
	intakeOwner, err := librarydomain.NewIntakeService(db, libraryBlobs, inspectIntakeGLB, nil, nil)
	if err != nil {
		log.Fatalf("library intake owner failed: %v", err)
	}
	renderModule, renderPump, renderCoordinator, err := renderjobsH.BuildWithCoordinatorAndCreationOwners(db, fileRoots, log.Default(), physicalOwners.Shots, physicalOwners.Library, rendering.ResolvePhysicalBlenderProfile, rendering.PhysicalAdapterBytes, intakeOwner)
	if err != nil {
		log.Fatalf("render jobs initialization failed: %v", err)
	}
	qualificationOwner, err := physicalruntime.NewQualificationService(physicalOwners, renderCoordinator)
	if err != nil {
		log.Fatalf("physical qualification review owner failed: %v", err)
	}
	physicalOwners.Qualification = qualificationOwner
	photoClient := photoowner.NewClient(discovery.ResolveScenarioURLDefault, nil)
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "device-studio-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		projectsH.ModuleWithRootsAndInspection(db, fileRoots, renderCoordinator, renderPump, log.Default()),
		photographicH.Module(projectService, photoClient),
		devicesH.ModuleWithService(physicalOwners.Devices, qualificationOwner),
		scenesH.ModuleWithService(physicalOwners.Scenes),
		motionsH.ModuleWithService(physicalOwners.Motions),
		libraryH.ModuleWithIntake(physicalOwners.LibraryCatalog, intakeOwner, db, fileRoots),
		creationH.ModuleWithCoordinator(renderCoordinator, jobs.NewService(jobs.NewSQLiteRepository(db), nil), renderPump),
		physicalruntimeH.Module(physicalOwners),
		shotsH.ModuleWithPhysicalOwners(db, fileRoots, renderCoordinator, physicalOwners.Shots, physicalOwners.Presets, log.Default()),
		renderModule,
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

	renderPump.Start(context.Background())
	if err := apiserver.Run(apiserver.Config{
		Handler:        handler,
		Authentication: sharedAuth,
		Cleanup: func(ctx context.Context) error {
			if err := renderPump.Stop(ctx); err != nil {
				return fmt.Errorf("stop render progress before database close: %w", err)
			}
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func inspectIntakeGLB(ctx context.Context, data []byte) (librarydomain.IntakeInspection, error) {
	if err := ctx.Err(); err != nil {
		return librarydomain.IntakeInspection{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return librarydomain.IntakeInspection{}, err
	}
	return librarydomain.IntakeInspection{
		Profile: "device-studio-physical-glb-inventory-v1", ProfileVersion: 1, Supported: true,
		NodeCount: uint32(len(inventory.Nodes)), MeshCount: uint32(len(inventory.Meshes)), TriangleCount: inventory.TriangleCount,
	}, nil
}

// configureTestPoolSchemas gives each Test Genie lease the same domain-owned
// schema providers as the primary pool. Leased jobs otherwise fail on the
// first public render operation even though primary startup is healthy.
func configureTestPoolSchemas(db *database.RoutedDB) {
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(ctx, pool, modules.AllSchemas()...)
	})
}
