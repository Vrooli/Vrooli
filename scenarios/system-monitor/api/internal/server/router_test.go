package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"
	capacityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity/capacityconnect"
	healthpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/health"
	healthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/health/healthconnect"
	investigationspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations"
	investigationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations/investigationsconnect"
	maintenancepb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/maintenance"
	maintenanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/maintenance/maintenanceconnect"
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics/metricsconnect"
	reportspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/reports"
	reportsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/reports/reportsconnect"
	scriptspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts"
	scriptsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts/scriptsconnect"
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings/settingsconnect"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers"
	handlermocks "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

func TestMountDebugRoutesDevelopment(t *testing.T) {
	r := http.NewServeMux()
	mountDebugRoutes(&config.Config{Server: config.ServerConfig{Environment: "development"}}, r)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /debug/pprof/ = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMountDebugRoutesProduction(t *testing.T) {
	r := http.NewServeMux()
	mountDebugRoutes(&config.Config{Server: config.ServerConfig{Environment: "production"}}, r)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /debug/pprof/ = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMountConnectRoutesHealth(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{CPUUsage: 12.5, MemoryUsage: 42})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := healthconnect.NewHealthServiceClient(server.Client(), server.URL)
	resp, err := client.Health(context.Background(), connect.NewRequest(&healthpb.HealthRequest{}))
	if err != nil {
		t.Fatalf("Health Connect call failed: %v", err)
	}

	if resp.Msg.GetService() != "system-monitor" {
		t.Fatalf("service = %q, want system-monitor", resp.Msg.GetService())
	}
	if resp.Msg.GetStatus() == commonv1.HealthStatus_HEALTH_STATUS_UNSPECIFIED {
		t.Fatalf("status = %s, want specified health status", resp.Msg.GetStatus())
	}
	if !resp.Msg.GetReadiness() {
		t.Fatalf("readiness = false, want true")
	}
	if resp.Msg.GetMetrics()["cpu_usage_percent"].GetDoubleValue() != 12.5 {
		t.Fatalf("cpu_usage_percent = %v, want 12.5", resp.Msg.GetMetrics()["cpu_usage_percent"])
	}
}

func TestMountConnectRoutesMetrics(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().
		WithCurrentMetrics(&models.MetricsResponse{CPUUsage: 12.5, MemoryUsage: 42}).
		WithFreshMetrics(&models.MetricsResponse{CPUUsage: 88.8, MemoryUsage: 44})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := metricsconnect.NewMetricsServiceClient(server.Client(), server.URL)
	resp, err := client.GetCurrentMetrics(context.Background(), connect.NewRequest(&metricspb.GetCurrentMetricsRequest{Fresh: true}))
	if err != nil {
		t.Fatalf("GetCurrentMetrics Connect call failed: %v", err)
	}

	if resp.Msg.GetMetrics().GetCpuUsage() != 88.8 {
		t.Fatalf("cpu_usage = %v, want 88.8", resp.Msg.GetMetrics().GetCpuUsage())
	}
}

func TestMountConnectRoutesReports(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := reportsconnect.NewReportsServiceClient(server.Client(), server.URL)
	resp, err := client.GetReport(context.Background(), connect.NewRequest(&reportspb.GetReportRequest{Id: "report-1"}))
	if err != nil {
		t.Fatalf("GetReport Connect call failed: %v", err)
	}

	if resp.Msg.GetReport().GetReportId() != "report-1" {
		t.Fatalf("report_id = %q, want report-1", resp.Msg.GetReport().GetReportId())
	}
}

func TestMountConnectRoutesSettings(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := settingsconnect.NewSettingsServiceClient(server.Client(), server.URL)
	resp, err := client.GetSettings(context.Background(), connect.NewRequest(&settingspb.GetSettingsRequest{}))
	if err != nil {
		t.Fatalf("GetSettings Connect call failed: %v", err)
	}

	if resp.Msg.GetSettings().GetMetricCollectionInterval() != 20 {
		t.Fatalf("metric_collection_interval = %d, want 20", resp.Msg.GetSettings().GetMetricCollectionInterval())
	}
}

func TestMountConnectRoutesCapacity(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := capacityconnect.NewCapacityServiceClient(server.Client(), server.URL)
	resp, err := client.GetCapacityOverview(context.Background(), connect.NewRequest(&capacitypb.GetCapacityOverviewRequest{}))
	if err != nil {
		t.Fatalf("GetCapacityOverview Connect call failed: %v", err)
	}

	if got := resp.Msg.GetGpus()[0].GetName(); got != "RTX" {
		t.Fatalf("gpu name = %q, want RTX", got)
	}
	if !resp.Msg.GetSensingAvailable() {
		t.Fatalf("sensing_available = false, want true")
	}
}

func TestMountConnectRoutesMaintenance(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := maintenanceconnect.NewMaintenanceServiceClient(server.Client(), server.URL)
	resp, err := client.MetricsRetentionPreview(context.Background(), connect.NewRequest(&maintenancepb.MetricsRetentionPreviewRequest{RetentionDays: 7}))
	if err != nil {
		t.Fatalf("MetricsRetentionPreview Connect call failed: %v", err)
	}

	if got := resp.Msg.GetEstimate().GetRowCount(); got != 3 {
		t.Fatalf("row_count = %d, want 3", got)
	}
	if got := resp.Msg.GetDatabaseStats().GetMetricRows(); got != 11 {
		t.Fatalf("metric_rows = %d, want 11", got)
	}
}

func TestMountConnectRoutesScripts(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := scriptsconnect.NewScriptsServiceClient(server.Client(), server.URL)
	resp, err := client.GetScript(context.Background(), connect.NewRequest(&scriptspb.GetScriptRequest{Id: "cpu-check"}))
	if err != nil {
		t.Fatalf("GetScript Connect call failed: %v", err)
	}

	if got := resp.Msg.GetScript().GetName(); got != "CPU Check" {
		t.Fatalf("script name = %q, want CPU Check", got)
	}
	if got := resp.Msg.GetContent(); got != "#!/usr/bin/env bash\n" {
		t.Fatalf("content = %q, want script body", got)
	}
}

func TestMountConnectRoutesInvestigations(t *testing.T) {
	r := http.NewServeMux()
	monitor := handlermocks.NewMonitorQuerier().WithCurrentMetrics(&models.MetricsResponse{})
	health := handlers.NewHealthHandler(
		&config.Config{Server: config.ServerConfig{ServiceName: "system-monitor", Version: "test"}},
		monitor,
		nil,
	)
	metrics := handlers.NewMetricsHandler(&config.Config{}, monitor, nil)
	reports := handlers.NewReportHandler(&config.Config{}, fakeReportGenerator{reports: []*models.EnhancedSystemReport{sampleReport("report-1")}}, nil)
	settings := handlers.NewSettingsHandler(newFakeSettingsProvider(), nil)
	capacity := handlers.NewCapacityHandler(newFakeCapacityProvider(), nil)
	maintenance := handlers.NewMaintenanceHandler(newFakeMaintenanceProvider(), nil)
	investigation := handlers.NewInvestigationHandler(&config.Config{Server: config.ServerConfig{APIPort: "16914"}}, newFakeInvestigationManager(), newFakeScriptRunner(), nil)
	mountConnectRoutes(r, health, metrics, reports, settings, capacity, maintenance, investigation)

	server := httptest.NewServer(r)
	defer server.Close()

	client := investigationsconnect.NewInvestigationsServiceClient(server.Client(), server.URL)
	listResp, err := client.ListInvestigations(context.Background(), connect.NewRequest(&investigationspb.ListInvestigationsRequest{}))
	if err != nil {
		t.Fatalf("ListInvestigations Connect call failed: %v", err)
	}
	if got := listResp.Msg.GetInvestigations()[0].GetId(); got != "inv-1" {
		t.Fatalf("investigation id = %q, want inv-1", got)
	}

	triggerResp, err := client.TriggerInvestigation(context.Background(), connect.NewRequest(&investigationspb.TriggerInvestigationRequest{AutoFix: true, Note: "check CPU"}))
	if err != nil {
		t.Fatalf("TriggerInvestigation Connect call failed: %v", err)
	}
	if got := triggerResp.Msg.GetInvestigationId(); got != "inv-triggered" {
		t.Fatalf("triggered investigation id = %q, want inv-triggered", got)
	}
	if !triggerResp.Msg.GetAutoFix() {
		t.Fatalf("auto_fix = false, want true")
	}
}

func TestServerWriteTimeoutAllowsDevelopmentProfiles(t *testing.T) {
	dev := serverWriteTimeout(&config.Config{Server: config.ServerConfig{Environment: "development"}})
	if dev < 60*time.Second {
		t.Fatalf("development write timeout = %v, want at least 60s", dev)
	}

	prod := serverWriteTimeout(&config.Config{Server: config.ServerConfig{Environment: "production"}})
	if prod != 15*time.Second {
		t.Fatalf("production write timeout = %v, want 15s", prod)
	}
}

type fakeSettingsProvider struct {
	settings         services.Settings
	maintenanceState string
}

func newFakeSettingsProvider() *fakeSettingsProvider {
	return &fakeSettingsProvider{
		settings: services.Settings{
			Active:                        true,
			MetricCollectionInterval:      20,
			AnomalyDetectionInterval:      30,
			ThresholdCheckInterval:        40,
			CooldownPeriodSeconds:         60,
			CPUThreshold:                  80,
			MemoryThreshold:               85,
			DiskThreshold:                 90,
			MetricsRetentionDays:          30,
			RetentionCheckIntervalSeconds: 3600,
			RetentionRunOnStartup:         true,
			CompactAfterRetention:         true,
		},
		maintenanceState: "inactive",
	}
}

func (f *fakeSettingsProvider) GetSettings() services.Settings {
	return f.settings
}

func (f *fakeSettingsProvider) UpdateSettings(newSettings services.Settings) error {
	f.settings = newSettings
	return nil
}

func (f *fakeSettingsProvider) IsActive() bool {
	return f.settings.Active
}

func (f *fakeSettingsProvider) SetActive(active bool) error {
	f.settings.Active = active
	return nil
}

func (f *fakeSettingsProvider) ResetSettings() error {
	f.settings = newFakeSettingsProvider().settings
	return nil
}

func (f *fakeSettingsProvider) GetMaintenanceState() string {
	return f.maintenanceState
}

func (f *fakeSettingsProvider) SetMaintenanceState(state string) error {
	f.maintenanceState = state
	return nil
}

type fakeCapacityProvider struct {
	overview services.CapacityOverview
	claims   []capacityapp.ClaimView
	findings []engine.Finding
	policy   []capacityapp.PolicyEntry
}

func newFakeCapacityProvider() *fakeCapacityProvider {
	return &fakeCapacityProvider{
		overview: services.CapacityOverview{
			GPUs:             []services.GpuContention{{Index: 0, Name: "RTX", TotalBytes: 16, FreeBytes: 4, ClaimedBytes: 8}},
			Claims:           []capacityapp.ClaimView{{ClaimID: "claim-1", OwnerID: "system-monitor"}},
			SensingAvailable: true,
		},
		claims: []capacityapp.ClaimView{{ClaimID: "claim-1", OwnerID: "system-monitor"}},
		policy: []capacityapp.PolicyEntry{{Key: "enforce", Value: "off"}},
	}
}

func (f *fakeCapacityProvider) Overview(context.Context) (services.CapacityOverview, error) {
	return f.overview, nil
}

func (f *fakeCapacityProvider) ListClaims(context.Context, string, bool) ([]capacityapp.ClaimView, error) {
	return f.claims, nil
}

func (f *fakeCapacityProvider) Reconcile(context.Context) ([]engine.Finding, error) {
	return f.findings, nil
}

func (f *fakeCapacityProvider) Policy(context.Context) ([]capacityapp.PolicyEntry, error) {
	return f.policy, nil
}

func (f *fakeCapacityProvider) SetPolicy(context.Context, string, string) ([]capacityapp.PolicyEntry, error) {
	return f.policy, nil
}

type fakeMaintenanceProvider struct {
	estimate repository.RetentionEstimate
	stats    repository.DatabaseStats
	result   repository.RetentionResult
}

func newFakeMaintenanceProvider() *fakeMaintenanceProvider {
	return &fakeMaintenanceProvider{
		estimate: repository.RetentionEstimate{RowCount: 3, PayloadBytes: 128},
		stats:    repository.DatabaseStats{PageSize: 4096, PageCount: 10, FreelistCount: 2, SizeBytes: 40960, MetricRows: 11},
		result:   repository.RetentionResult{DeletedRows: 3},
	}
}

func (f *fakeMaintenanceProvider) RetentionPreview(context.Context, int) (repository.RetentionEstimate, repository.DatabaseStats, error) {
	return f.estimate, f.stats, nil
}

func (f *fakeMaintenanceProvider) RetentionApply(context.Context, int, bool) (repository.RetentionResult, repository.DatabaseStats, repository.DatabaseStats, error) {
	return f.result, f.stats, f.stats, nil
}

func (f *fakeMaintenanceProvider) CompactionPreview(context.Context) (repository.DatabaseStats, int64, error) {
	return f.stats, 8192, nil
}

func (f *fakeMaintenanceProvider) CompactionApply(context.Context, bool) (repository.CompactionResult, error) {
	return repository.CompactionResult{StatsBefore: f.stats, StatsAfter: f.stats, ReclaimedBytes: 8192}, nil
}

type fakeInvestigationManager struct {
	investigation *models.Investigation
	triggers      map[string]*models.TriggerConfig
}

func newFakeInvestigationManager() *fakeInvestigationManager {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	return &fakeInvestigationManager{
		investigation: &models.Investigation{
			ID:        "inv-1",
			Status:    models.StatusQueued,
			AnomalyID: "anom-1",
			StartTime: now,
			Progress:  10,
		},
		triggers: map[string]*models.TriggerConfig{
			"high_cpu": {
				ID:          "high_cpu",
				Name:        "High CPU",
				Description: "CPU usage above threshold",
				Enabled:     true,
				AutoFix:     false,
				Threshold:   80,
				Unit:        "%",
				Condition:   "above",
			},
		},
	}
}

func (f *fakeInvestigationManager) TriggerInvestigation(context.Context, bool, string) (*models.Investigation, error) {
	now := time.Date(2026, 6, 24, 12, 1, 0, 0, time.UTC)
	return &models.Investigation{ID: "inv-triggered", Status: models.StatusQueued, StartTime: now}, nil
}

func (f *fakeInvestigationManager) GetInvestigation(context.Context, string) (*models.Investigation, error) {
	return f.investigation, nil
}

func (f *fakeInvestigationManager) GetLatestInvestigation(context.Context) (*models.Investigation, error) {
	return f.investigation, nil
}

func (f *fakeInvestigationManager) ListInvestigations(context.Context, int) ([]*models.Investigation, error) {
	return []*models.Investigation{f.investigation}, nil
}

func (f *fakeInvestigationManager) UpdateInvestigationStatus(_ context.Context, _ string, status string) error {
	f.investigation.Status = status
	return nil
}

func (f *fakeInvestigationManager) UpdateInvestigationFindings(_ context.Context, _ string, findings string, details map[string]interface{}) error {
	f.investigation.Findings = findings
	f.investigation.Details = details
	return nil
}

func (f *fakeInvestigationManager) UpdateInvestigationProgress(_ context.Context, _ string, progress int) error {
	f.investigation.Progress = progress
	return nil
}

func (f *fakeInvestigationManager) AddInvestigationStep(_ context.Context, _ string, step models.InvestigationStep) error {
	f.investigation.Steps = append(f.investigation.Steps, step)
	return nil
}

func (f *fakeInvestigationManager) GetCooldownStatus(context.Context) (*models.CooldownStatus, error) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	return &models.CooldownStatus{
		CooldownPeriodSeconds: 60,
		RemainingSeconds:      0,
		LastTriggerTime:       now,
		IsReady:               true,
	}, nil
}

func (f *fakeInvestigationManager) ResetCooldown(context.Context) error {
	return nil
}

func (f *fakeInvestigationManager) UpdateCooldownPeriod(context.Context, int) error {
	return nil
}

func (f *fakeInvestigationManager) GetTriggers(context.Context) (map[string]*models.TriggerConfig, error) {
	return f.triggers, nil
}

func (f *fakeInvestigationManager) UpdateTrigger(_ context.Context, id string, enabled *bool, autoFix *bool, threshold *float64) error {
	trigger := f.triggers[id]
	if enabled != nil {
		trigger.Enabled = *enabled
	}
	if autoFix != nil {
		trigger.AutoFix = *autoFix
	}
	if threshold != nil {
		trigger.Threshold = *threshold
	}
	return nil
}

func (f *fakeInvestigationManager) GetInvestigationAgentStatus(context.Context, string) (*models.Investigation, error) {
	return f.investigation, nil
}

func (f *fakeInvestigationManager) StopInvestigationAgent(context.Context, string) error {
	f.investigation.Status = models.StatusStopped
	return nil
}

func (f *fakeInvestigationManager) GetAgentConfig(context.Context) (*services.AgentConfigResponse, error) {
	return &services.AgentConfigResponse{}, nil
}

func (f *fakeInvestigationManager) GetAvailableRunners(context.Context) ([]services.RunnerResponse, error) {
	return nil, nil
}

func (f *fakeInvestigationManager) UpdateAgentConfig(context.Context, string, string, int32, int32, []string, bool, string) (*services.AgentConfigResponse, error) {
	return &services.AgentConfigResponse{}, nil
}

func (f *fakeInvestigationManager) GetAgentStatus(context.Context) (*services.AgentStatusResponse, error) {
	return &services.AgentStatusResponse{}, nil
}

type fakeScriptRunner struct {
	script services.ScriptMeta
}

func newFakeScriptRunner() *fakeScriptRunner {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	return &fakeScriptRunner{
		script: services.ScriptMeta{
			ID:          "cpu-check",
			Name:        "CPU Check",
			Description: "Checks CPU pressure",
			Category:    "cpu",
			Author:      "system-monitor",
			CreatedAt:   now,
			UpdatedAt:   now,
			Enabled:     true,
		},
	}
}

func (f *fakeScriptRunner) ListScripts() ([]services.ScriptMeta, error) {
	return []services.ScriptMeta{f.script}, nil
}

func (f *fakeScriptRunner) GetScript(string) (services.ScriptMeta, string, error) {
	return f.script, "#!/usr/bin/env bash\n", nil
}

func (f *fakeScriptRunner) ExecuteScript(context.Context, string, string) (services.ScriptExecution, error) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	return services.ScriptExecution{
		ScriptID:        f.script.ID,
		ExecutionID:     "exec-1",
		Status:          "completed",
		StartedAt:       now,
		CompletedAt:     now,
		Stdout:          "ok\n",
		ExitCode:        0,
		DurationSeconds: 0.1,
	}, nil
}

type fakeReportGenerator struct {
	reports []*models.EnhancedSystemReport
}

func (f fakeReportGenerator) GenerateReport(_ context.Context, reportType string) (*models.EnhancedSystemReport, error) {
	return sampleReport("generated-" + reportType), nil
}

func (f fakeReportGenerator) ListReports(context.Context) ([]*models.EnhancedSystemReport, error) {
	return f.reports, nil
}

func (f fakeReportGenerator) GetReport(_ context.Context, reportID string) (*models.EnhancedSystemReport, error) {
	for _, report := range f.reports {
		if report.ReportID == reportID {
			return report, nil
		}
	}
	return nil, nil
}

func sampleReport(id string) *models.EnhancedSystemReport {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	return &models.EnhancedSystemReport{
		ReportID:         id,
		ReportType:       "daily",
		GeneratedAt:      now,
		TimeRange:        models.TimeRange{StartTime: now.Add(-24 * time.Hour), EndTime: now},
		ActualDuration:   "24h",
		DateRangeDisplay: "June 23, 2026 to June 24, 2026",
		ExecutiveSummary: models.EnhancedExecutiveSummary{
			OverallHealth:   "healthy",
			KeyFindings:     []string{"No critical alerts"},
			TimeDescription: "24 hours",
			MetricsAnalyzed: 1,
		},
		Performance: models.PerformanceAnalysis{
			CPU:       models.MetricStats{Average: 12.5, Min: 10, Max: 15, PeakTime: now, MinTime: now.Add(-time.Hour)},
			Memory:    models.MetricStats{Average: 42, Min: 40, Max: 44, PeakTime: now, MinTime: now.Add(-time.Hour)},
			TimeRange: "2026-06-23 12:00 to 2026-06-24 12:00",
		},
		MetricsCount: 1,
	}
}
