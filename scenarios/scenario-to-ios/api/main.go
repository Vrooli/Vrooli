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

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The path scope is the variant-aware namespace (storage.ScenarioNamespace),
// not the bare slug: under a Baseline Modes shadow engagement the lifecycle
// injects VROOLI_STORAGE_NAMESPACE, so the shadow's SQLite file lands beside
// "<scenario>_shadow" and never shares live's database. Outside the lifecycle
// (local `go run`, tests) it falls back to the compile-time slug, so live paths
// are unchanged. This is why a generated scenario is shadow-safe with zero
// per-scenario work — see packages/api-core/storage/namespace.go.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("scenario-to-ios")
	if err != nil {
		return "", fmt.Errorf("resolve scenario-to-ios storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"scenario-to-ios.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve scenario-to-ios db path: %w", err)
	}
	return sqliteFileDSN(path)
}

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

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "scenario-to-ios"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
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
	matrixService := validationmatrix.NewService(
		matrixStore,
		validationmatrix.Executors{Local: matrixExecutor, Bridge: matrixExecutor},
		validationmatrix.WithCatalogResolver(releases.Catalog{Probe: prober, Journey: journeySelection}),
	)
	matrixService.RecoverStale()
	matrixHandler := validationmatrix.NewHandler(matrixService)
	readinessProbe := readiness.Probe{
		DeveloperProgram: envBool("APPLE_DEVELOPER_PROGRAM"),
		VerifiedIdentity: envBool("APPLE_VERIFIED_IDENTITY"),
		MacOSBuildHost:   envBool("APPLE_MACOS_BUILD_HOST"),
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
		targetsH.Module(prober),
		journeysH.Module(),
		readinessH.Module(readinessProbe),
		distributionH.Module(distributor),
		releasesH.Module([]*validationmatrix.Handler{matrixHandler}, releasesH.Surface{
			Probe: func(ctx context.Context) (deliveryramp.Inventory, error) {
				return prober.Probe(ctx, deliveryramp.ProbeRequest{})
			},
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
