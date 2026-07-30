// This file exposes focused service capabilities for operation handlers.
package orchestration

import (
	"context"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/findings"
	"agent-manager/internal/health"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/runreport"

	agentconfig "agent-manager/internal/config"

	"github.com/google/uuid"
)

// RunService owns run creation, lifecycle, recovery, and user-visible run
// state. Its 22 methods stay below the per-domain contract limit.
type RunService interface {
	CreateRun(context.Context, CreateRunRequest) (*domain.Run, error)
	CreateInvestigationRun(context.Context, CreateInvestigationRequest) (*domain.Run, error)
	CreateInvestigationApplyRun(context.Context, CreateInvestigationApplyRequest) (*domain.Run, error)
	ResumeFromFailedRun(context.Context, ResumeFromFailedRunRequest) (*domain.Run, error)
	GetRun(context.Context, uuid.UUID) (*domain.Run, error)
	GetRunByTag(context.Context, string) (*domain.Run, error)
	ListRuns(context.Context, RunListOptions) ([]*domain.Run, error)
	DeleteRun(context.Context, uuid.UUID) error
	StopRun(context.Context, uuid.UUID) error
	StopRunByTag(context.Context, string) error
	StopAllRuns(context.Context, StopAllOptions) (*StopAllResult, error)
	QuiesceScenario(context.Context, QuiesceOptions) (*QuiesceResult, error)
	ContinueRun(context.Context, ContinueRunRequest) (*domain.Run, error)
	ParkRunFromAgent(context.Context, ParkRunFromAgentRequest) (*ParkRunResult, error)
	GetAwaitResult(context.Context, uuid.UUID) (*AwaitResult, error)
	WakeRun(context.Context, WakeRunInput) (*domain.Run, error)
	RecoverRun(context.Context, uuid.UUID) (*RecoverResult, error)
	DeleteRunMessage(context.Context, uuid.UUID, uuid.UUID) (*domain.RunEvent, error)
	ResumeRun(context.Context, uuid.UUID) (*domain.Run, error)
	GetRunProgress(context.Context, uuid.UUID) (*domain.RunProgress, error)
	ListStaleRuns(context.Context, time.Duration) ([]*domain.Run, error)
	GetRunDiff(context.Context, uuid.UUID) (*sandbox.DiffResult, error)
	ImportTranscript(context.Context, ImportTranscriptRequest) (*domain.Run, error)
}

type ApprovalService interface {
	ApproveRun(context.Context, ApproveRequest) (*ApproveResult, error)
	RejectRun(context.Context, uuid.UUID, string, string) error
	PartialApprove(context.Context, PartialApproveRequest) (*ApproveResult, error)
	SyncRunFromSandbox(context.Context, SandboxSyncRequest) (*domain.Run, error)
}

type EventService interface {
	GetRunEvents(context.Context, uuid.UUID, event.GetOptions) ([]*domain.RunEvent, error)
	StreamRunEvents(context.Context, uuid.UUID, event.StreamOptions) (<-chan *domain.RunEvent, error)
}

type PolicyService interface {
	GetModelHealthSnapshot(context.Context) (health.Snapshot, error)
	ExplainProfilePolicy(context.Context, uuid.UUID) (*domain.ExecutionPolicySnapshot, error)
	ExplainRunPolicy(context.Context, uuid.UUID) (*domain.ExecutionPolicySnapshot, error)
}

type StatusService interface {
	GetHealth(context.Context) (*HealthStatus, error)
	GetRunnerStatus(context.Context) ([]*RunnerStatus, error)
	ProbeRunner(context.Context, domain.RunnerType) (*ProbeResult, error)
	SpawnStats() spawn.Stats
}

type MaintenanceService interface {
	PurgeData(context.Context, PurgeRequest) (*PurgeResult, error)
}

type InvestigationSettingsService interface {
	GetInvestigationSettings(context.Context) (*domain.InvestigationSettings, error)
	UpdateInvestigationSettings(context.Context, *domain.InvestigationSettings) error
	ResetInvestigationSettings(context.Context) error
}

type OrchestrationSettingsService interface {
	GetOrchestrationSettings(context.Context) (*agentconfig.OrchestrationSettings, error)
	UpdateOrchestrationSettings(context.Context, *agentconfig.OrchestrationSettings) error
	ResetOrchestrationSettings(context.Context) error
}

type PathValidationService interface {
	ValidatePath(context.Context, string, string) (*sandbox.PathValidationResult, error)
}

type IdentityService interface {
	VerifyIdentityToken(context.Context, string) (*IdentityVerifyResult, error)
}

type RunReportService interface {
	BuildRunReport(context.Context, uuid.UUID) (*runreport.RunReport, error)
}

type InvocationFactService interface {
	InvocationFacts(context.Context, uuid.UUID) ([]runreport.InvocationFact, error)
	Episodes(context.Context, uuid.UUID) ([]runreport.FrictionEpisode, error)
	SelfReportSpans(context.Context, uuid.UUID) ([]runreport.SelfReportSpan, error)
	EpisodeCohort(context.Context, invocationreadmodel.Filter, int) (runreport.EpisodeCohort, error)
	ReplayInvocationFacts(context.Context, uuid.UUID) (*ReplayResult, error)
	RefreshInvocationFacts(context.Context, uuid.UUID) (*ReplayResult, error)
	ReplayInvocationCorpus(context.Context, ReplayFilter, bool) (*ReplaySummary, error)
	AggregateInvocationFacts(context.Context, invocationreadmodel.Filter, string, int) ([]invocationreadmodel.AggregateRow, error)
	SelectInvocationCohort(context.Context, invocationreadmodel.Filter, int) (invocationreadmodel.Cohort, error)
	InvocationMetrics(context.Context, invocationreadmodel.Filter) (invocationreadmodel.Metrics, error)
}

type FindingsService interface {
	ListFindings(context.Context, findings.Filter) ([]findings.Finding, error)
}

type ProjectRootService interface {
	GetDefaultProjectRoot() string
}

// HandlerServices is wiring data, not a god-interface. Each field is a
// narrow capability boundary; handlers only gain the capabilities their route
// implementation explicitly uses.
type HandlerServices struct {
	ProfileService
	TaskService
	WorkflowService
	RunService
	ApprovalService
	EventService
	PolicyService
	StatusService
	MaintenanceService
	InvestigationSettingsService
	OrchestrationSettingsService
	PathValidationService
	IdentityService
	RunReportService
	InvocationFactService
	FindingsService
	ProjectRootService
}

// NewHandlerServices adapts the concrete coordinator at the composition root.
// The bundle is intentionally constructed only during wiring; application
// code receives individual capability interfaces instead of Orchestrator.
func NewHandlerServices(orchestrator *Orchestrator) HandlerServices {
	return HandlerServices{
		ProfileService:               orchestrator,
		TaskService:                  orchestrator,
		WorkflowService:              orchestrator,
		RunService:                   orchestrator,
		ApprovalService:              orchestrator,
		EventService:                 orchestrator,
		PolicyService:                orchestrator,
		StatusService:                orchestrator,
		MaintenanceService:           orchestrator,
		InvestigationSettingsService: orchestrator,
		OrchestrationSettingsService: orchestrator,
		PathValidationService:        orchestrator,
		IdentityService:              orchestrator,
		RunReportService:             orchestrator,
		InvocationFactService:        orchestrator,
		FindingsService:              orchestrator,
		ProjectRootService:           orchestrator,
	}
}

// EmptyHandlerServices is for handler tests that exercise middleware or a
// route with no orchestration dependency.
func EmptyHandlerServices() HandlerServices { return HandlerServices{} }

var (
	_ RunService                   = (*Orchestrator)(nil)
	_ ApprovalService              = (*Orchestrator)(nil)
	_ EventService                 = (*Orchestrator)(nil)
	_ PolicyService                = (*Orchestrator)(nil)
	_ StatusService                = (*Orchestrator)(nil)
	_ MaintenanceService           = (*Orchestrator)(nil)
	_ InvestigationSettingsService = (*Orchestrator)(nil)
	_ OrchestrationSettingsService = (*Orchestrator)(nil)
	_ PathValidationService        = (*Orchestrator)(nil)
	_ IdentityService              = (*Orchestrator)(nil)
	_ RunReportService             = (*Orchestrator)(nil)
	_ InvocationFactService        = (*Orchestrator)(nil)
	_ FindingsService              = (*Orchestrator)(nil)
	_ ProjectRootService           = (*Orchestrator)(nil)
)
