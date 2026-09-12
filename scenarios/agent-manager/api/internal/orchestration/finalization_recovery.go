package orchestration

import (
	"context"
	"fmt"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/phases"
)

// recoverFinalization retries only post-run effects with the persisted origin
// and policy. It neither resumes the transcript nor launches another agent.
// The workspace owner must verify live or archived evidence before this run
// can lose its failed-finalization warning.
func (o *Orchestrator) recoverFinalization(ctx context.Context, run *domain.Run) (*RecoverResult, error) {
	if run.RunMode != domain.RunModeSandboxed || run.SandboxID == nil || o.sandbox == nil {
		return nil, fmt.Errorf("finalization recovery requires the original sandbox and its owner")
	}
	cfg := phases.EffectiveSandboxConfig(run)
	if cfg == nil || cfg.ManualReview || !cfg.GetAutoApply() {
		return nil, fmt.Errorf("persisted sandbox policy does not authorize automatic finalization recovery")
	}
	outcome := domain.ContractRunOutcomeSuccess
	switch run.Status {
	case domain.RunStatusFailed:
		outcome = domain.ContractRunOutcomeFailure
	case domain.RunStatusCancelled:
		outcome = domain.ContractRunOutcomeCancelled
	case domain.RunStatusComplete:
	default:
		return nil, fmt.Errorf("run status %s requires its review workflow", run.Status)
	}
	if outcome != domain.ContractRunOutcomeSuccess && !cfg.GetApplyOnFailure() {
		return nil, fmt.Errorf("persisted sandbox policy disables finalization for this outcome")
	}
	levers := o.runLevers()
	recoveryCtx, cancel := context.WithTimeout(ctx, levers.Heartbeat.TeardownTimeout)
	defer cancel()
	cost := 0.0
	if run.Summary != nil {
		cost = run.Summary.CostEstimate
	}
	deps := phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: levers, WorkspaceSandbox: o.workspaceSandbox, Clock: o.now}
	phases.ApplyAtRunEnd(recoveryCtx, phases.ApplyAtRunEndInput{Deps: deps, Run: run, SandboxID: run.SandboxID, Sandbox: o.sandbox, Outcome: outcome, Cost: cost})
	if err := o.runs.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("persist recovered finalization: %w", err)
	}
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}
	if run.FinalizationStatus != domain.RunFinalizationStatusSucceeded {
		return nil, fmt.Errorf("sandbox finalization recovery failed: %s", run.FinalizationError)
	}
	// Only verified success releases the retained workspace. No runner executes
	// and the original terminal result, usage and end boundary remain unchanged.
	phases.ApplySandboxLifecycle(recoveryCtx, phases.ApplySandboxLifecycleInput{Deps: deps, Run: run, SandboxID: run.SandboxID, Sandbox: o.sandbox, Event: phases.LifecycleEventForStatus(run.Status), Reason: "finalization recovered"})
	o.nudgeWorkflowForRun(run.ID)
	return &RecoverResult{Run: run, Recovered: true, Message: "recovered sandbox finalization under the original run; execution was not repeated"}, nil
}
