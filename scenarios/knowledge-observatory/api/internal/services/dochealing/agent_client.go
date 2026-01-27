package dochealing

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
	GetRunDiff(ctx context.Context, runID string) (*RunDiff, error)
	ApproveRun(ctx context.Context, req ApprovalRequest) (*ApproveResult, error)
	RejectRun(ctx context.Context, req RejectRequest) error
}

// AgentRunRequest describes the task + run configuration for a healing job.
type AgentRunRequest struct {
	Title       string
	Description string
	Prompt      string
	ScopePath   string
	ProjectRoot string
	Tag         string
	Timeout     time.Duration
}

// AgentRun captures the minimal run state needed for healing tracking.
type AgentRun struct {
	ID      string
	Status  RunStatus
	Error   string
	Summary string
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

// AgentRunEvent normalizes agent-manager events for healing.
type AgentRunEvent struct {
	Sequence        int64
	Type            AgentRunEventType
	Role            string
	Content         string
	ProgressPercent int32
	ProgressAction  string
	ProgressPhase   string
}

// RunDiff holds diff details for a run.
type RunDiff struct {
	Content string
	Files   []RunFileDiff
}

// RunFileDiff summarizes a single file change.
type RunFileDiff struct {
	Path       string
	ChangeType string
	Additions  int32
	Deletions  int32
}

// ApprovalRequest wraps run approval parameters.
type ApprovalRequest struct {
	RunID     string
	Actor     string
	CommitMsg string
	Force     bool
}

// ApproveResult returns approval metadata.
type ApproveResult struct {
	Success      bool
	FilesApplied int32
	CommitHash   string
	Message      string
}

// RejectRequest wraps rejection parameters.
type RejectRequest struct {
	RunID  string
	Actor  string
	Reason string
}
