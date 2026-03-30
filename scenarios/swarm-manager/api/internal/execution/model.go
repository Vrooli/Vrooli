package execution

import "time"

// DOC: docs/concepts/ARCHITECTURE.md#domain-concepts
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INVARIANTS.md

// Status represents an execution lifecycle state.
type Status string

const (
	StatusPending     Status = "pending"
	StatusScheduled   Status = "scheduled"
	StatusStarting    Status = "starting"
	StatusRunning     Status = "running"
	StatusNeedsReview Status = "needs_review"
	StatusValidating  Status = "validating"
	StatusNeedsFixup  Status = "needs_fixup"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
	StatusCanceled    Status = "canceled"
)

// Mode controls when an execution starts.
type Mode string

const (
	ModeManual    Mode = "manual"
	ModeScheduled Mode = "scheduled"
	ModeYOLO      Mode = "yolo"
)

// ValidateMode returns true if m is a known execution mode.
func ValidateMode(m Mode) bool {
	switch m {
	case ModeManual, ModeScheduled, ModeYOLO:
		return true
	default:
		return false
	}
}

// ArchiveContext captures archive parameters for spec-sync-archive executions.
type ArchiveContext struct {
	ScenarioName   string   `json:"scenario_name"`
	ScenarioPath   string   `json:"scenario_path"`
	PresetOrCustom string   `json:"preset_or_custom,omitempty"`
	PreservePaths  []string `json:"preserve_paths,omitempty"`
	PreservePreset string   `json:"preserve_preset,omitempty"`
}

// Record is a persisted execution run record.
type Record struct {
	ExecutionID       string          `json:"execution_id"`
	BacklogKind       string          `json:"backlog_kind"`
	BacklogName       string          `json:"backlog_name"`
	PreviousStatus    string          `json:"previous_status,omitempty"`
	TaskID            string          `json:"task_id,omitempty"`
	RunID             string          `json:"run_id,omitempty"`
	Status            Status          `json:"status"`
	Mode              Mode            `json:"mode"`
	ScheduledAt       string          `json:"scheduled_at,omitempty"`
	StartedAt         string          `json:"started_at,omitempty"`
	FinishedAt        string          `json:"finished_at,omitempty"`
	FailureReason     string          `json:"failure_reason,omitempty"`
	StartedBy         string          `json:"started_by,omitempty"`
	Operation         string          `json:"operation,omitempty"`
	Force             bool            `json:"force,omitempty"`
	PromptTrace       *PromptTrace    `json:"prompt_trace,omitempty"`
	ArchiveContext    *ArchiveContext `json:"archive_context,omitempty"`
	ParentExecutionID string          `json:"parent_execution_id,omitempty"`
	FixupAttempt      int             `json:"fixup_attempt,omitempty"`
	Finalization      *Finalization   `json:"finalization,omitempty"`
	// Deprecated: migration-only fields preserved so legacy execution history can
	// be converted into the unified finalization model on read.
	LegacyReviewResult     *ReviewResult `json:"review_result,omitempty"`
	LegacyReviewJobID      string        `json:"review_job_id,omitempty"`
	LegacyReviewSkipReason string        `json:"review_skip_reason,omitempty"`
	LegacyReviewStartedAt  string        `json:"review_started_at,omitempty"`
	CreatedAt              string        `json:"created_at"`
	UpdatedAt              string        `json:"updated_at"`
}

// PromptTrace captures prompt details used to launch the execution.
type PromptTrace struct {
	Purpose        string `json:"purpose"`
	Prompt         string `json:"prompt"`
	PromptRevision string `json:"prompt_revision,omitempty"`
	UsedFallback   bool   `json:"used_fallback"`
	CapturedAt     string `json:"captured_at"`
}

// ReviewResult captures the outcome of a post-execution readiness review.
type ReviewResult struct {
	JobID          string            `json:"job_id"`
	Classification string            `json:"classification"` // ready, ready_with_notes, needs_work, not_assessable
	Dimensions     []ReviewDimension `json:"dimensions,omitempty"`
	Summary        string            `json:"summary"`
	ReviewedAt     string            `json:"reviewed_at"`
}

// ReviewDimension captures a single review dimension result.
type ReviewDimension struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // green, yellow, red, skipped
	Details string `json:"details,omitempty"`
}

// ProcessPreflight summarizes whether a backlog item can be processed safely.
type ProcessPreflight struct {
	BacklogKind              string                    `json:"backlog_kind"`
	BacklogName              string                    `json:"backlog_name"`
	Ready                    bool                      `json:"ready"`
	ArchivedRevival          bool                      `json:"archived_revival"`
	ResolvedTargetScenarioID string                    `json:"resolved_target_scenario_id,omitempty"`
	TargetScenarioExists     bool                      `json:"target_scenario_exists"`
	SuggestedOperation       string                    `json:"suggested_operation,omitempty"`
	SuggestedSteerProfileID  string                    `json:"suggested_steer_profile_id,omitempty"`
	BlockingReasons          []string                  `json:"blocking_reasons,omitempty"`
	BlockingQuestions        []ProcessBlockingQuestion `json:"blocking_questions,omitempty"`
}

// ProcessBlockingQuestion represents an unanswered critical question.
type ProcessBlockingQuestion struct {
	ID         string `json:"id,omitempty"`
	Importance string `json:"importance,omitempty"`
	Question   string `json:"question,omitempty"`
}

// CreateRequest creates an execution record.
type CreateRequest struct {
	BacklogKind  string `json:"backlog_kind"`
	BacklogName  string `json:"backlog_name"`
	Mode         Mode   `json:"mode"`
	DelaySeconds int64  `json:"delay_seconds,omitempty"`
	StartedBy    string `json:"started_by,omitempty"`
	Operation    string `json:"operation,omitempty"`
	Force        bool   `json:"force,omitempty"`
}

// Policy controls default execution behavior when callers do not provide mode/delay.
type Policy struct {
	DefaultMode         Mode  `json:"default_mode"`
	DefaultDelaySeconds int64 `json:"default_delay_seconds"`
	MaxFixupAttempts    int   `json:"max_fixup_attempts"`
	AutoFixup           bool  `json:"auto_fixup"`
}

// FollowUpRequest describes a user-initiated follow-up from a completed/failed execution.
type FollowUpRequest struct {
	ExecutionID  string `json:"execution_id"`
	FollowUpType string `json:"follow_up_type"` // fixup, followup, custom
	Context      string `json:"context,omitempty"`
	RunMode      string `json:"run_mode"` // continue, new
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
