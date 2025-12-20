package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"system-monitor-api/internal/agentmanager"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/handlers"
	"system-monitor-api/internal/repository"
	"system-monitor-api/internal/repository/memory"
	"system-monitor-api/internal/services"
)

// Run wires dependencies, starts the HTTP server, and blocks until shutdown.
func Run(cfg *config.Config) error {
	setupLogging(cfg)

	db, repo := connectRepository(cfg)

	alertSvc := services.NewAlertService(cfg, repo)
	monitorSvc := services.NewMonitorService(cfg, repo, alertSvc)

	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		URL:         cfg.AgentManager.URL,
		ProfileName: cfg.AgentManager.ProfileName,
		Timeout:     cfg.AgentManager.Timeout,
		Enabled:     cfg.AgentManager.Enabled,
	})

	if agentSvc.IsEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := agentSvc.Initialize(ctx, agentmanager.DefaultProfileConfig()); err != nil {
			log.Printf("Warning: Failed to initialize agent-manager profile: %v", err)
			log.Println("Will fall back to direct CLI execution for investigations")
		}
		cancel()
	}

	investigationSvc := services.NewInvestigationService(cfg, repo, alertSvc, agentSvc)
	reportSvc := services.NewReportService(cfg, repo)
	settingsMgr := services.NewSettingsManager()

	if err := monitorSvc.Start(); err != nil {
		return fmt.Errorf("start monitor service: %w", err)
	}

	healthHandler := handlers.NewHealthHandler(cfg, monitorSvc)
	metricsHandler := handlers.NewMetricsHandler(cfg, monitorSvc)
	investigationHandler := handlers.NewInvestigationHandler(cfg, investigationSvc)
	reportHandler := handlers.NewReportHandler(cfg, reportSvc)
	settingsHandler := handlers.NewSettingsHandler(settingsMgr)

	router := buildRouter(healthHandler, metricsHandler, investigationHandler, reportHandler, settingsHandler)
	handler := buildMiddleware(cfg, router)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 System Monitor API starting on port %s", cfg.Server.APIPort)
		log.Printf("   Environment: %s", cfg.Server.Environment)
		log.Printf("   Version: %s", cfg.Server.Version)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	waitForShutdown(monitorSvc, srv, db)
	return nil
}

func connectRepository(cfg *config.Config) (*sql.DB, repository.Repository) {
	if cfg.HasDatabase() {
		db, err := connectDatabase(cfg)
		if err != nil {
			log.Printf("Warning: Failed to connect to database, using in-memory storage: %v", err)
			return nil, memory.NewRepository()
		}

		log.Println("Using in-memory repository (PostgreSQL implementation pending)")
		return db, memory.NewRepository()
	}

	log.Println("No database configured, using in-memory storage")
	return nil, memory.NewRepository()
}
