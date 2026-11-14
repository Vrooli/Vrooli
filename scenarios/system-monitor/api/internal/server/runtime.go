package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/filerouting"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/agentmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/investigations"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
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
	apiLog := setupLogging(cfg)

	closer, repo, routedDB, fileRoots, err := connectRepository(cfg, apiLog)
	if err != nil {
		return fmt.Errorf("initialize metrics storage: %w", err)
	}

	alertSvc := services.NewAlertService(cfg, repo,
		services.WithAlertLogger(apiLog.With("service", "alerts")),
		services.WithAlertEventPublisher(eventbus.NewDiscoveredClient(context.Background())))
	monitorSvc := services.NewMonitorService(cfg, repo, infrastructure.NewMeasuredProvider(),
		services.WithMonitorLogger(apiLog.With("service", "monitor")))

	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		Logger:      apiLog.With("service", "agent_manager"),
		ProfileName: cfg.AgentManager.ProfileName,
		ProfileKey:  cfg.AgentManager.ProfileKey,
		Timeout:     cfg.AgentManager.Timeout,
		Enabled:     cfg.AgentManager.Enabled,
	})

	settingsMgr := services.NewSettingsManager(services.WithSettingsLogger(apiLog.With("service", "settings")))

	investigationSvc, err := services.NewInvestigationService(cfg, repo, monitorSvc, agentSvc, settingsMgr,
		services.WithLogger(apiLog.With("service", "investigation")))
	if err != nil {
		return fmt.Errorf("create investigation service: %w", err)
	}
	catalog, err := investigations.Load(services.ResolveRuntimeStateBasePath())
	if err != nil {
		return fmt.Errorf("load investigation catalog: %w", err)
	}
	scriptSvc := services.NewCatalogScriptService(catalog, services.ResolveRuntimeStateBasePath())
	nativeInvestigator := services.NewNativeInvestigator()
	nativeInvestigator.SetMemoryPressureSource(monitorSvc.GetMemoryPressure)
	nativeInvestigator.SetKernelMemorySource(monitorSvc.GetKernelMemory)
	scriptSvc.SetNativeRunner(nativeInvestigator)
	var runHistory *investigations.Service
	if sqliteRepo, ok := repo.(*sqliterepo.Repository); ok {
		runRepo := investigations.NewSQLiteRepository(sqliteRepo.RoutedDB())
		runHistory = investigations.NewService(runRepo, 7)
		scriptSvc.SetRunRepository(runRepo)
	}
	reportSvc := services.NewReportService(cfg, repo)
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
	maintenanceSvc := services.NewMetricsMaintenanceService(repo, services.WithCollectorRetention(cfg.Monitoring.CollectorRetention))
	retentionScheduler := services.NewRetentionScheduler(maintenanceSvc, settingsMgr, apiLog.With("service", "retention")).
		WithProcessRetention(repo, cfg.Monitoring.RawRetention, cfg.Monitoring.RollupRetention)
	if receiptPath, err := services.AutoCompactionReceiptPath(); err == nil {
		policy := services.ParseCompactionWindow(cfg.Monitoring.CompactionWindow, services.DefaultAutoCompactionPolicy())
		retentionScheduler.WithAutoCompaction(services.NewAutoCompactor(repo, policy, receiptPath))
	} else {
		apiLog.Warn("auto compaction disabled: receipt path unavailable", "error", err)
	}
	retentionScheduler.Start()

	// Disk-pressure lifecycle: evaluate the configured threshold on a schedule
	// and persist a violation when it is crossed. Without this loop the
	// threshold setting, the violation model, and the alert repository are all
	// reachable only from tests.
	thresholdScheduler := services.NewThresholdScheduler(settingsMgr, alertSvc, repo, apiLog.With("service", "threshold"),
		services.WithPressureReporter(cleanupmanager.NewClient(cleanupmanager.Config{BaseURL: cfg.Resources.StorageManagerURL})),
		services.WithWriterSampler(defaultWriterSampler(cfg)),
		services.WithCPUObservationSource(monitorSvc))
	thresholdScheduler.Start()

	// Server-owned alert state: every rule (CPU, memory, TCP, GPU, memory
	// pressure, orphaned agents) is evaluated here and persisted with
	// sustain/dedup/auto-resolve. Clients read it; none derive alerts.
	alertEvaluator := services.NewAlertEvaluator(settingsMgr, monitorSvc, repo, alertSvc, apiLog.With("service", "alerts"),
		services.WithAdditionalAlertRules(services.NewOrphanedAgentsRule(services.NewCLIAgentInventory(nil, nil), nil)),
		services.WithAlertEscalator(services.NewAnomalyEscalator(alertSvc, scriptSvc, nil)))
	alertEvaluator.Start()

	healthHandler := handlers.NewHealthHandler(cfg, monitorSvc, settingsMgr)
	metricsHandler := handlers.NewMetricsHandler(cfg, monitorSvc, apiLog.With("handler", "metrics"))
	investigationHandler := handlers.NewInvestigationHandler(cfg, investigationSvc, scriptSvc, apiLog.With("handler", "investigations"))
	if runHistory != nil {
		investigationHandler.SetRunHistory(runHistory)
	}
	reportHandler := handlers.NewReportHandler(cfg, reportSvc, apiLog.With("handler", "reports"))
	settingsHandler := handlers.NewSettingsHandler(settingsMgr, apiLog.With("handler", "settings"))
	maintenanceHandler := handlers.NewMaintenanceHandler(maintenanceSvc, apiLog.With("handler", "maintenance"))
	capacityHandler := handlers.NewCapacityHandler(services.NewCapacityService(), apiLog.With("handler", "capacity"))

	// Forensics + logs wiring.
	executor := shellExec{}
	journalReader := journal.NewReader(executor)
	forensicsSvc := forensics.NewService(journalReader, executor, forensics.DefaultFileSystem(), time.Now)
	autohealClient := autoheal.NewClient(autoheal.Config{BaseURL: cfg.Resources.AutohealURL})
	forensicsHandler := handlers.NewForensicsHandler(forensicsSvc, autohealClient, apiLog.With("handler", "forensics"))
	logsHandler := handlers.NewLogsHandler(journalReader, apiLog.With("handler", "logs"))
	diskPressureHandler := handlers.NewDiskPressureHandler(thresholdScheduler, repo, apiLog.With("handler", "disk-pressure"))

	// The device-graph read verb serves from the monitor service's shared
	// cached provider, the same one the 60s collector uses.
	alertsHandler := handlers.NewAlertsHandler(alertEvaluator, repo, apiLog.With("handler", "alerts"))

	deviceGraphHandler := handlers.NewDeviceGraphHandler(monitorSvc.DeviceGraphs(), apiLog.With("handler", "device-graph"), cfg)

	router := buildRouter(cfg, healthHandler, metricsHandler, investigationHandler, reportHandler, settingsHandler, maintenanceHandler, capacityHandler, forensicsHandler, logsHandler, diskPressureHandler, deviceGraphHandler, alertsHandler)
	rootMux := http.NewServeMux()
	if routedDB != nil && fileRoots != nil {
		devrouting.RegisterWithFileRoots(rootMux, routedDB, fileRoots)
	}
	rootMux.Handle("/", router)
	handler := apihttp.TestModeMiddleware(buildMiddleware(cfg, rootMux, apiLog))

	srv := &http.Server{
		Addr:         ":" + cfg.Server.APIPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: serverWriteTimeout(cfg),
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		apiLog.Info("System Monitor API starting",
			"port", cfg.Server.APIPort,
			"environment", cfg.Server.Environment,
			"version", cfg.Server.Version,
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			apiLog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	waitForShutdown(monitorSvc, investigationSvc, retentionScheduler, thresholdScheduler, alertEvaluator, srv, closer, apiLog)
	return nil
}

func defaultWriterSampler(cfg *config.Config) *services.WriterSampler {
	return services.NewWriterSamplerWithInterval(defaultWriterRoots(cfg), writerSampleInterval())
}

func writerSampleInterval() time.Duration {
	const fallback = 60 * time.Second
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return fallback
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		return fallback
	}
	var doc struct {
		Storage struct {
			Recovery struct {
				SampleIntervalSeconds int `json:"sample_interval_seconds"`
			} `json:"recovery"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(data, &doc); err != nil || doc.Storage.Recovery.SampleIntervalSeconds <= 0 {
		return fallback
	}
	return time.Duration(doc.Storage.Recovery.SampleIntervalSeconds) * time.Second
}

func defaultWriterRoots(cfg *config.Config) []services.GovernedRoot {
	home, _ := os.UserHomeDir()
	cache, _ := os.UserCacheDir()
	goCache := cfg.Monitoring.GoCachePath
	if goCache == "" {
		goCache = filepath.Join(cache, "go-build")
	}
	playwright := cfg.Monitoring.PlaywrightCachePath
	if playwright == "" {
		playwright = filepath.Join(cache, "ms-playwright")
	}
	runtimeHome := cfg.Monitoring.RuntimeHome
	if runtimeHome == "" {
		runtimeHome = filepath.Join(home, ".vrooli")
	}
	runtimeHome = strings.TrimSpace(runtimeHome)
	return []services.GovernedRoot{
		{ID: "go-build-cache", Root: goCache, Mount: "/", HotWriterBytesHour: 20 * 1024 * 1024 * 1024, MeasureBudget: 100 * time.Millisecond},
		{ID: "playwright-cache", Root: playwright, Mount: "/", HotWriterBytesHour: 20 * 1024 * 1024 * 1024, MeasureBudget: 100 * time.Millisecond},
		{ID: "var-log", Root: "/var/log", Mount: "/", HotWriterBytesHour: 20 * 1024 * 1024 * 1024, MeasureBudget: 100 * time.Millisecond},
		{ID: "go-work-dirs", Root: filepath.Join(runtimeHome, "tmp", "go-work"), Mount: "/", HotWriterBytesHour: 1 * 1024 * 1024 * 1024, MeasureBudget: 2 * time.Second, ExpandChildren: true},
	}
}

func connectRepository(cfg *config.Config, logger logx.Logger) (io.Closer, repository.Repository, *database.RoutedDB, *filerouting.RoutedRoots, error) {
	if cfg.Monitoring.StorageMode == "memory" {
		if cfg == nil || cfg.Server.Environment != "production" {
			logger.Warn("using explicit in-memory metrics storage; history is not durable")
			return io.NopCloser(nil), memory.NewRepository(), nil, nil, nil
		}
		return nil, nil, nil, nil, fmt.Errorf("in-memory storage is not allowed in production")
	}
	sqliteRepo, fileRoots, err := connectSQLite(logger)
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
