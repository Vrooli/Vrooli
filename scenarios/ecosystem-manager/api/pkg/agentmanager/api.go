package agentmanager

import (
	"context"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// AgentServiceAPI defines the interface for agent execution services.
// This abstraction enables unit testing without requiring a real agent-manager connection.
type AgentServiceAPI interface {
	// IsAvailable checks if agent-manager is reachable.
	IsAvailable(ctx context.Context) bool

	// Initialize ensures both task and insights profiles exist.
	// Call this at startup to create/update profiles.
	Initialize(ctx context.Context) error

	// UpdateProfiles updates both profiles with current settings.
	// Call this when settings change to propagate new config.
	UpdateProfiles(ctx context.Context) error

	// ResolveURL returns the current agent-manager base URL.
	ResolveURL(ctx context.Context) (string, error)

	// ExecuteTask starts a task run and waits for completion.
	ExecuteTask(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)

	// ExecuteTaskAsync starts a task run without waiting for completion.
	// Returns the run ID for tracking.
	ExecuteTaskAsync(ctx context.Context, req ExecuteRequest) (string, error)

	// ExecuteInsight runs an insight generation task.
	ExecuteInsight(ctx context.Context, req InsightRequest) (*ExecuteResult, error)

	// GetRunStatus returns the current status of a run.
	GetRunStatus(ctx context.Context, runID string) (*domainpb.Run, error)

	// StopRun stops an active run.
	StopRun(ctx context.Context, runID string) error

	// GetRunEvents returns events for a run.
	GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error)

	// WaitForRun waits for a run to complete.
	WaitForRun(ctx context.Context, runID string) (*domainpb.Run, error)
}

// Compile-time assertion that AgentService implements AgentServiceAPI.
var _ AgentServiceAPI = (*AgentService)(nil)
