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
	"time"

	"storage-manager/internal/census"
	"storage-manager/internal/httpx"
	"storage-manager/internal/modules"
	managerRetention "storage-manager/internal/retention"
	"storage-manager/internal/server"

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

	advisorH "storage-manager/handlers/advisor"
	cleanupH "storage-manager/handlers/cleanup"
	fleetH "storage-manager/handlers/fleet"
	healthH "storage-manager/handlers/health"
	storageH "storage-manager/handlers/storage"
	validationH "storage-manager/handlers/validation"
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
	scenarioID, err := storage.ScenarioNamespace("storage-manager")
	if err != nil {
		return "", fmt.Errorf("resolve storage-manager storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"storage-manager.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve storage-manager db path: %w", err)
	}
	return sqliteFileDSN(path)
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
	if preflight.Run(preflight.Config{ScenarioName: "storage-manager"}) {
		return
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	httpx.SetLogger(logger)

	dsn, err := sqliteDSN()
	if err != nil {
		logger.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		logger.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		logger.Fatalf("schema initialization failed: %v", err)
	}
	repoRoot, repoErr := repocontract.ResolveRepoRoot()
	if repoErr != nil {
		logger.Printf("repo root resolution failed; validation will report unresolved targets: %v", repoErr)
	}
	primaryPaths, pathsErr := scenarioStoragePaths()
	if pathsErr != nil {
		logger.Fatalf("file storage configuration failed: %v", pathsErr)
	}
	fileRoots := filerouting.New(primaryPaths)

	// Scheduled census is intentionally delayed until the first interval. The
	// API remains fast to ready, while long-lived processes persist immutable
	// observations for infra-health and the operator console.
	schedulerContext, stopScheduler := context.WithCancel(context.Background())
	if repoRoot != "" {
		snapshotStore := census.NewSnapshotStore(db)
		storageScheduler := census.NewScheduler(censusInterval(), func(ctx context.Context) error {
			inventory, err := storage.LoadOwnerInventory(storage.InventoryOptions{RepoRoot: repoRoot, Platform: storage.Platform(runtime.GOOS)})
			if err != nil {
				return err
			}
			report, err := census.ScanInventory(repoRoot, inventory)
			if err != nil {
				return err
			}
			_, err = snapshotStore.Save(ctx, report)
			return err
		})
		storageScheduler.Start(schedulerContext)
		retentionScheduler := managerRetention.NewScheduler(retentionInterval(), func(ctx context.Context) error {
			inventory, err := storage.LoadOwnerInventory(storage.InventoryOptions{RepoRoot: repoRoot, Platform: storage.Platform(runtime.GOOS)})
			if err != nil {
				return err
			}
			_, err = (managerRetention.Enforcer{RepoRoot: repoRoot, Platform: storage.Platform(runtime.GOOS)}).Enforce(ctx, inventory)
			return err
		})
		retentionScheduler.Start(schedulerContext)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: logger},
		healthH.Module(db, "storage-manager-api", "1.0.0"),
		cleanupH.Module(logger, db, fileRoots),
		fleetH.Module(logger, repoRoot, db, schedule.System()),
		advisorH.Module(logger, repoRoot),
		validationH.Module(logger, repoRoot),
		storageH.Module(storageH.ModuleDeps{
			RepoRoot:        repoRoot,
			DB:              db,
			OllamaInventory: storageH.NewOllamaInventoryFromEnvironment(),
			OllamaModelRoot: storageH.DefaultOllamaModelRoot(),
		}),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	apiMountPath := "/"
	rootMux.Handle(apiMountPath, srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// Fleet scans classify every owner and can legitimately take several
		// minutes on a large workspace. The default api-core write timeout is
		// 30 seconds, which closes the socket after the server has completed
		// the scan and leaves clients with an opaque unexpected-EOF error.
		// Keep the long-running RPC within the same bounded window as the CLI.
		WriteTimeout: 20 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			stopScheduler()
			return db.Close()
		},
	}); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

func scenarioStoragePaths() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("storage-manager")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve storage-manager storage namespace: %w", err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: scenarioID})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve storage-manager storage roots: %w", err)
	}
	return paths, nil
}

func retentionInterval() time.Duration {
	const defaultInterval = 15 * time.Minute
	raw := strings.TrimSpace(os.Getenv("STORAGE_RETENTION_INTERVAL"))
	if raw == "" {
		return defaultInterval
	}
	interval, err := time.ParseDuration(raw)
	if err != nil || interval < time.Minute {
		return defaultInterval
	}
	return interval
}

func censusInterval() time.Duration {
	const defaultInterval = 30 * time.Minute
	raw := strings.TrimSpace(os.Getenv("STORAGE_CENSUS_INTERVAL"))
	if raw == "" {
		return defaultInterval
	}
	interval, err := time.ParseDuration(raw)
	if err != nil || interval < time.Minute {
		return defaultInterval
	}
	return interval
}
