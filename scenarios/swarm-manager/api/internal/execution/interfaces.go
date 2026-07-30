package execution

import (
	"context"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/review"
)

// AgentManagerAvailability probes whether agent-manager is reachable. Used as
// a preflight for user-initiated retry / follow-up; actual spawning happens
// exclusively through the operation runner (OperationStarter).
type AgentManagerAvailability interface {
	IsEnabled() bool
}

// RunStopper cancels a running agent session.
type RunStopper interface {
	StopRun(ctx context.Context, runID string) error
}

// RunApprover releases a run held at needs_review, merging its sandbox overlay
// into the working tree. The Baseline Modes pre-merge hold calls this after the
// shadow restore point has captured the clean working tree.
type RunApprover interface {
	ApproveRun(ctx context.Context, runID, actor, commitMsg string) error
}

// RunContinuer sends a follow-up message to a running agent session.
type RunContinuer interface {
	ContinueRun(ctx context.Context, runID string, message string) error
}

// Archiver performs scenario archive operations after spec-sync completes.
type Archiver interface {
	ArchiveScenario(ctx context.Context, ac ArchiveContext) error
}

// PolicyProvider reads execution policy defaults from the unified settings store.
type PolicyProvider interface {
	LoadPolicy() (Policy, error)
}

// ReviewThresholdsProvider reads review threshold settings.
type ReviewThresholdsProvider interface {
	LoadReviewThresholds() (*ReviewThresholds, error)
}

// GovernanceProvider reads governance settings from the unified settings store.
type GovernanceProvider interface {
	LoadGovernance() (GovernanceSettings, error)
}

// AutoFilerWaker receives feature-queue hints from the fix-before-feature gate.
// The gate remains advisory/blocking only; this wake path just asks the
// governed background auto-filer to run early when feature-pending strategy is
// enabled.
type AutoFilerWaker interface {
	WakeAutoFiler()
}

// GoalPriorityProvider supplies per-item goal priority so the drain comparator
// can prefer items in higher-priority goals. Optional: when unset, the drain
// falls back to pure FIFO by CreatedAt (behavior-preserving for ungoaled work).
type GoalPriorityProvider interface {
	// ItemGoalPriorities maps a backlog item ref ("<kind>/<name>") to the
	// highest priority among active goals whose closure contains it. Items
	// absent from the map are ungoaled.
	ItemGoalPriorities() (map[string]int, error)
}

// GoalReadyProvider lists ready backlog items across active goals for the
// continuous goal-directed auto-enqueue drain, highest goal priority first.
// Optional: when unset, continuous drain enqueues nothing.
type GoalReadyProvider interface {
	ReadyGoalItems() ([]GoalReadyItem, error)
}

// AutoDrainProvider reports whether continuous goal-directed auto-enqueue is
// enabled. Optional: when unset, continuous drain is off (the D4 default).
type AutoDrainProvider interface {
	AutoDrainEnabled() bool
}

// ReviewServiceIntegration is the subset of the review service that finalization
// needs to trigger evidence gathering. Uses a callback signature to avoid
// import cycles between the execution and review packages.
type ReviewServiceIntegration interface {
	StartReviewForExecution(ctx context.Context, executionID, backlogKind, backlogName, itemTitle, itemDescription, itemDir string, acceptanceCriteria any, machineEvidence []review.EvidenceItem, affectedScenarios []string, changedPathsByScenario map[string][]string, gctResultsJSON, baselineDiffJSON string) error
	// RecordUnavailableReview writes a synthetic terminal review round when no
	// review agent ran (disabled or spawn failure) so the review surface can
	// explain why no evidence exists for an item routed straight to
	// review_pending. Best-effort; must not block finalization.
	RecordUnavailableReview(kind, name, executionID, reason string) error
}

// ActivityLaneReader exposes per-lane active counts from the agentactivity
// store so execution.GovernanceStatus can render the four-lane utilization
// view without execution importing agentactivity at the type level (the
// concrete implementation in agentactivity.Service satisfies this seam).
type ActivityLaneReader interface {
	LaneActiveCounts() (map[agentactivity.Lane]int, error)
}

// EventLogger records execution state-change events for analytics.
type EventLogger interface {
	EmitExecutionCreated(execID, backlogKind, backlogName, mode string)
	EmitExecutionStatusChanged(execID, from, to string)
	EmitExecutionCompleted(execID string, durationSecs float64, hadFixups bool)
	EmitExecutionFailed(execID, reason string, durationSecs float64)
	EmitExecutionCanceled(execID, reason string)
	EmitExecutionManuallyAccepted(execID, acceptedBy, reason, previousStatus string)
	EmitExecutionViewed(execID string)
}
