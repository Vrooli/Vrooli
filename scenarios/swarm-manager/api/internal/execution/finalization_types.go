package execution

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/agentmanager"
)

// FinalizationStatus represents the lifecycle state of a finalization step
// (restart, health check, review, or the finalization as a whole).
type FinalizationStatus string

const (
	FinalizationStatusPending   FinalizationStatus = "pending"
	FinalizationStatusRunning   FinalizationStatus = "running"
	FinalizationStatusCompleted FinalizationStatus = "completed"
	FinalizationStatusSkipped   FinalizationStatus = "skipped"
	FinalizationStatusFailed    FinalizationStatus = "failed"
)

var ErrInvalidFinalizationStatusTransition = fmt.Errorf("invalid finalization status transition")

var finalizationStatusTransitions = map[FinalizationStatus]map[FinalizationStatus]struct{}{
	FinalizationStatusPending: {
		FinalizationStatusRunning: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {},
	},
	FinalizationStatusRunning: {
		FinalizationStatusCompleted: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {},
	},
}

var scenarioRestartStatusTransitions = map[FinalizationStatus]map[FinalizationStatus]struct{}{
	FinalizationStatusPending: {FinalizationStatusRunning: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {}},
	FinalizationStatusRunning: {FinalizationStatusCompleted: {}, FinalizationStatusFailed: {}},
	FinalizationStatusFailed:  {FinalizationStatusRunning: {}},
}

var scenarioHealthStatusTransitions = map[FinalizationStatus]map[FinalizationStatus]struct{}{
	FinalizationStatusPending: {FinalizationStatusRunning: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {}},
	FinalizationStatusRunning: {FinalizationStatusCompleted: {}, FinalizationStatusFailed: {}},
	FinalizationStatusFailed:  {FinalizationStatusRunning: {}},
}

var scenarioReviewStatusTransitions = map[FinalizationStatus]map[FinalizationStatus]struct{}{
	FinalizationStatusPending: {FinalizationStatusRunning: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {}},
	FinalizationStatusRunning: {FinalizationStatusCompleted: {}, FinalizationStatusFailed: {}},
}

// Valid reports whether status belongs to the persisted finalization lifecycle.
func (status FinalizationStatus) Valid() bool {
	switch status {
	case FinalizationStatusPending, FinalizationStatusRunning, FinalizationStatusCompleted, FinalizationStatusSkipped, FinalizationStatusFailed:
		return true
	default:
		return false
	}
}

// CanTransitionTo reports whether the finalization lifecycle allows next.
func (status FinalizationStatus) CanTransitionTo(next FinalizationStatus) bool {
	if status == next {
		return status.Valid()
	}
	_, allowed := finalizationStatusTransitions[status][next]
	return allowed
}

// TransitionStatus is the sole mutation owner for an existing Finalization.
// Starting a new finalization (including an explicit review rerun) initializes
// a fresh value at Pending; recovery and ordinary progress use this method.
func (finalization *Finalization) TransitionStatus(next FinalizationStatus) error {
	if finalization == nil || !finalization.Status.Valid() || !next.Valid() {
		current := FinalizationStatus("<nil>")
		if finalization != nil {
			current = finalization.Status
		}
		return fmt.Errorf("%w: unknown state %q -> %q", ErrInvalidFinalizationStatusTransition, current, next)
	}
	if finalization.Status == next {
		return nil
	}
	if !finalization.Status.CanTransitionTo(next) {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidFinalizationStatusTransition, finalization.Status, next)
	}
	finalization.Status = next
	return nil
}

const (
	FinalizationPhaseScopeDetection    = "scope_detection"
	FinalizationPhaseRestarting        = "restarting"
	FinalizationPhaseHealthCheck       = "health_check"
	FinalizationPhaseBaselineDiff      = "baseline_diff"
	FinalizationPhaseReviewing         = "reviewing"
	FinalizationPhaseEvidenceGathering = "evidence_gathering"
	FinalizationPhaseCompleted         = "completed"
	FinalizationPhaseSkipped           = "skipped"
	FinalizationPhaseFailed            = "failed"
)

const (
	FinalizationScopeNone                        = "none"
	FinalizationScopeSandboxDiff                 = "sandbox_diff"
	FinalizationScopeAcceptanceAllow             = "acceptance_allow"
	FinalizationScopeSandboxDiffPlusAcceptance   = "sandbox_diff_plus_acceptance_allow"
	FinalizationAggregateReady                   = "ready"
	FinalizationAggregateReadyWithNotes          = "ready_with_notes"
	FinalizationAggregateNeedsWork               = "needs_work"
	FinalizationAggregateNotAssessable           = "not_assessable"
	FinalizationAggregateSkipped                 = "skipped"
	finalizationWarningScopeDiffUnavailable      = "scope_diff_unavailable"
	finalizationWarningScopeSharedPathBroadening = "scope_shared_path_broadening"
	finalizationWarningRestartRetry              = "restart_retry"
	finalizationWarningHealthRetry               = "health_retry"
	finalizationWarningHealthSchemaInvalid       = "health_schema_invalid"
	finalizationWarningReviewSkipped             = "review_skipped"
	finalizationWarningFinalizationInfra         = "finalization_infrastructure"
	finalizationWarningBaselineScopeExpanded     = "baseline_scope_expanded_no_baseline"
	finalizationWarningBaselineDiffFailed        = "baseline_diff_failed"
	finalizationWarningBaselineRegression        = "baseline_regression"
	finalizationWarningReviewAgentFailed         = "review_agent_failed"
	finalizationWarningEvidenceSkippedDisabled   = "evidence_skipped_disabled"
	finalizationWarningEvidenceSkippedPolicyErr  = "evidence_skipped_policy_error"
	finalizationWarningSelfRestartSkipped        = "self_restart_skipped"
)

// ScenarioLifecycle restarts affected scenarios after execution completion.
type ScenarioLifecycle interface {
	Restart(ctx context.Context, name string) error
}

// ScenarioHealthChecker probes scenario health using the standard Vrooli status
// contract.
type ScenarioHealthChecker interface {
	Check(ctx context.Context, name string) (ScenarioHealthSnapshot, error)
}

// RunDiffer resolves changed files for sandboxed agent-manager runs.
type RunDiffer interface {
	GetRunDiff(ctx context.Context, runID string) (agentmanager.RunDiff, error)
}

// ScenarioHealthSnapshot is the execution package's neutral view of scenario
// health.
type ScenarioHealthSnapshot struct {
	ScenarioStatus string
	HealthStatus   string
	SchemaValid    bool
	Healthy        bool
	Details        string
	CheckedAt      string
}

// Finalization captures the full post-run orchestration state for an
// execution.
type Finalization struct {
	Eligible                bool                   `json:"eligible"`
	Status                  FinalizationStatus     `json:"status,omitempty"`
	Phase                   string                 `json:"phase,omitempty"`
	ScopeSource             string                 `json:"scope_source,omitempty"`
	SkipReason              string                 `json:"skip_reason,omitempty"`
	StartedAt               string                 `json:"started_at,omitempty"`
	CompletedAt             string                 `json:"completed_at,omitempty"`
	Warnings                []FinalizationWarning  `json:"warnings,omitempty"`
	AffectedScenarios       []string               `json:"affected_scenarios,omitempty"`
	AggregateClassification string                 `json:"aggregate_classification,omitempty"`
	AggregateSummary        string                 `json:"aggregate_summary,omitempty"`
	Scenarios               []ScenarioFinalization `json:"scenarios,omitempty"`
}

// FinalizationWarning captures a non-fatal issue encountered while running the
// finalization flow.
type FinalizationWarning struct {
	Code         string `json:"code,omitempty"`
	ScenarioName string `json:"scenario_name,omitempty"`
	Message      string `json:"message,omitempty"`
	Retryable    bool   `json:"retryable,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// ScenarioFinalization captures the restart, health, and review work for one
// affected scenario.
type ScenarioFinalization struct {
	ScenarioName string             `json:"scenario_name,omitempty"`
	ChangedPaths []string           `json:"changed_paths,omitempty"`
	Restart      RestartResult      `json:"restart,omitempty"`
	Health       HealthCheckResult  `json:"health,omitempty"`
	Review       ScenarioReviewStep `json:"review,omitempty"`
	// BaselineDiff holds the before/after GCT baseline comparison for this
	// scenario when a pre-execution baseline was captured. Nil when the
	// feature is off or the scenario was touched outside acceptance_allow
	// (no pre-exec baseline to compare against — recorded not_comparable).
	BaselineDiff *BaselineDiffResult `json:"baseline_diff,omitempty"`
}

// RestartResult captures restart attempts for one scenario.
type RestartResult struct {
	Status     FinalizationStatus `json:"status,omitempty"`
	Attempts   int                `json:"attempts,omitempty"`
	LastError  string             `json:"last_error,omitempty"`
	StartedAt  string             `json:"started_at,omitempty"`
	FinishedAt string             `json:"finished_at,omitempty"`
}

// TransitionStatus owns the durable per-scenario restart attempt lifecycle.
func (restart *RestartResult) TransitionStatus(next FinalizationStatus) error {
	if restart == nil {
		return transitionScenarioStepStatus("restart", nil, next, nil)
	}
	return transitionScenarioStepStatus("restart", &restart.Status, next, scenarioRestartStatusTransitions)
}

// HealthCheckResult captures the structured health outcome for one scenario.
type HealthCheckResult struct {
	Status         FinalizationStatus `json:"status,omitempty"`
	ScenarioStatus string             `json:"scenario_status,omitempty"`
	HealthStatus   string             `json:"health_status,omitempty"`
	SchemaValid    bool               `json:"schema_valid"`
	Details        string             `json:"details,omitempty"`
	CheckedAt      string             `json:"checked_at,omitempty"`
}

// TransitionStatus owns the durable per-scenario health-check lifecycle.
func (health *HealthCheckResult) TransitionStatus(next FinalizationStatus) error {
	if health == nil {
		return transitionScenarioStepStatus("health", nil, next, nil)
	}
	return transitionScenarioStepStatus("health", &health.Status, next, scenarioHealthStatusTransitions)
}

// ScenarioReviewStep captures one scenario's review job and result.
type ScenarioReviewStep struct {
	Status     FinalizationStatus `json:"status,omitempty"`
	JobID      string             `json:"job_id,omitempty"`
	SkipReason string             `json:"skip_reason,omitempty"`
	Result     *ReviewResult      `json:"result,omitempty"`
}

// TransitionStatus owns the durable per-scenario review lifecycle.
func (review *ScenarioReviewStep) TransitionStatus(next FinalizationStatus) error {
	if review == nil {
		return transitionScenarioStepStatus("review", nil, next, nil)
	}
	return transitionScenarioStepStatus("review", &review.Status, next, scenarioReviewStatusTransitions)
}

func transitionScenarioStepStatus(name string, current *FinalizationStatus, next FinalizationStatus, allowed map[FinalizationStatus]map[FinalizationStatus]struct{}) error {
	if current == nil || !current.Valid() || !next.Valid() {
		from := FinalizationStatus("<nil>")
		if current != nil {
			from = *current
		}
		return fmt.Errorf("%w: %s unknown state %q -> %q", ErrInvalidFinalizationStatusTransition, name, from, next)
	}
	if *current == next {
		return nil
	}
	if _, ok := allowed[*current][next]; !ok {
		return fmt.Errorf("%w: %s %q -> %q", ErrInvalidFinalizationStatusTransition, name, *current, next)
	}
	*current = next
	return nil
}

type finalizationScope struct {
	source                 string
	affectedScenarios      []string
	changedPathsByScenario map[string][]string
	sandboxID              string
	warnings               []FinalizationWarning
}

func isFinalizationEligible(record Record) bool {
	if record.ArchiveContext != nil {
		return false
	}

	runType := strings.ToLower(strings.TrimSpace(record.effectiveRunType()))
	switch runType {
	case "", "process", "fixup", "followup", "custom":
		return true
	default:
		return false
	}
}

func (r Record) effectiveRunType() string {
	if r.PromptTrace != nil && strings.TrimSpace(r.PromptTrace.Purpose) != "" {
		return strings.ToLower(strings.TrimSpace(r.PromptTrace.Purpose))
	}
	switch strings.ToLower(strings.TrimSpace(r.Operation)) {
	case "fixup", "followup", "custom":
		return strings.ToLower(strings.TrimSpace(r.Operation))
	default:
		return "process"
	}
}
