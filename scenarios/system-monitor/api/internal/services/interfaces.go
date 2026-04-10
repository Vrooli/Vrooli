package services

import (
	"context"
	"net/http"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"system-monitor-api/internal/agentmanager"
	"system-monitor-api/internal/models"
)

// MetricsSource provides on-demand metrics collection.
// *MonitorService satisfies this interface.
type MetricsSource interface {
	GetCurrentMetricsFresh(ctx context.Context) (*models.MetricsResponse, error)
}

// AgentExecutor abstracts the agent-manager operations used by InvestigationService.
type AgentExecutor interface {
	IsEnabled() bool
	IsAvailable(ctx context.Context) bool
	Initialize(ctx context.Context, cfg *agentmanager.ProfileConfig) error
	Execute(ctx context.Context, req agentmanager.ExecuteRequest) (*agentmanager.ExecuteResult, error)
	GetProfile(ctx context.Context) (*domainpb.AgentProfile, error)
	GetProfileID() string
	UpdateProfile(ctx context.Context, cfg *agentmanager.ProfileConfig) (*domainpb.AgentProfile, error)
	GetAvailableRunners(ctx context.Context) ([]agentmanager.RunnerInfo, error)
	GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error)
	ListActiveRuns(ctx context.Context) ([]*domainpb.Run, error)
	StopRun(ctx context.Context, runID string) error
	ResolveURL(ctx context.Context) (string, error)
}

// HTTPDoer abstracts HTTP request execution for webhook delivery.
// *http.Client satisfies this interface.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
