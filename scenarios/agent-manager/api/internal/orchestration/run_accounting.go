// This file reads one run's metered usage for consumers that settle allowances.
package orchestration

import (
	"context"
	"fmt"
	"math"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agentcatalog"
)

type runBatchReader interface {
	GetMany(context.Context, []uuid.UUID) (map[uuid.UUID]*domain.Run, error)
}

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
	// NonCacheTokens excludes cache reads; CacheReadTokens is the remainder of
	// Tokens. Model names the producing model so readers can weight by tier.
	NonCacheTokens  int64
	CacheReadTokens int64
	Model           string
	WeightedTokens  float64
	WeightedPresent bool
	WeightedKnown   bool
	TokenCap        int64
	TokenCapMode    string
}

// RunAccounting reads a standalone run's terminal usage. A consumer that
// dispatched a run outside a workflow (for example, a Swarm goal run) has no
// workflow receipt; this is its equivalent.
func (o *Orchestrator) RunAccounting(ctx context.Context, runID uuid.UUID) (RunAccounting, error) {
	run, state, events, err := o.meteredRun(ctx, runID)
	if err != nil {
		return RunAccounting{}, err
	}
	out, err := runAccountingFromState(run, state)
	if err != nil {
		return out, err
	}
	return withTokenComponents(out, run, events, o.now()), nil
}

// RunAccountingBatch projects the same authoritative accounting as
// RunAccounting for a set of run identities, using the run and event owners'
// batch read seams. Missing runs have no result entry; an event or projection
// error fails the batch instead of returning a misleading partial total.
func (o *Orchestrator) RunAccountingBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Run, map[uuid.UUID]RunAccounting, error) {
	runs := make(map[uuid.UUID]*domain.Run, len(ids))
	accounts := make(map[uuid.UUID]RunAccounting, len(ids))
	if len(ids) == 0 {
		return runs, accounts, nil
	}
	runReader, ok := o.runs.(runBatchReader)
	if !ok {
		return nil, nil, fmt.Errorf("run repository does not support batch reads")
	}
	eventReader, ok := o.events.(event.BatchReader)
	if !ok {
		return nil, nil, fmt.Errorf("event store does not support batch reads")
	}
	var err error
	runs, err = runReader.GetMany(ctx, ids)
	if err != nil {
		return runs, accounts, err
	}
	policy := o.capPolicy()
	for id, run := range runs {
		if run == nil {
			continue
		}
		accounts[id] = RunAccounting{RunID: id, TokenCap: o.tokenCapFor(ctx, run, policy), TokenCapMode: string(policy.Mode)}
	}
	events, eventErrors, err := eventReader.GetMany(ctx, ids, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric, domain.EventTypeStatus, domain.EventTypeGoalStatusChanged}})
	if err != nil {
		return runs, accounts, err
	}
	now := o.now()
	for _, id := range ids {
		run := runs[id]
		if run == nil || eventErrors[id] != nil {
			continue
		}
		stream := events[id]
		state, err := meteredWorkflowChildState(run, stream, now)
		if err != nil {
			continue
		}
		out, err := runAccountingFromState(run, state)
		if err != nil {
			continue
		}
		account := withTokenComponents(out, run, stream, now)
		account.TokenCap = accounts[id].TokenCap
		account.TokenCapMode = accounts[id].TokenCapMode
		for _, evt := range stream {
			if _, ok := evt.Data.(*domain.UsageEventData); ok {
				weight, weightKnown := agentcatalog.TokenWeight(account.Model, policy.Weights)
				account.WeightedTokens = float64(account.NonCacheTokens) * weight
				account.WeightedPresent = true
				account.WeightedKnown = account.TokensKnown && weightKnown
				break
			}
		}
		accounts[id] = account
	}
	return runs, accounts, nil
}

// withTokenComponents splits the metered token total into cache reads and
// everything else, and names the model, from the run's usage events.
func withTokenComponents(out RunAccounting, run *domain.Run, events []*domain.RunEvent, now time.Time) RunAccounting {
	fact := invocationreadmodel.ProjectRun(run, events, now)
	out.CacheReadTokens = fact.CacheReadTokens
	out.NonCacheTokens = fact.TotalTokens - fact.CacheReadTokens
	out.Model = fact.Model
	return out
}

// meteredRun loads a run and its durable accounting events and applies the
// owner metering. Workflow child inspection and standalone accounting share it
// so the two can never read different evidence.
func (o *Orchestrator) meteredRun(ctx context.Context, runID uuid.UUID) (*domain.Run, workflowruntime.ChildState, []*domain.RunEvent, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, workflowruntime.ChildState{}, nil, err
	}
	events, err := o.meteredRunEvents(ctx, runID)
	if err != nil {
		return nil, workflowruntime.ChildState{}, nil, err
	}
	state, err := meteredWorkflowChildState(run, events, o.now())
	if err != nil {
		return nil, workflowruntime.ChildState{}, nil, err
	}
	return run, state, events, nil
}

// meteredRunEvents returns the complete stream needed by metering and its
// token-component projection. Keeping invocation status events in this one
// read prevents identical authoritative receipts in separate continuations
// from being deduplicated as though they belonged to one invocation.
func (o *Orchestrator) meteredRunEvents(ctx context.Context, runID uuid.UUID) ([]*domain.RunEvent, error) {
	return o.allRunEvents(ctx, runID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric, domain.EventTypeStatus, domain.EventTypeGoalStatusChanged}})
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
