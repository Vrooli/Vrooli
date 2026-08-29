// Vrooli Autoheal API - Self-healing infrastructure supervisor
// [REQ:CLI-TICK-001] [REQ:CLI-STATUS-001]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/bootstrap"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	apiHandlers "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/handlers"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/middleware"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/remediation"
	_ "modernc.org/sqlite"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "vrooli-autoheal",
	}) {
		return // Process was re-exec'd after rebuild
	}

	if err := run(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}

func run() error {
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		return fmt.Errorf("resolve file storage roots: %w", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	// Initialize user configuration manager
	// Config path: ~/.vrooli-autoheal/config.json or VROOLI_AUTOHEAL_CONFIG env var
	configPath := os.Getenv("VROOLI_AUTOHEAL_CONFIG")
	if configPath == "" {
		configPath = userconfig.DefaultConfigPath()
	}
	// Schema path is relative to the binary or in the api directory
	schemaPath := filepath.Join(filepath.Dir(os.Args[0]), "config.schema.json")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		// Try in current working directory
		schemaPath = "config.schema.json"
	}

	configMgr := userconfig.NewManager(configPath, schemaPath)
	if err := configMgr.Load(); err != nil {
		log.Printf("warning: could not load user config: %v (using defaults)", err)
	}
	log.Printf("user config loaded from %s", configMgr.GetConfigPath())

	db, err := connectPersistenceDB(context.Background(), fileRoots)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Printf("persistence backend selected: sqlite")
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return initializeSchema(pool)
	})

	// Initialize database schema through the scenario-owned embedded provider.
	if err := initializeSchema(db); err != nil {
		log.Printf("warning: schema initialization failed: %v (tables may already exist)", err)
	}
	log.Printf("startup stage=schema-ready")

	// Initialize components
	store := persistence.NewStore(db)
	plat := platform.Detect()
	systemEventService := systemevents.NewService(store, plat)
	if raw := os.Getenv("AUTOHEAL_SYSTEMEVENTS_INTERVAL"); raw != "" {
		if d, perr := time.ParseDuration(raw); perr == nil {
			systemEventService.SetIngestInterval(d)
			log.Printf("system-event ingest interval set to %s (from AUTOHEAL_SYSTEMEVENTS_INTERVAL)", d)
		} else {
			log.Printf("warning: invalid AUTOHEAL_SYSTEMEVENTS_INTERVAL %q: %v (using default %s)", raw, perr, systemevents.DefaultIngestInterval)
		}
	}
	registry := checks.NewRegistry(plat)
	if home, err := os.UserHomeDir(); err != nil {
		log.Printf("warning: runtime recovery ownership gate unavailable: %v", err)
	} else {
		registry.SetRecoveryOwnershipGate(checks.RuntimeRecoveryGate{HomeDir: home})
	}

	// Wire config manager into registry for enable/autoHeal checks
	registry.SetConfigProvider(configMgr)
	registry.SetHealTrackerStore(store)

	// Configure auto-heal cooldown/backoff from global config.
	// This is required - no internal hardcoded policy fallback.
	if err := applyAutoHealPolicyFromConfig(registry, configMgr.GetGlobal()); err != nil {
		return fmt.Errorf("invalid auto-heal policy configuration: %w", err)
	}

	// Historical SQLite state can be large and may be contended by the
	// retention worker. It is not part of API readiness: restore it in the
	// background so a healthy supervisor is available within the lifecycle
	// health window.
	startupCtx, cancelStartup := context.WithCancel(context.Background())
	var startupWG sync.WaitGroup
	checksRegistered := make(chan struct{})
	incidentReporterReady := make(chan struct{})
	startupWG.Add(1)
	go func() {
		defer startupWG.Done()
		ctxTrackers, cancelTrackers := context.WithTimeout(startupCtx, 10*time.Second)
		defer cancelTrackers()
		if err := registry.LoadHealTrackers(ctxTrackers); err != nil {
			log.Printf("warning: heal tracker restore failed (continuing with empty in-memory trackers): %v", err)
		} else {
			log.Printf("startup stage=heal-trackers-restored")
			select {
			case <-checksRegistered:
				select {
				case <-incidentReporterReady:
				case <-startupCtx.Done():
					return
				}
				ctxReconcile, cancelReconcile := context.WithTimeout(startupCtx, time.Minute)
				defer cancelReconcile()
				if err := registry.ReconcileHealTrackerDispositions(ctxReconcile); err != nil {
					log.Printf("warning: heal tracker disposition reconciliation failed: %v", err)
				} else {
					log.Printf("startup stage=heal-tracker-dispositions-reconciled")
				}
			case <-startupCtx.Done():
			}
		}
	}()

	// Register health checks using user's monitoring config (delegated to bootstrap module)
	supervisionController, err := bootstrap.RegisterChecksFromConfig(registry, plat, configMgr)
	if err != nil {
		close(checksRegistered)
		return fmt.Errorf("initialize canonical supervision set: %w", err)
	}
	close(checksRegistered)
	log.Printf("startup stage=checks-registered count=%d", registry.GetSummary().TotalCount)

	startupWG.Add(1)
	go func() {
		defer startupWG.Done()
		ctx, cancel := context.WithTimeout(startupCtx, 10*time.Second)
		defer cancel()
		if err := bootstrap.PopulateRecentResults(ctx, registry, store); err != nil {
			log.Printf("warning: could not pre-populate results: %v", err)
		}
		log.Printf("startup stage=recent-results-loaded")
		if _, err := systemEventService.Ingest(ctx); err != nil {
			log.Printf("warning: system event ingestion failed: %v", err)
		}
		log.Printf("startup stage=system-events-loaded")
	}()

	// Stagger the first run of interval checks so aligned intervals don't burst
	// together on every cycle. Checks restored from persistence keep their
	// existing schedule; only cold-start checks are jittered.
	registry.SeedStartupJitter(time.Now(), nil)

	// Schedule initial tick 5 seconds after startup to get fresh results
	bootstrap.ScheduleInitialTick(registry, store, 5*time.Second)

	// Setup HTTP server
	h := apiHandlers.New(registry, store, plat)
	close(incidentReporterReady)
	log.Printf("startup stage=handlers-created")
	if eventsBase, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-events"); resolveErr == nil {
		h.SetIncidentEventPublisher(incidents.NewEventBusPublisher(eventsBase))
	}
	log.Printf("startup stage=event-publisher-resolved")
	if notificationBase, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "notification-hub"); resolveErr == nil {
		if verifier, verifierErr := remediation.NewNotificationHubAskVerifier(notificationBase); verifierErr == nil {
			h.SetRemediationAskVerifier(verifier)
			log.Printf("startup stage=remediation-ask-verifier-resolved")
		} else {
			log.Printf("warning: notification ask verifier unavailable: %v", verifierErr)
		}
	} else {
		log.Printf("warning: notification-hub URL unavailable for remediation ask verification: %v", resolveErr)
	}
	h.SetSystemEventService(systemEventService)
	h.SetHistoryRetentionHoursProvider(func() int { return configMgr.GetGlobal().HistoryRetentionHours })
	configHandlers := apiHandlers.NewConfigHandlers(configMgr, registry)
	configHandlers.SetMonitoringRefresher(supervisionController)
	apiRouter := setupRouter(h, configHandlers)
	apiHandlers.RegisterTypedServices(apiRouter, h)
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/", apiRouter)
	handler := middleware.NewSecurityHeadersMiddleware()(apihttp.TestModeMiddleware(rootMux))

	// Retention is declared in .vrooli/service.json and enforced by the
	// framework. This is the entire integration: no selection rule, no
	// scheduler, and no cleanup loop lives in this scenario.
	retentionManager, retentionDB, err := startRetention(context.Background(), fileRoots)
	if err != nil {
		return fmt.Errorf("start retention: %w", err)
	}
	log.Printf("startup stage=retention-ready")

	log.Printf("starting server | service=vrooli-autoheal-api platform=%s", plat.Platform)

	// Start server with graceful shutdown
	return server.Run(server.Config{
		Handler:      handlers.RecoveryHandler()(handler),
		WriteTimeout: 6 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			cancelStartup()
			startupWG.Wait()
			// Retention stops first and Stop waits for the in-flight cycle, so
			// its connection is idle before either handle closes.
			retentionManager.Stop()
			if err := retentionDB.Close(); err != nil {
				log.Printf("warning: closing retention database handle: %v", err)
			}
			return db.Close()
		},
	})
}

// startRetention builds the retention engine from this scenario's own manifest
// and starts its schedule.
//
// Retention gets its OWN database handle, and the reason is the outage of
// 2026-08-01 rather than a preference about tidiness.
//
// It used to share the serving handle. That handle's pool is capped at one
// connection, so every statement retention issued — and a cycle issues thousands
// — took the only connection the API had. A correct, bounded, batched prune
// therefore presented to the rest of the process as a database that had stopped
// answering: the health probe's 150ms budget expired against the queue rather
// than against any real fault, /health reported the database disconnected, the
// supervisor loop counted three failed ticks and restarted the API, and
// RunOnStart began the cycle again from zero. Nothing was broken; the retention
// cycle simply could never finish, and no scenario was ever started because the
// tick loop never ran again.
//
// The old comment argued a second handle would contend for the write lock. It
// would, and that is the correct trade: SQLite in WAL mode lets readers proceed
// throughout, writer-vs-writer contention resolves in milliseconds under
// busy_timeout because every batch is its own short transaction, and contention
// measured in milliseconds is what we are buying instead of starvation measured
// in minutes. Sharing did not avoid the lock — it converted a lock into a queue
// in front of the entire API.
//
// A BoundBytes cycle is logged as a finding, not swallowed. It means the host is
// producing events faster than the declared 30-day horizon allows — which is
// exactly the condition that let this database reach 453 GB while its retention
// policy ran correctly and deleted nothing.
func startRetention(ctx context.Context, fileRoots *filerouting.RoutedRoots) (*retention.Manager, *database.RoutedDB, error) {
	retentionDB, err := connectPersistenceDB(ctx, fileRoots)
	if err != nil {
		return nil, nil, fmt.Errorf("open retention database handle: %w", err)
	}

	manager, err := retention.NewForScenario(retention.ScenarioConfig{
		ManifestPath: manifestPath(),
		Scenario:     "vrooli-autoheal",
		OpenDatabase: func(string) (retention.Execer, error) { return retentionDB, nil },
		RunOnStart:   true,
		OnFinding: func(f retention.Finding) {
			log.Printf("RETENTION FINDING: %s", f)
		},
	})
	if err != nil {
		_ = retentionDB.Close()
		return nil, nil, err
	}
	for _, b := range manager.Budgets() {
		log.Printf("retention budget declared | name=%s max_age=%s max_bytes=%s", b.Name, b.MaxAge, retention.FormatBytes(b.MaxBytes))
	}
	manager.Start(context.Background())
	return manager, retentionDB, nil
}

// manifestPath locates .vrooli/service.json next to the running binary or in the
// working directory, matching how the schema and config files above are found.
func manifestPath() string {
	candidates := []string{
		filepath.Join(filepath.Dir(os.Args[0]), "..", ".vrooli", "service.json"),
		filepath.Join("..", ".vrooli", "service.json"),
		filepath.Join(".vrooli", "service.json"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	// Empty means "discover by walking up", which is the right fallback when
	// the binary runs from an unexpected location.
	return ""
}

// setupRouter configures HTTP routes
func setupRouter(h *apiHandlers.Handlers, ch *apiHandlers.ConfigHandlers) *mux.Router {
	router := mux.NewRouter()
	router.Use(loggingMiddleware)

	router.HandleFunc("/health", h.Health).Methods("GET")
	router.HandleFunc("/api/v1/health", h.Health).Methods("GET")

	// Platform info
	router.HandleFunc("/api/v1/platform", h.Platform).Methods("GET")

	// Autoheal endpoints
	router.HandleFunc("/api/v1/status", h.Status).Methods("GET")
	router.HandleFunc("/api/v1/tick", h.Tick).Methods("POST")
	router.HandleFunc("/api/v1/checks", h.ListChecks).Methods("GET")
	// Note: /checks/trends must be before /checks/{checkId} to match correctly
	router.HandleFunc("/api/v1/checks/trends", h.CheckTrends).Methods("GET")
	router.HandleFunc("/api/v1/checks/shelves", h.ListCheckShelves).Methods("GET")
	router.HandleFunc("/api/v1/checks/suspended", h.ListSuspendedChecks).Methods("GET")
	router.HandleFunc("/api/v1/checks/{checkId}", h.CheckResult).Methods("GET")
	router.HandleFunc("/api/v1/checks/{checkId}/history", h.CheckHistory).Methods("GET")
	router.HandleFunc("/api/v1/checks/{checkId}/shelve", h.ShelveCheck).Methods("POST")
	router.HandleFunc("/api/v1/checks/{checkId}/shelve", h.UnshelveCheck).Methods("DELETE")
	router.HandleFunc("/api/v1/checks/{checkId}/resume", h.ResumeCheck).Methods("POST")

	// History and timeline endpoints [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
	router.HandleFunc("/api/v1/timeline", h.Timeline).Methods("GET")
	router.HandleFunc("/api/v1/system-events", h.SystemEvents).Methods("GET")
	router.HandleFunc("/api/v1/system-events/refresh", h.RefreshSystemEvents).Methods("POST")
	router.HandleFunc("/api/v1/uptime", h.UptimeStats).Methods("GET")
	router.HandleFunc("/api/v1/uptime/history", h.UptimeHistory).Methods("GET")

	// Host inventory and incident endpoints.
	router.HandleFunc("/api/v1/host/inventory", h.HostInventory).Methods("GET")
	router.HandleFunc("/api/v1/host/inventory/collect", h.CollectHostInventory).Methods("POST")
	router.HandleFunc("/api/v1/host/inventory/changes", h.HostInventoryChanges).Methods("GET")
	router.HandleFunc("/api/v1/incidents", h.Incidents).Methods("GET")
	router.HandleFunc("/api/v1/incidents/latest", h.LatestIncidents).Methods("GET")
	router.HandleFunc("/api/v1/incidents/{incidentId}", h.IncidentDetail).Methods("GET")
	router.HandleFunc("/api/v1/incidents/{incidentId}/observations", h.IncidentObservations).Methods("GET")
	router.HandleFunc("/api/v1/incidents/{incidentId}/remediations", h.IncidentRemediations).Methods("GET")
	router.HandleFunc("/api/v1/incidents/{incidentId}/remediations/{remediationId}/generate", h.GenerateIncidentRemediation).Methods("POST")
	router.HandleFunc("/api/v1/incidents/{incidentId}/remediations/{remediationId}/outcome", h.RecordIncidentRemediationOutcome).Methods("POST")
	router.HandleFunc("/api/v1/incidents/{incidentId}/remediations/{remediationId}/approve", h.ApproveIncidentRemediation).Methods("POST")
	router.HandleFunc("/api/v1/incidents/{incidentId}/{action:acknowledge|resolve|ignore|keep-open}", h.MutateIncidentStatus).Methods("POST")
	router.HandleFunc("/api/v1/transitions", h.Transitions).Methods("GET")

	// Watchdog endpoints [REQ:WATCH-DETECT-001] [REQ:WATCH-INSTALL-001]
	router.HandleFunc("/api/v1/watchdog", h.Watchdog).Methods("GET")
	router.HandleFunc("/api/v1/watchdog/template", h.WatchdogTemplate).Methods("GET")
	router.HandleFunc("/api/v1/watchdog/install", h.WatchdogInstall).Methods("POST")
	router.HandleFunc("/api/v1/watchdog/uninstall", h.WatchdogUninstall).Methods("POST")
	router.HandleFunc("/api/v1/watchdog/linger", h.WatchdogEnableLinger).Methods("POST")
	router.HandleFunc("/api/v1/watchdog/status", h.WatchdogStatus).Methods("GET")

	// Recovery action endpoints [REQ:HEAL-ACTION-001]
	router.HandleFunc("/api/v1/checks/{checkId}/actions", h.GetCheckActions).Methods("GET")
	router.HandleFunc("/api/v1/checks/{checkId}/actions/{actionId}", h.ExecuteCheckAction).Methods("POST")
	router.HandleFunc("/api/v1/actions/history", h.GetActionHistory).Methods("GET")

	// Documentation endpoints
	router.HandleFunc("/api/v1/docs/manifest", h.DocsManifest).Methods("GET")
	router.HandleFunc("/api/v1/docs/content", h.DocsContent).Methods("GET")

	// Configuration endpoints [REQ:CONFIG-*]
	router.HandleFunc("/api/v1/config", ch.GetConfig).Methods("GET")
	router.HandleFunc("/api/v1/config", ch.UpdateConfig).Methods("PUT")
	router.HandleFunc("/api/v1/config/validate", ch.ValidateConfig).Methods("POST")
	router.HandleFunc("/api/v1/config/schema", ch.GetSchema).Methods("GET")
	router.HandleFunc("/api/v1/config/export", ch.ExportConfig).Methods("GET")
	router.HandleFunc("/api/v1/config/import", ch.ImportConfig).Methods("POST")
	router.HandleFunc("/api/v1/config/defaults", ch.GetDefaults).Methods("GET")
	router.HandleFunc("/api/v1/config/global", ch.GetGlobalConfig).Methods("GET")
	router.HandleFunc("/api/v1/config/ui", ch.GetUIConfig).Methods("GET")
	// Per-check config routes - must come after /config/bulk
	router.HandleFunc("/api/v1/config/checks/bulk", ch.BulkUpdateChecks).Methods("PUT")
	router.HandleFunc("/api/v1/config/checks/{checkId}", ch.GetCheckConfig).Methods("GET")
	router.HandleFunc("/api/v1/config/checks/{checkId}/enabled", ch.UpdateCheckEnabled).Methods("PUT")
	router.HandleFunc("/api/v1/config/checks/{checkId}/autoheal", ch.UpdateCheckAutoHeal).Methods("PUT")

	// Monitoring configuration routes (which scenarios/resources to monitor)
	router.HandleFunc("/api/v1/config/monitoring", ch.GetMonitoring).Methods("GET")
	router.HandleFunc("/api/v1/config/monitoring", ch.UpdateMonitoring).Methods("PUT")
	router.HandleFunc("/api/v1/config/monitoring/scenarios", ch.AddScenario).Methods("POST")
	router.HandleFunc("/api/v1/config/monitoring/scenarios/{name}", ch.RemoveScenario).Methods("DELETE")
	router.HandleFunc("/api/v1/config/monitoring/scenarios/{name}/critical", ch.SetScenarioCritical).Methods("PUT")
	router.HandleFunc("/api/v1/config/monitoring/resources", ch.AddResource).Methods("POST")
	router.HandleFunc("/api/v1/config/monitoring/resources/{name}", ch.RemoveResource).Methods("DELETE")

	return router
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// initializeSchema applies the scenario-owned embedded schema. The provider is
// shared with tests so runtime and isolated test databases cannot drift.
func initializeSchema(db interface {
	database.SchemaExecer
	database.SchemaQuerier
},
) error {
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(persistence.Schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	if err := migrateHealthResultStatusContract(db); err != nil {
		return fmt.Errorf("failed to migrate health_results.status: %w", err)
	}

	if err := migrateActionLogsAddTimedOut(db); err != nil {
		return fmt.Errorf("failed to migrate action_logs.timed_out: %w", err)
	}

	log.Printf("database schema initialized successfully")
	return nil
}

// migrateHealthResultStatusContract upgrades the original three-value health
// status constraint so platform-boundary observations can be persisted as
// not-applicable. SQLite cannot alter a CHECK constraint in place, so this is
// a transactional table rebuild that preserves every observation and index.
func migrateHealthResultStatusContract(db interface {
	database.SchemaExecer
	database.SchemaQuerier
},
) error {
	var createSQL string
	rows, err := db.QueryContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type='table' AND name='health_results'`)
	if err != nil {
		return err
	}
	if rows.Next() {
		if err := rows.Scan(&createSQL); err != nil {
			rows.Close()
			return err
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if createSQL == "" {
		return nil
	}
	if strings.Contains(strings.ToLower(createSQL), "not-applicable") {
		return nil
	}
	_, err = db.ExecContext(context.Background(), `
BEGIN;
ALTER TABLE health_results RENAME TO health_results_legacy;
CREATE TABLE health_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    check_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ok', 'warning', 'critical', 'not-applicable')),
    message TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '{}',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
INSERT INTO health_results (id, check_id, status, message, details, duration_ms, created_at)
SELECT id, check_id, status, message, details, duration_ms, created_at FROM health_results_legacy;
DROP TABLE health_results_legacy;
CREATE INDEX IF NOT EXISTS idx_health_results_check_id_created ON health_results (check_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_health_results_created_at ON health_results (created_at DESC);
COMMIT;`)
	if err != nil {
		_, _ = db.ExecContext(context.Background(), "ROLLBACK")
	}
	return err
}

// migrateActionLogsAddTimedOut adds the timed_out column to action_logs for
// databases created before the column existed. Idempotent: a duplicate-column
// error is treated as success.
func migrateActionLogsAddTimedOut(db interface {
	database.SchemaExecer
	database.SchemaQuerier
},
) error {
	rows, err := db.QueryContext(context.Background(), `PRAGMA table_info(action_logs)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasColumn := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "timed_out" {
			hasColumn = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasColumn {
		return nil
	}

	_, err = db.ExecContext(context.Background(), `ALTER TABLE action_logs ADD COLUMN timed_out INTEGER NOT NULL DEFAULT 0`)
	return err
}

func connectPersistenceDB(ctx context.Context, fileRoots *filerouting.RoutedRoots) (*database.RoutedDB, error) {
	dsn, err := resolveSQLiteDSN(ctx, fileRoots)
	if err != nil {
		return nil, err
	}
	db, err := database.Open(ctx, database.Config{
		Driver:       "sqlite",
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read/write performance.
	// WAL allows readers to proceed without blocking writers and vice-versa,
	// eliminating most SQLITE_BUSY errors from concurrent tick operations.
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("warning: failed to enable WAL mode: %v", err)
	}

	// Set busy_timeout so SQLite retries internally on lock contention instead
	// of returning SQLITE_BUSY immediately. 5 seconds is generous enough for
	// the autoheal tick's persistence writes to complete without failing.
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout=5000"); err != nil {
		log.Printf("warning: failed to set busy_timeout: %v", err)
	}

	return db, nil
}

func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("vrooli-autoheal")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve storage namespace: %w", err)
	}
	return storage.EnsureAllDirs(resolver, storage.Options{ScenarioID: scenarioID}, 0)
}

// autohealDatabaseFile is this scenario's database file name. It predates the
// "<scenario>.db" convention and is kept so the existing database is found.
const autohealDatabaseFile = "autoheal.sqlite"

func fileRootPath(ctx context.Context, roots *filerouting.RoutedRoots, class storage.Class, rel string) (string, error) {
	root, err := roots.Pick(ctx, class)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

// resolveSQLiteDSN resolves vrooli-autoheal's own database path.
//
// The routed roots are consulted first so Test Genie can lease this scenario an
// isolated data root for the duration of a run. Outside a run they resolve to
// the same place storage.SQLitePath would.
//
// This scenario used to read SQLITE_PATH and SQLITE_DB. That is why the defect
// existed at all: vrooli-autoheal declared SQLITE_PATH in its own manifest,
// then restarted sick scenarios by exec'ing the CLI, and every child inherited
// the value and opened THIS database instead of its own.
func resolveSQLiteDSN(ctx context.Context, fileRoots *filerouting.RoutedRoots) (string, error) {
	path, err := fileRootPath(ctx, fileRoots, storage.ClassData, autohealDatabaseFile)
	if err != nil {
		return "", fmt.Errorf("resolve sqlite data path: %w", err)
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return "", fmt.Errorf("create sqlite parent directory %q: %w", parent, err)
	}
	return path, nil
}

func applyAutoHealPolicyFromConfig(registry *checks.Registry, global userconfig.GlobalConfig) error {
	policy, err := checks.NewAutoHealPolicyFromGlobal(
		global.RestartCooldownSeconds,
		global.MaxRestartAttempts,
		global.ActionTimeoutFastSeconds,
		global.ActionTimeoutRestartSeconds,
		global.TimeoutRetrySeconds,
	)
	if err != nil {
		return err
	}
	if err := registry.SetHealInterlockWindow(time.Duration(global.HealInterlockSeconds) * time.Second); err != nil {
		return err
	}
	return registry.SetAutoHealPolicy(policy)
}
