package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/agentmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
	sqliterepo "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/sqlite"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/autoheal"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/forensics"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/journal"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/toolexecution"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/toolhandlers"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/toolregistry"
)

// shellExec is a minimal CommandExecutor wrapping exec.CommandContext for
// the journal reader and forensics service.
type shellExec struct{}

func (shellExec) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

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
			slog.Warn("Failed to initialize agent-manager profile", "error", err)
			slog.Info("Investigations require agent-manager; anomaly checks will fail until it is available")
		}
		cancel()
	}

	apiLog := slog.Default()

	investigationSvc := services.NewInvestigationService(cfg, repo, monitorSvc, agentSvc, services.WithLogger(apiLog.With("service", "investigation")))
	scriptSvc := services.NewScriptService(services.ResolveScriptsDir())
	reportSvc := services.NewReportService(cfg, repo)
	settingsMgr := services.NewSettingsManager()
	monitorSvc.SetActive(settingsMgr.IsActive())
	monitorSvc.ApplySettings(settingsMgr.GetSettings())
	settingsMgr.SetActiveChangedCallback(func(active bool) {
		monitorSvc.SetActive(active)
	})
	settingsMgr.SetSettingsChangedCallback(func(next services.Settings) {
		monitorSvc.ApplySettings(next)
	})

	if err := monitorSvc.Start(); err != nil {
		return fmt.Errorf("start monitor service: %w", err)
	}

	// Metrics lifecycle: maintenance service + settings-driven retention scheduler.
	maintenanceSvc := services.NewMetricsMaintenanceService(repo)
	retentionScheduler := services.NewRetentionScheduler(maintenanceSvc, settingsMgr, apiLog.With("service", "retention"))
	retentionScheduler.Start()

	healthHandler := handlers.NewHealthHandler(cfg, monitorSvc, settingsMgr)
	metricsHandler := handlers.NewMetricsHandler(cfg, monitorSvc, apiLog.With("handler", "metrics"))
	investigationHandler := handlers.NewInvestigationHandler(cfg, investigationSvc, scriptSvc, apiLog.With("handler", "investigations"))
	reportHandler := handlers.NewReportHandler(cfg, reportSvc, apiLog.With("handler", "reports"))
	settingsHandler := handlers.NewSettingsHandler(settingsMgr, apiLog.With("handler", "settings"))
	maintenanceHandler := handlers.NewMaintenanceHandler(maintenanceSvc, apiLog.With("handler", "maintenance"))
	capacityHandler := handlers.NewCapacityHandler(services.NewCapacityService(), apiLog.With("handler", "capacity"))

	// Forensics + logs wiring.
	executor := shellExec{}
	journalReader := journal.NewReader(executor)
	forensicsSvc := forensics.NewService(journalReader, executor, forensics.DefaultFileSystem(), time.Now)
	autohealClient := autoheal.NewClient(autoheal.Config{})
	forensicsHandler := handlers.NewForensicsHandler(forensicsSvc, autohealClient, apiLog.With("handler", "forensics"))
	logsHandler := handlers.NewLogsHandler(journalReader, apiLog.With("handler", "logs"))

	// Initialize Tool Discovery Protocol registry
	toolRegistry := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "system-monitor",
		ScenarioVersion:     cfg.Server.Version,
		ScenarioDescription: "Real-time system monitoring with AI-driven anomaly detection and automated root cause analysis",
	})
	toolRegistry.RegisterProvider(toolregistry.NewMetricsToolProvider())
	toolRegistry.RegisterProvider(toolregistry.NewInvestigationToolProvider())
	toolRegistry.RegisterProvider(toolregistry.NewConfigurationToolProvider())
	slog.Info("Tool registry initialized", "providers", toolRegistry.ProviderCount(), "tools", len(toolRegistry.ListToolNames(context.Background())))

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

	router := buildRouter(healthHandler, metricsHandler, investigationHandler, reportHandler, settingsHandler, maintenanceHandler, capacityHandler, forensicsHandler, logsHandler, toolsHandler, toolExecHandler)
	handler := buildMiddleware(cfg, router)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("System Monitor API starting",
			"port", cfg.Server.APIPort,
			"environment", cfg.Server.Environment,
			"version", cfg.Server.Version,
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	waitForShutdown(monitorSvc, investigationSvc, retentionScheduler, srv, closer)
	return nil
}

func connectRepository(_ *config.Config) (io.Closer, repository.Repository) {
	sqliteRepo, err := connectSQLite()
	if err != nil {
		slog.Warn("Failed to initialize SQLite, using in-memory storage", "error", err)
		return io.NopCloser(nil), memory.NewRepository()
	}

	return sqliteRepo, sqliteRepo
}

// Ensure *sqliterepo.Repository satisfies repository.Repository at compile time.
var _ repository.Repository = (*sqliterepo.Repository)(nil)
