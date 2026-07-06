package execution

import (
	"encoding/json"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#domain-concepts
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INVARIANTS.md

// Status represents an execution lifecycle state. This is the *run* status —
// distinct from (and often unrelated to) the backlog item's status. A single
// backlog item may have many executions in any of these states over its
// lifetime.
//
// IMPORTANT: StatusNeedsFixup (run-level, "the finalization validator thinks
// this run has actionable issues — auto-fixup may retry") is NOT the same
// concept as backlog.StatusNeedsFollowup (item-level terminal state the user
// picks via review-decide). They share a "needs something more" vibe but:
//
//   - StatusNeedsFixup is set by the execution system from finalization,
//     and may cause an automatic re-run (see followup.go:FollowUp and
//     the AutoFixup governance policy).
//   - backlog.StatusNeedsFollowup is only set by a user through the
//     review-decide endpoint, and never by the execution system. It marks
//     the item as delivered-but-needs-more-work so it surfaces in list
//     views; any follow-up run is initiated by the user separately.
//
// Never translate one enum to the other without going through a deliberate
// handler (review-decide, FollowUp). See TestBacklogStatus_NotConflatedWithExecutionStatus.
type Status string

const (
	StatusPending  Status = "pending"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	// StatusNeedsReview is the run-level state "agent flagged this for a
	// human before continuing." Different from the backlog `in_review` /
	// `review_pending` statuses, which describe the item's lifecycle.
	StatusNeedsReview Status = "needs_review"
	StatusValidating  Status = "validating"
	// StatusNeedsFixup is the run-level state set by finalization when the
	// post-run validator detects actionable failures. See the type-level
	// comment for the distinction from backlog.StatusNeedsFollowup.
	StatusNeedsFixup Status = "needs_fixup"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCanceled   Status = "canceled"
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
	ExecutionID    string `json:"execution_id"`
	BacklogKind    string `json:"backlog_kind"`
	BacklogName    string `json:"backlog_name"`
	PreviousStatus string `json:"previous_status,omitempty"`
	TaskID         string `json:"task_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	Status         Status `json:"status"`
	Mode           Mode   `json:"mode"`
	// QueuedAt records when the item entered the pending queue (record
	// creation). Kept distinct from StartedAt so wait-in-queue
	// (StartedAt - QueuedAt) is measurable separately from execution time
	// (FinishedAt - StartedAt). Preserved verbatim through drain/start.
	QueuedAt          string          `json:"queued_at,omitempty"`
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
	// PreExecBaselines maps an affected scenario name to the GCT baseline
	// captured for it just before execution started. Finalization diffs each
	// of these against the post-execution working tree to separate regressions
	// this item caused from pre-existing failures. Keyed only by scenarios
	// declared in acceptance_allow (the sole pre-execution scope signal).
	PreExecBaselines map[string]string `json:"pre_exec_baselines,omitempty"`
	// EngagementHoldAt timestamps when this run's pre-merge Baseline Modes hold
	// was processed (shadow restore points opened from the actual diff, then the
	// overlay merge approved). It is the idempotency marker for the hold: the
	// poller may observe needs_review on several cycles before ApproveRun takes
	// effect, so a non-empty value tells processEngagementHold to skip re-opening
	// and re-approving. The engagements themselves are owned by the backlog item
	// (EngagementStore, keyed by ownerKeyFor), not the execution Record — so they
	// survive across the main run, every fixup, and the gap until review-decide.
	EngagementHoldAt string `json:"engagement_hold_at,omitempty"`
	// ManuallyAccepted is set when the user overrode a failed/canceled/
	// needs_fixup execution by manually marking the backlog item completed.
	// Stats treat manually-accepted runs as successful so the Agent tab
	// reflects end-to-end success, not the agent's own verdict.
	ManuallyAccepted       bool   `json:"manually_accepted,omitempty"`
	AcceptedBy             string `json:"accepted_by,omitempty"`
	AcceptedReason         string `json:"accepted_reason,omitempty"`
	AcceptedPreviousStatus Status `json:"accepted_previous_status,omitempty"`
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
	// ForceableBlockingReasons block the queue but can be overridden with
	// force=true (e.g. the fix-before-feature gate in "block" mode). Kept
	// separate from BlockingReasons (which are structural / non-forceable) so
	// callers can render forceability correctly.
	ForceableBlockingReasons []string `json:"forceable_blocking_reasons,omitempty"`
	// Advisories are non-blocking messages surfaced to the caller (e.g. the
	// fix-before-feature gate in "suggest" mode). They never affect Ready.
	Advisories []string `json:"advisories,omitempty"`
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

// GoalReadyItem is a backlog item that is ready to run and belongs to an active
// goal's closure, carried by the continuous auto-enqueue drain. GoalPriority is
// the highest priority among the goals it belongs to.
type GoalReadyItem struct {
	Kind         string
	Name         string
	GoalPriority int
}

// GovernanceSettings controls concurrency, queue depth, circuit breaker, and cost caps.
type GovernanceSettings struct {
	// LaneLimits caps simultaneous tracked agent activity by phase-kind
	// lane. Keys are lane names matching agentactivity.Lane /
	// operatingmode.PhaseKind: "investigate", "execute", "review",
	// "reconcile". Zero or missing keys fall back to the per-lane default
	// in DefaultGovernanceSettings — see docs/internal/SEAMS.md
	// "Concurrency Lane Boundary" for the full contract.
	LaneLimits                    map[string]int `json:"lane_limits"`
	MaxQueueDepth                 int            `json:"max_queue_depth"`
	CircuitBreakerThreshold       int            `json:"circuit_breaker_threshold"`
	CircuitBreakerCooldownMinutes int            `json:"circuit_breaker_cooldown_minutes"`
	ExecutionCostCapPerRun        float64        `json:"execution_cost_cap_per_run"`
	CostPerTurnEstimate           float64        `json:"cost_per_turn_estimate"`
	AgentMaxTurns                 int            `json:"agent_max_turns"`

	// FixBeforeFeature controls the fix-before-feature gate: "off", "suggest"
	// (default), or "block". FixBeforeFeatureDiscovery enables async on-demand
	// readiness discovery when a feature item targets a scenario with no known
	// open remediation work. See fix_before_feature.go.
	FixBeforeFeature          string `json:"fix_before_feature"`
	FixBeforeFeatureDiscovery bool   `json:"fix_before_feature_discovery"`
}

// DefaultGovernanceSettings returns safe defaults for governance settings.
func DefaultGovernanceSettings() GovernanceSettings {
	return GovernanceSettings{
		LaneLimits: map[string]int{
			"investigate": 6,
			"execute":     3,
			"review":      8,
			"reconcile":   2,
		},
		MaxQueueDepth:                 50,
		CircuitBreakerThreshold:       3,
		CircuitBreakerCooldownMinutes: 60,
		ExecutionCostCapPerRun:        0,
		CostPerTurnEstimate:           0.10,
		AgentMaxTurns:                 600,
		FixBeforeFeature:              FixBeforeFeatureSuggest,
		FixBeforeFeatureDiscovery:     false,
	}
}

// LaneStatus reports utilization for one phase-kind lane.
type LaneStatus struct {
	Lane     string `json:"lane"`
	Active   int    `json:"active"`
	Capacity int    `json:"capacity"`
	Queue    int    `json:"queue"`
}

// GovernanceStatusResponse contains governance state for the overview endpoint.
type GovernanceStatusResponse struct {
	// Lanes carries per-phase-kind utilization (active / capacity / queue)
	// in the canonical Investigate → Execute → Review → Reconcile order.
	// Always populated for every canonical lane, even when active=0.
	Lanes []LaneStatus `json:"lanes"`
	// ActiveExecutions sums active counts across the four lanes (legacy
	// compatibility for callers that have not migrated to per-lane).
	ActiveExecutions    int      `json:"active_executions"`
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

// RetryRequest describes a user-initiated retry of a terminal execution.
// Retry creates a *new* execution Record parented to ExecutionID, copying the
// scope verbatim — no derived feedback, no follow-up note. The parent Record
// is never mutated; its logs, finalization, and outcome remain intact for
// audit and stats.
//
// Note flows through to the new run's prompt as optional retry context (e.g.,
// "fixed agent-manager hesitation bug"). It is purely informational and does
// not influence scope.
type RetryRequest struct {
	ExecutionID string `json:"execution_id"`
	Note        string `json:"note,omitempty"`
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
