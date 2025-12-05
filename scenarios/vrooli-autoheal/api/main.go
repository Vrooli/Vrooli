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
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"vrooli-autoheal/internal/bootstrap"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/config"
	apiHandlers "vrooli-autoheal/internal/handlers"
	"vrooli-autoheal/internal/persistence"
	"vrooli-autoheal/internal/platform"
	"vrooli-autoheal/internal/userconfig"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `This binary must be run through the Vrooli lifecycle system.

Instead, use:
   vrooli scenario start vrooli-autoheal

The lifecycle system provides environment variables, port allocation,
and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	if err := run(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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

	// Connect to database with retry logic for boot scenarios
	// Database may not be ready immediately after system boot
	db, err := connectWithRetry(cfg.DatabaseURL, 120*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to database after retries: %w", err)
	}
	defer db.Close()

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

	log.Printf("starting server | service=vrooli-autoheal-api port=%s platform=%s", cfg.Port, plat.Platform)

	return startServer(cfg.Port, router)
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

// startServer runs the HTTP server with graceful shutdown
func startServer(port string, router *mux.Router) error {
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      handlers.RecoveryHandler()(router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server startup failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("server stopped")
	return nil
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// connectWithRetry attempts to connect to the database with exponential backoff.
// This is critical for boot scenarios where postgres may not be ready immediately.
func connectWithRetry(databaseURL string, maxWait time.Duration) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	startTime := time.Now()
	backoff := 1 * time.Second
	maxBackoff := 10 * time.Second
	attempt := 0

	for {
		attempt++
		err := db.Ping()
		if err == nil {
			if attempt > 1 {
				log.Printf("database connected after %d attempts (%.1fs)", attempt, time.Since(startTime).Seconds())
			}
			return db, nil
		}

		elapsed := time.Since(startTime)
		if elapsed >= maxWait {
			db.Close()
			return nil, fmt.Errorf("database not available after %v: %w", maxWait, err)
		}

		remaining := maxWait - elapsed
		sleepDuration := backoff
		if sleepDuration > remaining {
			sleepDuration = remaining
		}

		log.Printf("database not ready (attempt %d), retrying in %v... (error: %v)", attempt, sleepDuration, err)
		time.Sleep(sleepDuration)

		// Exponential backoff with cap
		backoff = backoff * 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
