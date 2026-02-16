package execution

import "time"

// DOC: docs/concepts/ARCHITECTURE.md#domain-concepts
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INVARIANTS.md

// Status represents an execution lifecycle state.
type Status string

const (
	StatusPending   Status = "pending"
	StatusScheduled Status = "scheduled"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// Mode controls when an execution starts.
type Mode string

const (
	ModeManual    Mode = "manual"
	ModeScheduled Mode = "scheduled"
	ModeYOLO      Mode = "yolo"
)

// Record is a persisted execution run record.
type Record struct {
	ExecutionID   string       `json:"execution_id"`
	BacklogKind   string       `json:"backlog_kind"`
	BacklogName   string       `json:"backlog_name"`
	TaskID        string       `json:"task_id,omitempty"`
	RunID         string       `json:"run_id,omitempty"`
	Status        Status       `json:"status"`
	Mode          Mode         `json:"mode"`
	ScheduledAt   string       `json:"scheduled_at,omitempty"`
	StartedAt     string       `json:"started_at,omitempty"`
	FinishedAt    string       `json:"finished_at,omitempty"`
	FailureReason string       `json:"failure_reason,omitempty"`
	StartedBy     string       `json:"started_by,omitempty"`
	Operation     string       `json:"operation,omitempty"`
	PromptTrace   *PromptTrace `json:"prompt_trace,omitempty"`
	CreatedAt     string       `json:"created_at"`
	UpdatedAt     string       `json:"updated_at"`
}

// PromptTrace captures prompt details used to launch the execution.
type PromptTrace struct {
	SkillID      string            `json:"skill_id"`
	Purpose      string            `json:"purpose"`
	Variables    map[string]string `json:"variables,omitempty"`
	Prompt       string            `json:"prompt"`
	UsedFallback bool              `json:"used_fallback"`
	CapturedAt   string            `json:"captured_at"`
}

// CreateRequest creates an execution record.
type CreateRequest struct {
	BacklogKind  string `json:"backlog_kind"`
	BacklogName  string `json:"backlog_name"`
	Mode         Mode   `json:"mode"`
	DelaySeconds int64  `json:"delay_seconds,omitempty"`
	StartedBy    string `json:"started_by,omitempty"`
	Operation    string `json:"operation,omitempty"`
}

// Policy controls default execution behavior when callers do not provide mode/delay.
type Policy struct {
	DefaultMode         Mode  `json:"default_mode"`
	DefaultDelaySeconds int64 `json:"default_delay_seconds"`
}

// ListFilters defines list query filters.
type ListFilters struct {
	Status      string
	Mode        string
	BacklogKind string
	BacklogName string
	StartedBy   string
	CreatedFrom string
	CreatedTo   string
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
