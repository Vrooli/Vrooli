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

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize components
	store := persistence.NewStore(db)
	plat := platform.Detect()
	registry := checks.NewRegistry(plat)

	// Register health checks (delegated to bootstrap module)
	bootstrap.RegisterDefaultChecks(registry, plat)

	// Setup HTTP server
	h := apiHandlers.New(registry, store, plat)
	router := setupRouter(h)

	log.Printf("starting server | service=vrooli-autoheal-api port=%s platform=%s", cfg.Port, plat.Platform)

	return startServer(cfg.Port, router)
}

// setupRouter configures HTTP routes
func setupRouter(h *apiHandlers.Handlers) *mux.Router {
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

	// Watchdog endpoints [REQ:WATCH-DETECT-001]
	router.HandleFunc("/api/v1/watchdog", h.Watchdog).Methods("GET")
	router.HandleFunc("/api/v1/watchdog/template", h.WatchdogTemplate).Methods("GET")

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
