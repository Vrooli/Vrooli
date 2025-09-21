package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/handlers"
	"system-monitor-api/internal/middleware"
	"system-monitor-api/internal/repository"
	"system-monitor-api/internal/services"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start system-monitor

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Load configuration
	cfg := config.Load()
	
	// Setup logging
	setupLogging(cfg)
	
	// Connect to database if configured
	var db *sql.DB
	var repo repository.Repository
	
	if cfg.HasDatabase() {
		var err error
		db, err = connectDatabase(cfg)
		if err != nil {
			log.Printf("Warning: Failed to connect to database, using in-memory storage: %v", err)
			repo = repository.NewMemoryRepository()
		} else {
			// TODO: Implement PostgreSQL repository
			// For now, use memory repository even with database
			repo = repository.NewMemoryRepository()
			log.Println("Using in-memory repository (PostgreSQL implementation pending)")
		}
	} else {
		log.Println("No database configured, using in-memory storage")
		repo = repository.NewMemoryRepository()
	}
	
	// Create services
	alertSvc := services.NewAlertService(cfg, repo)
	monitorSvc := services.NewMonitorService(cfg, repo, alertSvc)
	investigationSvc := services.NewInvestigationService(cfg, repo, alertSvc)
	reportSvc := services.NewReportService(cfg, repo)
	
	// Start monitoring service
	if err := monitorSvc.Start(); err != nil {
		log.Fatalf("Failed to start monitor service: %v", err)
	}
	
	// Create handlers
	healthHandler := handlers.NewHealthHandler(cfg, monitorSvc)
	metricsHandler := handlers.NewMetricsHandler(cfg, monitorSvc)
	investigationHandler := handlers.NewInvestigationHandler(cfg, investigationSvc)
	reportHandler := handlers.NewReportHandler(cfg, reportSvc)
	
	// Setup routes
	router := setupRoutes(cfg, healthHandler, metricsHandler, investigationHandler, reportHandler)
	
	// Setup middleware
	handler := setupMiddleware(cfg, router)
	
	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in goroutine
	go func() {
		log.Printf("🚀 System Monitor API starting on port %s", cfg.Server.APIPort)
		log.Printf("   Environment: %s", cfg.Server.Environment)
		log.Printf("   Version: %s", cfg.Server.Version)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	waitForShutdown(monitorSvc, srv, db)
}

func setupLogging(cfg *config.Config) {
	// Configure log output
	if cfg.Logging.Output == "file" {
		file, err := os.OpenFile(cfg.Logging.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file, using stdout: %v", err)
		} else {
			log.SetOutput(file)
		}
	}
	
	// Set log format
	if cfg.Logging.Format == "json" {
		// Would configure JSON logging here
		log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	} else {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	}
}

func connectDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	
	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConnections)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	
	// Test connection with retries
	maxRetries := 10
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.Ping()
		if err == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			return db, nil
		}
		
		waitTime := time.Duration(attempt+1) * time.Second
		log.Printf("⚠️  Database connection attempt %d/%d failed: %v", attempt+1, maxRetries, err)
		log.Printf("⏳ Waiting %v before next attempt", waitTime)
		time.Sleep(waitTime)
	}
	
	return nil, err
}

func setupRoutes(cfg *config.Config, health *handlers.HealthHandler, metrics *handlers.MetricsHandler, investigation *handlers.InvestigationHandler, report *handlers.ReportHandler) *mux.Router {
	r := mux.NewRouter()
	
	// Health endpoint
	r.HandleFunc("/health", health.Handle).Methods("GET")
	
	// Metrics endpoints
	r.HandleFunc("/api/metrics/current", metrics.GetCurrentMetrics).Methods("GET")
	r.HandleFunc("/api/metrics/detailed", metrics.GetDetailedMetrics).Methods("GET")
	r.HandleFunc("/api/metrics/processes", metrics.GetProcessMonitor).Methods("GET")
	r.HandleFunc("/api/metrics/infrastructure", metrics.GetInfrastructureMonitor).Methods("GET")
	
	// Investigation endpoints
	r.HandleFunc("/api/investigations/latest", investigation.GetLatestInvestigation).Methods("GET")
	r.HandleFunc("/api/investigations/trigger", investigation.TriggerInvestigation).Methods("POST")
	r.HandleFunc("/api/investigations/cooldown", investigation.GetCooldownStatus).Methods("GET")
	r.HandleFunc("/api/investigations/cooldown/reset", investigation.ResetCooldown).Methods("POST")
	r.HandleFunc("/api/investigations/cooldown/period", investigation.UpdateCooldownPeriod).Methods("PUT")
	r.HandleFunc("/api/investigations/triggers", investigation.GetTriggers).Methods("GET")
	r.HandleFunc("/api/investigations/triggers/{id}", investigation.UpdateTrigger).Methods("PUT")
	r.HandleFunc("/api/investigations/triggers/{id}/threshold", investigation.UpdateTriggerThreshold).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}", investigation.GetInvestigation).Methods("GET")
	r.HandleFunc("/api/investigations/{id}/status", investigation.UpdateInvestigationStatus).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/findings", investigation.UpdateInvestigationFindings).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/progress", investigation.UpdateInvestigationProgress).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/step", investigation.AddInvestigationStep).Methods("POST")
	
	// Report endpoints (order matters - more specific routes first)
	r.HandleFunc("/api/reports/generate", report.GenerateReport).Methods("POST")
	r.HandleFunc("/api/reports/{id}", report.GetReport).Methods("GET")
	r.HandleFunc("/api/reports", report.ListReports).Methods("GET")
	
	// Legacy endpoints for backward compatibility
	r.HandleFunc("/api/logs", legacyLogsHandler).Methods("GET")
	
	return r
}

func setupMiddleware(cfg *config.Config, router *mux.Router) http.Handler {
	// Create middleware chain
	handler := http.Handler(router)
	
	// Add CORS middleware
	handler = middleware.CORS(handler)
	
	// Add logging middleware
	logger := log.New(os.Stdout, "[HTTP] ", log.LstdFlags)
	handler = middleware.Logging(logger)(handler)
	
	// Add auth middleware if configured
	if cfg.Alerts.EnableWebhooks {
		// Could add API key auth here
	}
	
	return handler
}

func waitForShutdown(monitorSvc *services.MonitorService, srv *http.Server, db *sql.DB) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	// Stop monitoring service
	monitorSvc.Stop()
	
	// Shutdown HTTP server
	if err := srv.Close(); err != nil {
		log.Printf("Error closing server: %v", err)
	}
	
	// Close database connection
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
	
	log.Println("Server shutdown complete")
}

// Legacy handlers for backward compatibility

func legacyLogsHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"logs": []map[string]interface{}{},
		"count": 0,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}