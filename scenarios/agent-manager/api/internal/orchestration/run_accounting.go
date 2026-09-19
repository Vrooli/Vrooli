// This file reads one run's metered usage for consumers that settle allowances.
package orchestration

import (
	"context"
	"fmt"
	"math"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
)

// RunAccounting is one run's metered usage, projected from the same metering
// the workflow owner applies to its child runs. A consumer may settle a
// reservation only when Terminal, TokensKnown and ChargeMeasured all hold;
// anything less is an unresolved reservation, never zero usage.
type RunAccounting struct {
	RunID          uuid.UUID
	Terminal       bool
	Tokens         int64
	Turns          int64
	TokensKnown    bool
	ChargeMicroUSD int64
	ChargeMeasured bool
	// WallSeconds is conservative elapsed time from the durable start to end.
	WallSeconds int64
}

// RunAccounting reads a standalone run's terminal usage. A consumer that
// dispatched a run outside a workflow (for example, a Swarm goal run) has no
// workflow receipt; this is its equivalent.
func (o *Orchestrator) RunAccounting(ctx context.Context, runID uuid.UUID) (RunAccounting, error) {
	run, state, err := o.meteredRun(ctx, runID)
	if err != nil {
		return RunAccounting{}, err
	}
	return runAccountingFromState(run, state)
}

// meteredRun loads a run and its durable accounting events and applies the
// owner metering. Workflow child inspection and standalone accounting share it
// so the two can never read different evidence.
func (o *Orchestrator) meteredRun(ctx context.Context, runID uuid.UUID) (*domain.Run, workflowruntime.ChildState, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, workflowruntime.ChildState{}, err
	}
	events, err := o.allRunEvents(ctx, runID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric, domain.EventTypeStatus, domain.EventTypeGoalStatusChanged}})
	if err != nil {
		return nil, workflowruntime.ChildState{}, err
	}
	state, err := meteredWorkflowChildState(run, events, o.now())
	if err != nil {
		return nil, workflowruntime.ChildState{}, err
	}
	return run, state, nil
}

// runAccountingFromState projects metering onto the accounting contract.
// Terminal follows the workflow child view (a finished needs_review turn is
// done) and the domain's terminal statuses, so failed, cancelled and unknown
// runs settle too.
func runAccountingFromState(run *domain.Run, state workflowruntime.ChildState) (RunAccounting, error) {
	terminal := state.Terminal || run.Status.IsTerminal()
	out := RunAccounting{
		RunID:          run.ID,
		Terminal:       terminal,
		Tokens:         int64(state.Tokens),
		Turns:          int64(state.Turns),
		TokensKnown:    state.TokensKnown,
		ChargeMicroUSD: state.ChargeMicroUSD,
		ChargeMeasured: state.ChargeMeasured,
	}
	if !terminal || run.EndedAt == nil {
		return out, nil
	}
	start := run.CreatedAt
	if run.StartedAt != nil {
		start = *run.StartedAt
	}
	elapsed := run.EndedAt.Sub(start)
	if elapsed < 0 {
		return RunAccounting{}, fmt.Errorf("run %s ended before it started", run.ID)
	}
	out.WallSeconds = int64(math.Ceil(elapsed.Seconds()))
	return out, nil
}
