package execution

import (
	"context"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
)

// AgentSpawner spawns agent-manager sessions for execution runs.
type AgentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
}

// RunInspector retrieves the current state of an agent run.
type RunInspector interface {
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
}

// RunStopper cancels a running agent session.
type RunStopper interface {
	StopRun(ctx context.Context, runID string) error
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

// ReviewServiceIntegration is the subset of the review service that finalization
// needs to trigger evidence gathering. Uses a callback signature to avoid
// import cycles between the execution and review packages.
type ReviewServiceIntegration interface {
	StartReviewForExecution(ctx context.Context, executionID, backlogKind, backlogName, itemTitle, itemDir string, affectedScenarios []string, changedPathsByScenario map[string][]string, gctResultsJSON string) error
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
