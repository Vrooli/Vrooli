package handlers

import (
	"context"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

// MonitorQuerier provides read access to system metrics.
type MonitorQuerier interface {
	GetCurrentMetrics(ctx context.Context) (*models.MetricsResponse, error)
	GetCurrentMetricsFresh(ctx context.Context) (*models.MetricsResponse, error)
	GetDetailedMetrics(ctx context.Context) (*models.DetailedMetrics, error)
	GetDiskDetail(ctx context.Context) (*models.DiskDetailResponse, error)
	GetMetricsTimeline(ctx context.Context, windowSeconds, sampleIntervalSeconds int) (*models.MetricsTimelineResponse, error)
	GetProcessMonitorData(ctx context.Context) (*models.ProcessMonitorData, error)
	GetProcessTimeline(ctx context.Context, window time.Duration, owner string, top int) ([]repository.ProcessTimelineEntry, error)
	GetInfrastructureMonitorData(ctx context.Context) (*models.InfrastructureMonitorData, error)
	IsActive() bool
}

// InvestigationManager provides investigation operations.
type InvestigationManager interface {
	TriggerInvestigation(ctx context.Context, autoFix bool, note string) (*models.Investigation, error)
	GetInvestigation(ctx context.Context, id string) (*models.Investigation, error)
	GetLatestInvestigation(ctx context.Context) (*models.Investigation, error)
	ListInvestigations(ctx context.Context, limit int) ([]*models.Investigation, error)
	UpdateInvestigationStatus(ctx context.Context, id string, status string) error
	UpdateInvestigationFindings(ctx context.Context, id string, findings string, details map[string]interface{}) error
	UpdateInvestigationProgress(ctx context.Context, id string, progress int) error
	AddInvestigationStep(ctx context.Context, id string, step models.InvestigationStep) error
	GetCooldownStatus(ctx context.Context) (*models.CooldownStatus, error)
	ResetCooldown(ctx context.Context) error
	UpdateCooldownPeriod(ctx context.Context, periodSeconds int) error
	GetTriggers(ctx context.Context) (map[string]*models.TriggerConfig, error)
	UpdateTrigger(ctx context.Context, id string, enabled *bool, autoFix *bool, threshold *float64) error
	GetInvestigationAgentStatus(ctx context.Context, id string) (*models.Investigation, error)
	StopInvestigationAgent(ctx context.Context, id string) error
	GetAgentConfig(ctx context.Context) (*services.AgentConfigResponse, error)
	GetAvailableRunners(ctx context.Context) ([]services.RunnerResponse, error)
	UpdateAgentConfig(ctx context.Context, runnerType, model string, maxTurns, timeoutSeconds int32, allowedTools []string, skipPermissions bool, sandboxMode string) (*services.AgentConfigResponse, error)
	GetAgentStatus(ctx context.Context) (*services.AgentStatusResponse, error)
}

// ScriptRunner provides script management and execution.
type ScriptRunner interface {
	ListScripts() ([]services.ScriptMeta, error)
	GetScript(id string) (services.ScriptMeta, string, error)
	ExecuteScript(ctx context.Context, id string, contentOverride string) (services.ScriptExecution, error)
}

// ReportGenerator provides report operations.
type ReportGenerator interface {
	GenerateReport(ctx context.Context, reportType string) (*models.EnhancedSystemReport, error)
	ListReports(ctx context.Context) ([]*models.EnhancedSystemReport, error)
	GetReport(ctx context.Context, reportID string) (*models.EnhancedSystemReport, error)
}

// MaintenanceProvider provides metrics-lifecycle maintenance operations.
type MaintenanceProvider interface {
	RetentionPreview(ctx context.Context, retentionDays int) (repository.RetentionEstimate, repository.DatabaseStats, error)
	RetentionApply(ctx context.Context, retentionDays int, confirm bool) (repository.RetentionResult, repository.DatabaseStats, repository.DatabaseStats, error)
	CompactionPreview(ctx context.Context) (repository.DatabaseStats, int64, error)
	CompactionApply(ctx context.Context, confirm bool) (repository.CompactionResult, error)
}

// CapacityProvider provides read access to the platform capacity ledger plus
// policy mutation. It reads claims/findings/policy and the live per-GPU
// contention picture; it never mutates claims (those flow through the broker).
type CapacityProvider interface {
	Overview(ctx context.Context) (services.CapacityOverview, error)
	ListClaims(ctx context.Context, ownerID string, activeOnly bool) ([]capacityapp.ClaimView, error)
	Reconcile(ctx context.Context) ([]engine.Finding, error)
	Policy(ctx context.Context) ([]capacityapp.PolicyEntry, error)
	SetPolicy(ctx context.Context, key, value string) ([]capacityapp.PolicyEntry, error)
}

// SettingsProvider provides settings management.
type SettingsProvider interface {
	GetSettings() services.Settings
	UpdateSettings(newSettings services.Settings) error
	IsActive() bool
	SetActive(active bool) error
	ResetSettings() error
	GetMaintenanceState() string
	SetMaintenanceState(state string) error
}
