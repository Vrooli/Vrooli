package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"storage-manager/internal/census"
	"storage-manager/internal/dbmaintenance"
	"storage-manager/internal/deletiongrant"
	"storage-manager/internal/httpx"
	"storage-manager/internal/inventorycache"
	"storage-manager/internal/modules"
	"storage-manager/internal/orchestrator"
	"storage-manager/internal/policybridge"
	"storage-manager/internal/providers"
	managerRetention "storage-manager/internal/retention"
	"storage-manager/internal/server"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
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
	// The deletion-grant store is shared by the policy bridge (which projects
	// approved grants as allow rules) and the request/resolution endpoints below.
	// grantRepublisher lets an approval trigger an immediate snapshot refresh so
	// the new allow rule takes effect in seconds rather than at the next interval.
	grantStore := deletiongrant.NewStore(db)
	grantRepublisher := policybridge.NewRepublisher()
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
		snapshotStore := census.NewSnapshotStore(db).WithFreshness(censusInterval())
		storageScheduler := census.NewScheduler(censusInterval(), func(ctx context.Context) error {
			inventory, err := inventorycache.Load(repoRoot, storage.Platform(runtime.GOOS))
			if err != nil {
				return err
			}
			report, err := census.ScanInventory(repoRoot, inventory)
			if err != nil {
				return err
			}
			_, err = snapshotStore.Save(ctx, report)
			return err
		}).WithDutyCycle(censusDutyCycle()).WithObserver(func(cycle census.Cycle) {
			if cycle.Err != nil {
				logger.Printf("census cycle failed after %s: %v", cycle.Duration.Round(time.Millisecond), cycle.Err)
			}
			// One line per pass. The scheduler already spaces passes so the
			// walk stays under its duty-cycle cap; this records what it chose.
			if cycle.Overran {
				logger.Printf("census cycle took %s, at or beyond its %s interval; the next walk starts after %s of idle time to keep census work under %.0f%% of wall time",
					cycle.Duration.Round(time.Second), censusInterval(), cycle.IdleFor.Round(time.Second), censusDutyCycle()*100)
			} else {
				logger.Printf("census cycle took %s; next walk in %s", cycle.Duration.Round(time.Second), cycle.IdleFor.Round(time.Second))
			}
		})
		storageScheduler.Start(schedulerContext)
		// Coding agents decide deletions from the agent-policy bundle; publish
		// where the contract declares deletion safe, owner-confirmed, or durable,
		// plus any location an operator has approved a time-boxed deletion grant
		// for through the decision inbox (projected as an allow rule).
		policybridge.Start(schedulerContext, logger, repoRoot, policyGrantSource{store: grantStore}, grantRepublisher)
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
			QdrantReader:    providers.NewHTTPQdrantGenerationReader(discovery.ResolveScenarioURLDefault, http.DefaultClient, func() string { return os.Getenv("AGENT_MANAGER_SEARCH_CONTROL_TOKEN") }),
		}),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	// Deletion-grant request + resolution endpoints. An agent posts to the
	// request path to raise an operator decision (surfaced in the web-console
	// inbox); on approval, vrooli-events delivers the resolution to the signed
	// webhook, which records the grant so the policy bridge projects it as an
	// allow rule. More specific than apiMountPath, so these win on the mux.
	grantRaiser := operatordecision.Raiser{
		Source:         deletiongrant.Source,
		Kind:           deletiongrant.Kind,
		Store:          grantStore,
		Publisher:      eventbus.NewDiscoveredClient(context.Background()),
		MaxOutstanding: 10,
		Window:         7 * 24 * time.Hour,
	}
	// Verify the requesting coding agent's identity (via the shared
	// agent-provenance layer, which checks the token against agent-manager) so
	// the request records a trusted run id for the automatic answer tie-back.
	// The signed resolution webhook carries no agent token and is left unwrapped.
	grantProvenance := provenance.Middleware(provenance.CLIUtilVerifier{})
	rootMux.Handle(deletiongrant.RequestPath, grantProvenance(deletiongrant.RequestHandler(grantRaiser, declaredProtection(repoRoot))))
	if secret, secretErr := deletiongrant.ResolveWebhookSecret(os.Getenv); deletiongrant.ShouldMountWebhook(secret, secretErr) {
		rootMux.Handle(deletiongrant.WebhookPath, operatordecision.Webhook(secret, deletiongrant.NewResolver(grantStore).WithOnApprove(grantRepublisher.Trigger)))
		if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
			target := "http://127.0.0.1:" + port + deletiongrant.WebhookPath
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := deletiongrant.ReconcileSubscription(ctx, target); err != nil {
					logger.Printf("deletion-grant answer subscription reconcile failed: %v", err)
				}
			}()
		}
	} else {
		logger.Printf("deletion-grant webhook secret unavailable; approvals will not be delivered (set VROOLI_EVENTS_WEBHOOK_SECRET)")
	}

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
	raw := strings.TrimSpace(os.Getenv("STORAGE_CENSUS_INTERVAL"))
	if raw == "" {
		return census.DefaultInterval
	}
	interval, err := time.ParseDuration(raw)
	if err != nil || interval < time.Minute {
		return census.DefaultInterval
	}
	return interval
}

// censusDutyCycle reads STORAGE_CENSUS_DUTY_CYCLE, the maximum fraction of
// wall time the scheduled census may spend walking (0 < value <= 1).
func censusDutyCycle() float64 {
	raw := strings.TrimSpace(os.Getenv("STORAGE_CENSUS_DUTY_CYCLE"))
	if raw == "" {
		return census.DefaultDutyCycle
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 || value > 1 {
		return census.DefaultDutyCycle
	}
	return value
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
	reclaimAttempts := map[string]time.Time{}
	reclaimer := &managerRetention.HTTPOwnerReclaimer{ResolveURL: discovery.ResolveScenarioURLDefault, HTTPClient: http.DefaultClient}
	events := eventbus.NewDiscoveredClient(ctx)
	managerRetention.NewScheduler(retentionInterval(), func(cycleCtx context.Context) error {
		inventory, err := inventorycache.Load(repoRoot, platform)
		if err != nil {
			return err
		}
		var results map[string]managerRetention.Result
		results, err = (managerRetention.Enforcer{
			RepoRoot: repoRoot, Platform: platform, Ledger: ledger, RecoveryLockPath: storageRecoveryLockPath(repoRoot),
			OverBudgetCycles: budgetCycles, OwnerReclaim: reclaimer, OwnerReclaimAttempts: reclaimAttempts,
			BudgetEvent: func(eventCtx context.Context, eventType string, payload map[string]any) error {
				return events.PublishDomainEvent(eventCtx, eventbus.DomainEvent{Source: "storage-manager", EventType: eventType, Payload: payload, Occurred: time.Now().UTC()})
			},
		}).Enforce(cycleCtx, inventory)
		managerRetention.RecordCycle(time.Now().UTC(), results, err)
		// Ledger rows are derived evidence and have their own retention
		// policy; do not route them through owner filesystem pruning. They
		// are pruned even when owner enforcement failed this cycle, because a
		// persistently failing owner must not leave the manager's own tables
		// unbounded.
		store := orchestrator.NewSQLiteStore(db)
		snapshots := census.NewSnapshotStore(db)
		err = errors.Join(err,
			store.PruneRecoveryLedger(cycleCtx, time.Now().UTC()),
			store.PruneAuditTrail(cycleCtx, time.Now().UTC()),
			snapshots.Prune(cycleCtx, time.Now().UTC()),
		)
		maintainOwnDatabase(cycleCtx, logger, db, snapshots)
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

// maintainOwnDatabase returns the pages retention just freed to the
// filesystem. The one-time incremental auto-vacuum conversion waits until the
// census report backlog is compacted, so the rewrite copies the pruned
// database rather than the bloated one and holds the single connection briefly.
func maintainOwnDatabase(ctx context.Context, logger *log.Logger, db *database.RoutedDB, snapshots *census.SnapshotStore) {
	pending, err := snapshots.PendingReportCompaction(ctx)
	if err != nil {
		logger.Printf("database maintenance skipped: %v", err)
		return
	}
	result, err := dbmaintenance.Maintain(ctx, db, dbmaintenance.Options{AllowConversion: pending == 0})
	switch {
	case err != nil:
		logger.Printf("database maintenance failed (auto_vacuum=%s free=%d bytes): %v", result.Before.AutoVacuum, result.Before.FreeBytes, err)
	case result.Converted:
		logger.Printf("database converted to incremental auto-vacuum: %d -> %d pages", result.Before.PageCount, result.After.PageCount)
	case result.ReturnedPages > 0:
		logger.Printf("database returned %d free pages to the filesystem", result.ReturnedPages)
	}
}

// policyGrantSource adapts the deletion-grant store to the policy bridge's
// GrantSource, so approved, unexpired grants project into the agent-policy
// snapshot as allow rules without the bridge depending on the store package.
// declaredProtection reads the host's declarations at request time, so a
// deletion-grant ask shows the operator which paths are owner-protected; when
// they cannot be read the request is refused rather than asked without that.
func declaredProtection(repoRoot string) deletiongrant.Protection {
	if repoRoot == "" {
		return nil
	}
	return func(path string) (string, string, bool, error) {
		rules, _, err := policybridge.Declarations(repoRoot)
		if err != nil {
			return "", "", false, err
		}
		source, root, isRoot := policybridge.ProtectingDeclaration(rules, path)
		return source, root, isRoot, nil
	}
}

type policyGrantSource struct{ store *deletiongrant.Store }

func (p policyGrantSource) ActiveGrantRoots(ctx context.Context, now time.Time) ([]policybridge.GrantRule, error) {
	grants, err := p.store.ActiveGrantRoots(ctx, now)
	if err != nil {
		return nil, err
	}
	out := make([]policybridge.GrantRule, 0, len(grants))
	for _, grant := range grants {
		out = append(out, policybridge.GrantRule{Root: grant.Root, GrantID: grant.GrantID, Approver: grant.Approver, ExpiresAt: grant.ExpiresAt})
	}
	return out, nil
}

func storageRecoveryLockPath(repoRoot string) string {
	if resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto}); err == nil {
		if state, resolveErr := resolver.Resolve(storage.Options{}); resolveErr == nil && state.StateDir != "" {
			return filepath.Join(state.StateDir, "recovery.lock")
		}
	}
	return filepath.Join(repoRoot, ".vrooli", "state", "storage-manager", "recovery.lock")
}
