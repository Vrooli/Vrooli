package execution

import (
	"context"
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

const (
	FinalizationPhaseScopeDetection    = "scope_detection"
	FinalizationPhaseRestarting        = "restarting"
	FinalizationPhaseHealthCheck       = "health_check"
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
	finalizationWarningHealthChecksMissing       = "health_checks_missing"
	finalizationWarningReviewSkipped             = "review_skipped"
	finalizationWarningFinalizationInfra         = "finalization_infrastructure"
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
}

// RestartResult captures restart attempts for one scenario.
type RestartResult struct {
	Status     FinalizationStatus `json:"status,omitempty"`
	Attempts   int                `json:"attempts,omitempty"`
	LastError  string             `json:"last_error,omitempty"`
	StartedAt  string             `json:"started_at,omitempty"`
	FinishedAt string             `json:"finished_at,omitempty"`
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

// ScenarioReviewStep captures one scenario's review job and result.
type ScenarioReviewStep struct {
	Status     FinalizationStatus `json:"status,omitempty"`
	JobID      string             `json:"job_id,omitempty"`
	SkipReason string             `json:"skip_reason,omitempty"`
	Result     *ReviewResult      `json:"result,omitempty"`
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
