package autosteer

import (
	"context"
	"log"

	"github.com/ecosystem-manager/api/pkg/autosteer/gameguard"
)

// RunDiffProvider is the anti-gaming diff seam: given an agent-manager run ID it
// returns that run's code-level diff in the classifier's shape. Production wires
// an agent-manager-backed adapter (tracking-mode runs carry a diff); tests
// inject a fake. An in-place run with no sandbox yields an empty diff.
type RunDiffProvider interface {
	GetRunDiff(ctx context.Context, runID string) (gameguard.Diff, error)
}

// SetDiffProvider wires (or clears) the anti-gaming diff seam. nil disables
// gaming detection (fail-open). Returns the orchestrator for chaining.
func (o *ExecutionOrchestrator) SetDiffProvider(p RunDiffProvider) *ExecutionOrchestrator {
	o.diffProvider = p
	return o
}

// RecordRunID stashes the agent-manager run ID of the run that just completed for
// a task. The next EvaluateIteration consumes it to fetch the run's diff for
// anti-gaming classification. Empty IDs are ignored.
func (o *ExecutionOrchestrator) RecordRunID(taskID, runID string) {
	if o == nil || taskID == "" || runID == "" {
		return
	}
	o.costMu.Lock()
	defer o.costMu.Unlock()
	o.pendingRunID[taskID] = runID
}

// takeRunID pops the recorded run ID for a task (empty if none).
func (o *ExecutionOrchestrator) takeRunID(taskID string) string {
	o.costMu.Lock()
	defer o.costMu.Unlock()
	id := o.pendingRunID[taskID]
	delete(o.pendingRunID, taskID)
	return id
}

// classifyGaming fetches the just-completed run's code-level diff and runs the
// anti-gaming classifier over it. It fails open: a nil provider, a missing run
// ID (in-place run / no sandbox), or any fetch error yields a clean verdict, so
// gaming detection never blocks or distorts the loop on infrastructure trouble.
func (o *ExecutionOrchestrator) classifyGaming(ctx context.Context, taskID string) gameguard.Result {
	runID := o.takeRunID(taskID)
	if o.diffProvider == nil || runID == "" {
		return gameguard.Result{}
	}
	diff, err := o.diffProvider.GetRunDiff(ctx, runID)
	if err != nil {
		log.Printf("Auto Steer: failed to fetch run diff for gaming check (non-fatal): %v", err)
		return gameguard.Result{}
	}
	return gameguard.Classify(diff)
}

// gamingTrace renders the classifier verdict as the trace's gaming_cause token:
// "gamed:..." for a penalized iteration, "flagged-for-review" for an ambiguous
// one, or "" for clean.
func gamingTrace(r gameguard.Result) string {
	if r.Gamed {
		return r.CauseString()
	}
	if r.FlaggedForReview {
		return "flagged-for-review"
	}
	return ""
}
