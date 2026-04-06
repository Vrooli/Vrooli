package execution

import (
	"encoding/json"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#domain-concepts
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INVARIANTS.md

// Status represents an execution lifecycle state.
type Status string

const (
	StatusPending     Status = "pending"
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
	ModeManual Mode = "manual"
	ModeYOLO   Mode = "yolo"
)

// ValidateMode returns true if m is a known execution mode.
func ValidateMode(m Mode) bool {
	switch m {
	case ModeManual, ModeYOLO:
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
	ExperimentID   string `json:"experiment_id,omitempty"`
	VariantID      string `json:"variant_id,omitempty"`
}

// ReviewResult captures the outcome of a post-execution readiness review.
type ReviewResult struct {
	JobID          string            `json:"job_id"`
	Classification string            `json:"classification"` // ready, ready_with_notes, needs_work, not_assessable
	Dimensions     []ReviewDimension `json:"dimensions,omitempty"`
	RawDimensions  json.RawMessage   `json:"raw_dimensions,omitempty"` // Full GCT dimensions JSON for review agent consumption
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
	BacklogKind string `json:"backlog_kind"`
	BacklogName string `json:"backlog_name"`
	Mode        Mode   `json:"mode"`
	StartedBy   string `json:"started_by,omitempty"`
	Operation   string `json:"operation,omitempty"`
	Force       bool   `json:"force,omitempty"`
}

// Policy controls default execution behavior when callers do not provide mode/delay.
type Policy struct {
	DefaultMode        Mode `json:"default_mode"`
	MaxFixupAttempts   int  `json:"max_fixup_attempts"`
	AutoFixup          bool `json:"auto_fixup"`
	ReviewAgentEnabled bool `json:"review_agent_enabled"`
}

// GovernanceSettings controls concurrency, queue depth, circuit breaker, and cost caps.
type GovernanceSettings struct {
	MaxConcurrentExecutions       int     `json:"max_concurrent_executions"`
	MaxQueueDepth                 int     `json:"max_queue_depth"`
	CircuitBreakerThreshold       int     `json:"circuit_breaker_threshold"`
	CircuitBreakerCooldownMinutes int     `json:"circuit_breaker_cooldown_minutes"`
	ExecutionCostCapPerRun        float64 `json:"execution_cost_cap_per_run"`
	CostPerTurnEstimate           float64 `json:"cost_per_turn_estimate"`
	AgentMaxTurns                 int     `json:"agent_max_turns"`
}

// DefaultGovernanceSettings returns safe defaults for governance settings.
func DefaultGovernanceSettings() GovernanceSettings {
	return GovernanceSettings{
		MaxConcurrentExecutions:       3,
		MaxQueueDepth:                 50,
		CircuitBreakerThreshold:       3,
		CircuitBreakerCooldownMinutes: 60,
		ExecutionCostCapPerRun:        0,
		CostPerTurnEstimate:           0.10,
		AgentMaxTurns:                 60,
	}
}

// GovernanceStatusResponse contains governance state for the overview endpoint.
type GovernanceStatusResponse struct {
	ActiveExecutions    int      `json:"active_executions"`
	MaxConcurrent       int      `json:"max_concurrent"`
	QueueDepth          int      `json:"queue_depth"`
	MaxQueueDepth       int      `json:"max_queue_depth"`
	CircuitBrokenItems  []string `json:"circuit_broken_items"`
	EstimatedQueuedCost float64  `json:"estimated_queued_cost"`
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
