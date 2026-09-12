package orchestration

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
)

// Reuse the bounded, deduplicating workflow driver. Event persistence precedes
// the nudge; periodic recovery remains the backstop for a lost notification.
func (o *Orchestrator) nudgeWorkflowUsage(runID uuid.UUID, evt *domain.RunEvent) {
	if evt == nil || evt.EventType != domain.EventTypeMetric || o.workflowNudger == nil || o.workflowExecutions == nil {
		return
	}
	id, err := o.workflowExecutions.ExecutionIDForRun(context.Background(), runID)
	if err != nil {
		obs.Component("workflow-nudge").Warn("resolve owning execution for usage failed", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
		return
	}
	if id != uuid.Nil {
		o.workflowNudger.Enqueue(id)
	}
}

// InspectMetered reads durable provider usage, not the completion-only summary.
// Use the same invocation boundaries and deduplication rules as owner
// accounting. Continuations reuse the Run ID and its cumulative event stream.
func (l workflowChildLauncher) InspectMetered(ctx context.Context, id uuid.UUID) (workflowruntime.ChildState, error) {
	run, err := l.o.GetRun(ctx, id)
	if err != nil {
		return workflowruntime.ChildState{}, err
	}
	events, err := l.o.allRunEvents(ctx, id, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric, domain.EventTypeStatus, domain.EventTypeGoalStatusChanged}})
	if err != nil {
		return workflowruntime.ChildState{}, err
	}
	return meteredWorkflowChildState(run, events, l.o.now())
}

func meteredWorkflowChildState(run *domain.Run, events []*domain.RunEvent, now time.Time) (workflowruntime.ChildState, error) {
	state := childStateFromRun(run)
	if run != nil && run.EndedAt != nil {
		state.TerminalObservedAt = *run.EndedAt
	}
	state.Tokens, state.Turns, state.ChargeMicroUSD = 0, 0, 0
	state.ChargeMeasured = false
	terminalAuthority := false
	latestUsage, latestCharge := false, false
	priorInvocationsComplete := true
	chargeObserved, chargeComplete := false, true
	observeCharge := func(charge *domain.ChargeEventData) error {
		if charge == nil {
			return nil
		}
		chargeObserved = true
		latestCharge = true
		if charge.AmountMicroUSD == nil {
			chargeComplete = false
			return nil
		}
		if *charge.AmountMicroUSD < 0 {
			return fmt.Errorf("negative provider charge for run %s", run.ID)
		}
		switch charge.Basis {
		case domain.ChargeBasisMetered:
			// Preserve the existing owner metered-charge contract.
		case domain.ChargeBasisSubscription, domain.ChargeBasisLocal:
			// Known zero marginal charge is backed by the immutable run
			// snapshot and an explicit charge event, never a cost estimate.
			if run.Billing.EffectiveBasis() != charge.Basis || *charge.AmountMicroUSD != 0 {
				chargeComplete = false
			}
		default:
			chargeComplete = false
		}
		return nil
	}
	var observations []time.Time
	for _, evt := range events {
		if evt == nil {
			continue
		}
		if invocationreadmodel.BeginsRunInvocation(evt) {
			if latestUsage && (!terminalAuthority || !latestCharge) {
				priorInvocationsComplete = false
			}
			terminalAuthority, latestUsage, latestCharge = false, false, false
		}
		if goal, ok := evt.Data.(*domain.GoalStatusChangedEventData); ok {
			status := runner.GoalStatus(goal.Status)
			if status.Valid() {
				state.GoalStatus, state.GoalObjective = status, goal.Objective
			}
		}
		if usage, ok := evt.Data.(*domain.UsageEventData); ok {
			if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.CacheReadTokens < 0 || usage.CacheCreationTokens < 0 || usage.Turns < 0 {
				return state, fmt.Errorf("negative provider usage for run %s", run.ID)
			}
			state.TokensKnown = true
			latestUsage = true
			terminalAuthority = terminalAuthority || usage.ReconciliationAuthority
			if !evt.Timestamp.IsZero() {
				observations = append(observations, evt.Timestamp)
			}
			if err := observeCharge(usage.Charge); err != nil {
				return state, err
			}
		}
		if charge, ok := evt.Data.(*domain.ChargeEventData); ok {
			if err := observeCharge(charge); err != nil {
				return state, err
			}
		}
	}
	state.MeterCadence = workflowruntime.MeterCadenceFromObservations(observations)
	state.MeterCadence.TerminalAuthority = terminalAuthority
	fact := invocationreadmodel.ProjectRun(run, events, now)
	if fact.TotalTokens < 0 || int64(int(fact.TotalTokens)) != fact.TotalTokens {
		return state, fmt.Errorf("provider token total overflow for run %s", run.ID)
	}
	state.Tokens, state.Turns = int(fact.TotalTokens), int(fact.Turns)
	state.ChargeMicroUSD = fact.MeteredChargeMicroUSD
	state.ChargeMeasured = chargeObserved && chargeComplete && latestUsage && latestCharge
	// StopRun can publish cancelled before the final accounting tail arrives.
	// A terminal status plus a prior live reading is not a terminal receipt.
	if state.Terminal && (!terminalAuthority || !priorInvocationsComplete) {
		state.TokensKnown = false
	}
	return state, nil
}
