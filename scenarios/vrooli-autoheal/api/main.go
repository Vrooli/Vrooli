// Vrooli Autoheal API - Self-healing infrastructure supervisor
// [REQ:CLI-TICK-001] [REQ:CLI-STATUS-001]
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"vrooli-autoheal/internal/bootstrap"
	"vrooli-autoheal/internal/checks"
	apiHandlers "vrooli-autoheal/internal/handlers"
	"vrooli-autoheal/internal/persistence"
	"vrooli-autoheal/internal/platform"
	"vrooli-autoheal/internal/userconfig"
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

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverPostgres,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize components
	store := persistence.NewStore(db)
	plat := platform.Detect()
	registry := checks.NewRegistry(plat)

	// Wire config manager into registry for enable/autoHeal checks
	registry.SetConfigProvider(configMgr)

	// Register health checks using user's monitoring config (delegated to bootstrap module)
	bootstrap.RegisterChecksFromConfig(registry, plat, configMgr)

	// Pre-populate registry from database so dashboard shows data immediately
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := bootstrap.PopulateRecentResults(ctx, registry, store); err != nil {
		log.Printf("warning: could not pre-populate results: %v", err)
	}
	cancel()

	// Schedule initial tick 5 seconds after startup to get fresh results
	bootstrap.ScheduleInitialTick(registry, store, 5*time.Second)

	// Setup HTTP server
	h := apiHandlers.New(registry, store, plat)
	configHandlers := apiHandlers.NewConfigHandlers(configMgr, registry)
	router := setupRouter(h, configHandlers)

	log.Printf("starting server | service=vrooli-autoheal-api platform=%s", plat.Platform)

	// Start server with graceful shutdown
	return server.Run(server.Config{
		Handler: handlers.RecoveryHandler()(router),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	})
}

// setupRouter configures HTTP routes
func setupRouter(h *apiHandlers.Handlers, ch *apiHandlers.ConfigHandlers) *mux.Router {
	router := mux.NewRouter()
	router.Use(loggingMiddleware)

	// Health endpoints (required for lifecycle)
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
	router.HandleFunc("/api/v1/checks/{checkId}", h.CheckResult).Methods("GET")
	router.HandleFunc("/api/v1/checks/{checkId}/history", h.CheckHistory).Methods("GET")

	// History and timeline endpoints [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
	router.HandleFunc("/api/v1/timeline", h.Timeline).Methods("GET")
	router.HandleFunc("/api/v1/uptime", h.UptimeStats).Methods("GET")
	router.HandleFunc("/api/v1/uptime/history", h.UptimeHistory).Methods("GET")

	// Incidents endpoint [REQ:PERSIST-HISTORY-001]
	router.HandleFunc("/api/v1/incidents", h.Incidents).Methods("GET")

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

