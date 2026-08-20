package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/capabilities"
	internalcoverage "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/coverage"
	internalladder "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/ladder"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/modules"
	internalportability "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/server"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/sources"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	capsH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/capabilities"
	conditionH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/condition"
	coverageH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/coverage"
	focusH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/focus"
	healthH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/health"
	ladderH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/ladder"
	portabilityH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/portability"
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
	scenarioID, err := storage.ScenarioNamespace("infrastructure-manager")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve infrastructure-manager storage namespace: %w", err)
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

// hostOSToken maps the Go runtime's OS name onto the host OS vocabulary the
// capability grid and the ladder both use. "darwin" is the runtime's name for
// the platform every manifest in this repository calls "macos", and joining
// the two axes on different spellings would silently produce a grid where the
// live host matches no column.
func hostOSToken(goos string) string {
	if goos == "darwin" {
		return "macos"
	}
	return goos
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "infrastructure-manager"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "infrastructure-manager",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		log.Fatalf("repository root configuration failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	conditionModule, conditionService := conditionH.ModuleWithService(repoRoot, db, schedule.System())

	// The ladder joins three sources: the device graph published by
	// system-monitor, this scenario's own capability grid, and the platform
	// declarations on vrooli-autoheal's check registry. Each is read under the
	// standard per-source deadline, and any of them being unreachable is
	// reported as a sensor-channel finding rather than as a coverage collapse.
	ladderService := &internalladder.Service{
		Coverage:    internalcoverage.NewService(repoRoot, nil),
		DeviceGraph: sources.DeviceGraphReader{},
		Portability: sources.PortabilityReader{Grid: internalportability.NewService(repoRoot, nil)},
		Checks:      sources.CheckPlatformsReader{},
		HostOS:      hostOSToken(runtime.GOOS),
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "infrastructure-manager-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		conditionModule,
		coverageH.Module(repoRoot),
		focusH.ModuleWithDeps(repoRoot, db, conditionService),
		ladderH.Module(ladderService),
		portabilityH.Module(repoRoot),
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
