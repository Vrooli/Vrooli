package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"

	"system-monitor-api/internal/agentmanager"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/handlers"
	"system-monitor-api/internal/infrastructure"
	"system-monitor-api/internal/repository"
	"system-monitor-api/internal/repository/memory"
	sqliterepo "system-monitor-api/internal/repository/sqlite"
	"system-monitor-api/internal/services"
	"system-monitor-api/internal/toolexecution"
	"system-monitor-api/internal/toolhandlers"
	"system-monitor-api/internal/toolregistry"
)

// Run wires dependencies, starts the HTTP server, and blocks until shutdown.
func Run(cfg *config.Config) error {
	setupLogging(cfg)

	closer, repo := connectRepository(cfg)

	_ = services.NewAlertService(cfg, repo) // Alert service available for future wiring //nolint:ineffassign
	monitorSvc := services.NewMonitorService(cfg, repo, infrastructure.NewStaticProvider())

	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName: cfg.AgentManager.ProfileName,
		ProfileKey:  cfg.AgentManager.ProfileKey,
		Timeout:     cfg.AgentManager.Timeout,
		Enabled:     cfg.AgentManager.Enabled,
	})

	if agentSvc.IsEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := agentSvc.Initialize(ctx, agentmanager.DefaultProfileConfig()); err != nil {
			log.Printf("Warning: Failed to initialize agent-manager profile: %v", err)
			log.Println("Investigations require agent-manager; anomaly checks will fail until it is available")
		}
		cancel()
	}

	investigationSvc := services.NewInvestigationService(cfg, repo, monitorSvc, agentSvc)
	scriptSvc := services.NewScriptService(services.ResolveScriptsDir())
	reportSvc := services.NewReportService(cfg, repo)
	settingsMgr := services.NewSettingsManager()
	monitorSvc.SetActive(settingsMgr.IsActive())
	settingsMgr.SetActiveChangedCallback(func(active bool) {
		monitorSvc.SetActive(active)
	})

	if err := monitorSvc.Start(); err != nil {
		return fmt.Errorf("start monitor service: %w", err)
	}

	healthHandler := handlers.NewHealthHandler(cfg, monitorSvc, settingsMgr)
	metricsHandler := handlers.NewMetricsHandler(cfg, monitorSvc)
	investigationHandler := handlers.NewInvestigationHandler(cfg, investigationSvc, scriptSvc)
	reportHandler := handlers.NewReportHandler(cfg, reportSvc)
	settingsHandler := handlers.NewSettingsHandler(settingsMgr)

	// Initialize Tool Discovery Protocol registry
	toolRegistry := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "system-monitor",
		ScenarioVersion:     cfg.Server.Version,
		ScenarioDescription: "Real-time system monitoring with AI-driven anomaly detection and automated root cause analysis",
	})
	toolRegistry.RegisterProvider(toolregistry.NewMetricsToolProvider())
	toolRegistry.RegisterProvider(toolregistry.NewInvestigationToolProvider())
	toolRegistry.RegisterProvider(toolregistry.NewConfigurationToolProvider())
	log.Printf("Registered %d tool providers with %d tools", toolRegistry.ProviderCount(), len(toolRegistry.ListToolNames(context.Background())))

	// Initialize Tool Execution Protocol executor
	toolExecutor := toolexecution.NewServerExecutor(toolexecution.ServerExecutorConfig{
		MonitorSvc:       monitorSvc,
		InvestigationSvc: investigationSvc,
		ReportSvc:        reportSvc,
		SettingsMgr:      settingsMgr,
		Logger:           slog.Default(),
	})
	toolExecHandler := toolexecution.NewHandler(toolExecutor, slog.Default())
	toolsHandler := toolhandlers.NewToolsHandler(toolRegistry, slog.Default())

	router := buildRouter(healthHandler, metricsHandler, investigationHandler, reportHandler, settingsHandler, toolsHandler, toolExecHandler)
	handler := buildMiddleware(cfg, router)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("System Monitor API starting on port %s", cfg.Server.APIPort)
		log.Printf("   Environment: %s", cfg.Server.Environment)
		log.Printf("   Version: %s", cfg.Server.Version)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	waitForShutdown(monitorSvc, srv, closer)
	return nil
}

func connectRepository(cfg *config.Config) (io.Closer, repository.Repository) {
	sqliteRepo, err := connectSQLite()
	if err != nil {
		log.Printf("Warning: Failed to initialize SQLite, using in-memory storage: %v", err)
		return io.NopCloser(nil), memory.NewRepository()
	}

	// Start retention cleanup: hourly, delete metrics older than configured retention.
	maxAge := time.Duration(cfg.Monitoring.RetentionDays) * 24 * time.Hour
	sqliteRepo.StartRetentionCleanup(1*time.Hour, maxAge)

	return sqliteRepo, sqliteRepo
}

// Ensure *sqliterepo.Repository satisfies repository.Repository at compile time.
var _ repository.Repository = (*sqliterepo.Repository)(nil)
