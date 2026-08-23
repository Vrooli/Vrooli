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

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/agentmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
	sqliterepo "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/sqlite"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/autoheal"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/cleanupmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/forensics"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/journal"
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

	closer, repo, routedDB, fileRoots, err := connectRepository(cfg)
	if err != nil {
		return fmt.Errorf("initialize metrics storage: %w", err)
	}

	alertSvc := services.NewAlertService(cfg, repo)
	monitorSvc := services.NewMonitorService(cfg, repo, infrastructure.NewStaticProvider())

	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName: cfg.AgentManager.ProfileName,
		ProfileKey:  cfg.AgentManager.ProfileKey,
		Timeout:     cfg.AgentManager.Timeout,
		Enabled:     cfg.AgentManager.Enabled,
	})

	if agentSvc.IsEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := agentSvc.Initialize(ctx); err != nil {
			slog.Warn("Failed to initialize agent-manager profile", "error", err)
			slog.Info("Investigations require agent-manager; anomaly checks will fail until it is available")
		}
		cancel()
	}

	apiLog := slog.Default()

	investigationSvc := services.NewInvestigationService(cfg, repo, monitorSvc, agentSvc, services.WithLogger(apiLog.With("service", "investigation")))
	scriptSvc := services.NewScriptService(services.ResolveScriptsDir())
	scriptSvc.SetNativeRunner(services.NewNativeInvestigator())
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
	// The scheduler also drives per-process raw-then-rollup retention (additive
	// to the metrics-blob retention) on the same cadence.
	maintenanceSvc := services.NewMetricsMaintenanceService(repo)
	retentionScheduler := services.NewRetentionScheduler(maintenanceSvc, settingsMgr, apiLog.With("service", "retention")).
		WithProcessRetention(repo, cfg.Monitoring.RawRetention, cfg.Monitoring.RollupRetention)
	retentionScheduler.Start()

	// Disk-pressure lifecycle: evaluate the configured threshold on a schedule
	// and persist a violation when it is crossed. Without this loop the
	// threshold setting, the violation model, and the alert repository are all
	// reachable only from tests.
	thresholdScheduler := services.NewThresholdScheduler(settingsMgr, alertSvc, repo, apiLog.With("service", "threshold"),
		services.WithPressureReporter(cleanupmanager.NewClient(cleanupmanager.Config{})),
		services.WithCPUObservationSource(monitorSvc))
	thresholdScheduler.Start()

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
	diskPressureHandler := handlers.NewDiskPressureHandler(thresholdScheduler, repo, apiLog.With("handler", "disk-pressure"))

	// The device-graph read verb serves from the monitor service's shared
	// cached provider, the same one the 60s collector uses.
	deviceGraphHandler := handlers.NewDeviceGraphHandler(monitorSvc.DeviceGraphs(), apiLog.With("handler", "device-graph"))

	router := buildRouter(cfg, healthHandler, metricsHandler, investigationHandler, reportHandler, settingsHandler, maintenanceHandler, capacityHandler, forensicsHandler, logsHandler, diskPressureHandler, deviceGraphHandler)
	rootMux := http.NewServeMux()
	if routedDB != nil && fileRoots != nil {
		devrouting.RegisterWithFileRoots(rootMux, routedDB, fileRoots)
	}
	rootMux.Handle("/", router)
	handler := apihttp.TestModeMiddleware(buildMiddleware(cfg, rootMux))

	srv := &http.Server{
		Addr:         ":" + cfg.Server.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: serverWriteTimeout(cfg),
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

	waitForShutdown(monitorSvc, investigationSvc, retentionScheduler, thresholdScheduler, srv, closer)
	return nil
}

func connectRepository(cfg *config.Config) (io.Closer, repository.Repository, *database.RoutedDB, *filerouting.RoutedRoots, error) {
	if os.Getenv("SYSTEM_MONITOR_STORAGE_MODE") == "memory" {
		if cfg == nil || cfg.Server.Environment != "production" {
			slog.Warn("using explicit in-memory metrics storage; history is not durable")
			return io.NopCloser(nil), memory.NewRepository(), nil, nil, nil
		}
		return nil, nil, nil, nil, fmt.Errorf("in-memory storage is not allowed in production")
	}
	sqliteRepo, fileRoots, err := connectSQLite()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return sqliteRepo, sqliteRepo, sqliteRepo.RoutedDB(), fileRoots, nil
}

// Ensure *sqliterepo.Repository satisfies repository.Repository at compile time.
var _ repository.Repository = (*sqliterepo.Repository)(nil)

func serverWriteTimeout(cfg *config.Config) time.Duration {
	if cfg != nil && !cfg.IsProduction() {
		return 75 * time.Second
	}
	return 15 * time.Second
}
