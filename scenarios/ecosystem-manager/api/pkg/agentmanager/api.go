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

	// Initialize ensures the task profile exists.
	// Call this at startup to create/update the profile.
	Initialize(ctx context.Context) error

	// UpdateProfiles updates the task profile with current settings.
	// Call this when settings change to propagate new config.
	UpdateProfiles(ctx context.Context) error

	// ResolveURL returns the current agent-manager base URL.
	ResolveURL(ctx context.Context) (string, error)

	// ExecuteTask starts a task run and waits for completion.
	ExecuteTask(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)

	// ExecuteTaskAsync starts a task run without waiting for completion.
	// Returns the run ID for tracking.
	ExecuteTaskAsync(ctx context.Context, req ExecuteRequest) (string, error)

	// GetRunStatus returns the current status of a run.
	GetRunStatus(ctx context.Context, runID string) (*domainpb.Run, error)

	// GetRunDiff returns the code-level diff a run produced (nil for in-place
	// runs without a sandbox). Used by the anti-gaming classifier.
	GetRunDiff(ctx context.Context, runID string) (*domainpb.RunDiff, error)

	// StopRun stops an active run.
	StopRun(ctx context.Context, runID string) error

	// GetRunEvents returns events for a run.
	GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error)

	// WaitForRun waits for a run to complete.
	WaitForRun(ctx context.Context, runID string) (*domainpb.Run, error)
}

// Compile-time assertion that AgentService implements AgentServiceAPI.
var _ AgentServiceAPI = (*AgentService)(nil)
