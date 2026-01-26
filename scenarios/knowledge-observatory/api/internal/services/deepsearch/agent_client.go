package deepsearch

import (
	"context"
	"time"
)

// AgentClient defines the integration seam for agent-manager.
type AgentClient interface {
	EnsureProfile(ctx context.Context) error
	CreateRun(ctx context.Context, req AgentRunRequest) (string, error)
	GetRun(ctx context.Context, runID string) (*AgentRun, error)
	GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]AgentRunEvent, error)
}

// AgentRunRequest describes the task + run configuration for a deep search.
type AgentRunRequest struct {
	Title       string
	Description string
	Prompt      string
	ScopePath   string
	ProjectRoot string
	Tag         string
	Timeout     time.Duration
}

// AgentRun captures the minimal run state needed for deep search tracking.
type AgentRun struct {
	ID     string
	Status RunStatus
	Error  string
}

// RunStatus represents the status of an agent run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusComplete  RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

// AgentRunEventType indicates the event payload type.
type AgentRunEventType string

const (
	EventMessage  AgentRunEventType = "message"
	EventProgress AgentRunEventType = "progress"
)

// AgentRunEvent normalizes agent-manager events for deep search.
type AgentRunEvent struct {
	Sequence        int64
	Type            AgentRunEventType
	Role            string
	Content         string
	ProgressPercent int32
	ProgressAction  string
	ProgressPhase   string
}
