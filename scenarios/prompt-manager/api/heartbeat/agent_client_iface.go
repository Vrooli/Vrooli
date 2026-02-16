package heartbeat

import (
	"context"
	"time"
)

// AgentClient abstracts the agent-manager HTTP API so callers can be tested
// with a mock implementation. The concrete *AgentManagerClient satisfies this
// interface.
type AgentClient interface {
	Health(ctx context.Context) (bool, error)
	EnsureProfile(ctx context.Context, req *EnsureProfileRequest) (*EnsureProfileResponse, error)
	CreateTask(ctx context.Context, task *Task) (*Task, error)
	CreateRun(ctx context.Context, req *CreateRunRequest) (*Run, error)
	GetRun(ctx context.Context, runID string) (*Run, error)
	WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*Run, error)
	StopRun(ctx context.Context, runID string) error
	GetRunEvents(ctx context.Context, runID string, afterSequence int64, limit int) ([]byte, error)
}
