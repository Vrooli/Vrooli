package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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
	sharedscheduler "github.com/vrooli/vrooli/packages/scheduler"
	_ "modernc.org/sqlite"

	advisorH "storage-manager/handlers/advisor"
	cleanupH "storage-manager/handlers/cleanup"
	fleetH "storage-manager/handlers/fleet"
	healthH "storage-manager/handlers/health"
	storageH "storage-manager/handlers/storage"
	validationH "storage-manager/handlers/validation"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "storage-manager"}) {
		return
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	httpx.SetLogger(logger)

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "storage-manager",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		logger.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		logger.Fatalf("schema initialization failed: %v", err)
	}
	// Upgrade persisted report blobs into the narrow growth read model without
	// delaying API readiness. The operation is idempotent and resumes safely.
	if repoRoot, resolveErr := repocontract.ResolveRepoRoot(); resolveErr == nil && repoRoot != "" {
		go func() {
			store := census.NewSnapshotStore(db)
			root, rootErr := census.DeviceRoot(repoRoot)
			if rootErr != nil {
				logger.Printf("census sample backfill root resolution failed: %v", rootErr)
				return
			}
			if _, backfillErr := store.BackfillEntrySamples(context.Background(), root, 100); backfillErr != nil {
				logger.Printf("census sample backfill failed: %v", backfillErr)
			}
		}()
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
		}).WithObserver(func(cycle census.Cycle) {
			if cycle.Err != nil {
				logger.Printf("census cycle failed after %s: %v", cycle.Duration.Round(time.Millisecond), cycle.Err)
			}
			if cycle.Overran {
				logger.Printf("census cycle took %s, at or beyond its %s interval; the next walk waits a full interval so the host is not left under a continuous metadata scan",
					cycle.Duration.Round(time.Second), censusInterval())
			}
		})
		storageScheduler.Start(schedulerContext)
		retentionScheduler := managerRetention.NewScheduler(retentionInterval(), func(ctx context.Context) error {
			inventory, err := storage.LoadOwnerInventory(storage.InventoryOptions{RepoRoot: repoRoot, Platform: storage.Platform(runtime.GOOS)})
			if err != nil {
				return err
			}
			_, err = (managerRetention.Enforcer{RepoRoot: repoRoot, Platform: storage.Platform(runtime.GOOS)}).Enforce(ctx, inventory)
			return err
		}).WithObserver(func(cycle sharedscheduler.Cycle) {
			if cycle.Err != nil {
				logger.Printf("retention cycle failed after %s: %v", cycle.Duration.Round(time.Millisecond), cycle.Err)
			}
			if cycle.Overran {
				logger.Printf("retention cycle took %s, at or beyond its %s interval; the next walk waits a full interval", cycle.Duration.Round(time.Second), retentionInterval())
			}
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
