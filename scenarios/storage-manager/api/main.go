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
	"storage-manager/internal/orchestrator"
	managerRetention "storage-manager/internal/retention"
	"storage-manager/internal/server"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/artifactledger"
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
	// The lifecycle already owns freshness checks and builds before launching a
	// managed process. Running a second source walk and rebuild here can consume
	// the entire startup health window (and briefly create a competing compiler
	// workload). Keep the direct-run safety net, but make lifecycle startup
	// trust the artifact it just built.
	if preflight.Run(preflight.Config{
		ScenarioName: "storage-manager",
		// The lifecycle build step is the freshness authority. A second source
		// walk here can consume the entire readiness window and duplicate a
		// compiler workload while the managed API is trying to bind.
		DisableStaleness: true,
	}) {
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
	startCensusBackfill(schedulerContext, logger, db, repoRoot)
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
		// Fail closed: retention deletes, so a pass that cannot write receipts
		// must not start at all. Skipping the scheduler is the right blast
		// radius -- the API keeps serving, and the one subsystem that would
		// delete unrecorded stays off until an operator fixes the state
		// directory.
		if retentionLedger, ledgerErr := openRetentionLedger(); ledgerErr != nil {
			logger.Printf("retention scheduler disabled, nothing will be pruned: %v", ledgerErr)
		} else {
			startRetentionScheduler(schedulerContext, logger, repoRoot, retentionLedger, db)
		}
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: logger},
		healthH.Module(db, "storage-manager-api", "1.0.0"),
		cleanupH.ModuleWithContext(schedulerContext, logger, db, fileRoots),
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

// startCensusBackfill keeps the historical report migration out of the API
// readiness path. census_snapshots may contain large legacy report_json blobs;
// with the single-connection SQLite configuration, reading them during main
// can starve handler initialization and make lifecycle health time out. The
// migration is an explicit opt-in maintenance action and is delayed even when
// enabled so the listener is already serving.
func startCensusBackfill(ctx context.Context, logger *log.Logger, db *database.RoutedDB, repoRoot string) {
	if strings.TrimSpace(os.Getenv("STORAGE_CENSUS_BACKFILL")) != "1" || strings.TrimSpace(repoRoot) == "" {
		return
	}
	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		backfillCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		root, err := census.DeviceRoot(repoRoot)
		if err != nil {
			logger.Printf("census sample backfill root resolution failed: %v", err)
			return
		}
		if _, err := census.NewSnapshotStore(db).BackfillEntrySamples(backfillCtx, root, 100); err != nil {
			logger.Printf("census sample backfill failed: %v", err)
		}
	}()
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

// openRetentionLedger resolves the removal ledger retention writes receipts to.
func openRetentionLedger() (*artifactledger.Ledger, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory for retention receipts: %w", err)
	}
	ledger, err := artifactledger.New(home)
	if err != nil {
		return nil, fmt.Errorf("open removal ledger: %w", err)
	}
	return ledger, nil
}

// startRetentionScheduler runs declared storage-entry budgets on an interval.
func startRetentionScheduler(ctx context.Context, logger *log.Logger, repoRoot string, ledger *artifactledger.Ledger, db *database.RoutedDB) {
	platform := storage.Platform(runtime.GOOS)
	budgetCycles := map[string]int{}
	events := eventbus.NewDiscoveredClient(ctx)
	managerRetention.NewScheduler(retentionInterval(), func(cycleCtx context.Context) error {
		inventory, err := storage.LoadOwnerInventory(storage.InventoryOptions{RepoRoot: repoRoot, Platform: platform})
		if err != nil {
			return err
		}
		_, err = (managerRetention.Enforcer{RepoRoot: repoRoot, Platform: platform, Ledger: ledger, RecoveryLockPath: storageRecoveryLockPath(repoRoot), OverBudgetCycles: budgetCycles, BudgetEvent: func(eventCtx context.Context, eventType string, payload map[string]any) error {
			return events.PublishDomainEvent(eventCtx, eventbus.DomainEvent{Source: "storage-manager", EventType: eventType, Payload: payload, Occurred: time.Now().UTC()})
		}}).Enforce(cycleCtx, inventory)
		if err == nil {
			// Ledger rows are derived evidence and have their own retention
			// policy; do not route them through owner filesystem pruning.
			err = orchestrator.NewSQLiteStore(db).PruneRecoveryLedger(cycleCtx, time.Now().UTC())
		}
		if err == nil {
			err = census.NewSnapshotStore(db).Prune(cycleCtx, time.Now().UTC())
		}
		return err
	}).WithObserver(func(cycle sharedscheduler.Cycle) {
		if cycle.Err != nil {
			logger.Printf("retention cycle failed after %s: %v", cycle.Duration.Round(time.Millisecond), cycle.Err)
		}
		if cycle.Overran {
			logger.Printf("retention cycle took %s, at or beyond its %s interval; the next walk waits a full interval", cycle.Duration.Round(time.Second), retentionInterval())
		}
	}).Start(ctx)
}

func storageRecoveryLockPath(repoRoot string) string {
	if resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto}); err == nil {
		if state, resolveErr := resolver.Resolve(storage.Options{}); resolveErr == nil && state.StateDir != "" {
			return filepath.Join(state.StateDir, "recovery.lock")
		}
	}
	return filepath.Join(repoRoot, ".vrooli", "state", "storage-manager", "recovery.lock")
}
