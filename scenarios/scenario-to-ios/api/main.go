package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"scenario-to-ios/internal/builds"
	"scenario-to-ios/internal/capabilities"
	"scenario-to-ios/internal/distribution"
	"scenario-to-ios/internal/journeys"
	"scenario-to-ios/internal/modules"
	"scenario-to-ios/internal/readiness"
	"scenario-to-ios/internal/releases"
	"scenario-to-ios/internal/server"
	"scenario-to-ios/internal/targets"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	_ "modernc.org/sqlite"

	buildsH "scenario-to-ios/handlers/builds"
	capsH "scenario-to-ios/handlers/capabilities"
	distributionH "scenario-to-ios/handlers/distribution"
	healthH "scenario-to-ios/handlers/health"
	journeysH "scenario-to-ios/handlers/journeys"
	readinessH "scenario-to-ios/handlers/readiness"
	releasesH "scenario-to-ios/handlers/releases"
	targetsH "scenario-to-ios/handlers/targets"
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
	scenarioID, err := storage.ScenarioNamespace("scenario-to-ios")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve scenario-to-ios storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "scenario-to-ios"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "scenario-to-ios",
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
	matrixStore, err := validationmatrix.NewFileStore(filepath.Join(primaryFileRoots.DataDir, "validation-matrices"))
	if err != nil {
		log.Fatalf("validation matrix storage configuration failed: %v", err)
	}
	prober := targets.Prober{GOOS: runtime.GOOS}
	// No Apple toolchain runs on Linux, so a macOS bridge node is this ramp's
	// normal execution path rather than a fallback. Without a bridge source the
	// inventory can only ever answer "no registered macOS bridge node",
	// regardless of what the fleet actually holds.
	bridgeClient := validationmatrix.NewClient(resolveBridgeURL(), os.Getenv("VROOLI_BRIDGE_API_TOKEN"), nil, validationmatrix.WithPlatform("ios"))
	var bridgeSources []deliveryramp.BridgeSource
	if bridgeClient != nil {
		bridgeSources = append(bridgeSources, bridgeClient)
	}
	discoverTargets := func(ctx context.Context) (deliveryramp.Inventory, error) {
		return deliveryramp.Discover(ctx, prober, bridgeSources...)
	}
	builder := builds.Builder{GOOS: runtime.GOOS}
	iosPlan := journeys.Plan()
	journeySelection := validationmatrix.JourneySelection{
		JourneyID:     iosPlan.ID,
		DisplayName:   "iOS generated-app conformance",
		SourcePath:    "internal/journeys/plan.go",
		ExecutionMode: "platform",
		Required:      true,
		Category:      "ios",
		Requirements:  []string{"macOS simulator or iOS bridge", "device-control lease", "BAS WebView flow"},
		Safety:        validationmatrix.JourneySafety{Mutating: true, RequiresIsolation: true, RequiresConfirmation: true},
	}
	journeyRunner := func(ctx context.Context, request deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error) {
		return (journeys.Driver{GOOS: runtime.GOOS}).Execute(ctx, request)
	}
	matrixExecutor := releases.Executor{JourneyPlan: iosPlan, RunJourney: journeyRunner}
	// The bridge executor must not be the local one. A Linux host cannot build
	// or boot an iOS simulator, so running a "bridge" cell locally could only
	// fail or, worse, attribute a local result to a remote target.
	executors := validationmatrix.Executors{Local: matrixExecutor}
	if bridgeClient != nil {
		executors.Bridge = bridgeClient
	}
	matrixService := validationmatrix.NewService(
		matrixStore,
		executors,
		validationmatrix.WithCatalogResolver(releases.Catalog{Probe: prober, Journey: journeySelection}),
	)
	matrixService.RecoverStale()
	matrixHandler := validationmatrix.NewHandler(matrixService)
	readinessProbe := readiness.Probe{
		DeveloperProgram: envBool("APPLE_DEVELOPER_PROGRAM"),
		VerifiedIdentity: envBool("APPLE_VERIFIED_IDENTITY"),
		// The build-host rung is derived from live fleet state rather than an
		// environment flag. A remembered flag can claim a macOS host that is not
		// there, which is exactly what the ladder exists to prevent.
		ObserveBuildHost: readiness.BuildHostObserver(discoverTargets),
		SigningReference: envBool("APPLE_SIGNING_REFERENCE"),
		TestFlightAccess: envBool("APPLE_TESTFLIGHT_ACCESS"),
		AppStoreListing:  envBool("APPLE_APP_STORE_LISTING"),
	}
	distributor := distribution.Distributor{
		DeveloperProgram: readinessProbe.DeveloperProgram,
		SigningReference: readinessProbe.SigningReference,
		TestFlightAccess: readinessProbe.TestFlightAccess,
		AppStoreListing:  readinessProbe.AppStoreListing,
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "scenario-to-ios-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		buildsH.Module(builder),
		targetsH.Module(prober, bridgeSources...),
		journeysH.Module(),
		readinessH.Module(readinessProbe),
		distributionH.Module(distributor),
		releasesH.Module([]*validationmatrix.Handler{matrixHandler}, releasesH.Surface{
			Probe:        discoverTargets,
			ChapterCount: len(iosPlan.Steps),
		}),
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

func envBool(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes"
}

// resolveBridgeURL locates vrooli-bridge, preferring an explicit override and
// falling back to scenario discovery.
//
// Because no Apple toolchain runs on Linux, a macOS bridge node is this ramp's
// normal execution path. Requiring an environment variable to reach it would let
// an unset value silently disable remote execution, and the inventory would then
// report "no registered macOS bridge node" while a healthy fleet ran beside it.
func resolveBridgeURL() string {
	if configured := strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_URL")); configured != "" {
		return configured
	}
	resolved, err := discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-bridge")
	if err != nil {
		log.Printf("vrooli-bridge discovery failed; iOS bridge targets will report unavailable: %v", err)
		return ""
	}
	return resolved
}
