package execution

import "time"

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
	ExecutionID   string `json:"execution_id"`
	BacklogKind   string `json:"backlog_kind"`
	BacklogName   string `json:"backlog_name"`
	TaskID        string `json:"task_id,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	Status        Status `json:"status"`
	Mode          Mode   `json:"mode"`
	ScheduledAt   string `json:"scheduled_at,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	FinishedAt    string `json:"finished_at,omitempty"`
	FailureReason string `json:"failure_reason,omitempty"`
	StartedBy     string `json:"started_by,omitempty"`
	Operation     string `json:"operation,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
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
