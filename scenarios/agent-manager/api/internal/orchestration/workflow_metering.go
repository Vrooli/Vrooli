package orchestration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
	run, state, err := l.o.meteredRun(ctx, id)
	if err == nil && run != nil && state.Terminal && (!state.TokensKnown || !state.ChargeMeasured) {
		state.AccountingRecoveryUnavailable = !l.terminalAccountingRecoveryAvailable(ctx, run)
	}
	return state, err
}

// terminalAccountingRecoveryAvailable is intentionally conservative. A
// terminal child may remain unresolved while a durable transcript or native
// interactive transcript can still produce an authoritative receipt. Once
// those bounded sources are absent, the workflow owner must terminalize with
// accounting_unknown rather than retrying forever or inventing zero usage.
func (l workflowChildLauncher) terminalAccountingRecoveryAvailable(ctx context.Context, run *domain.Run) bool {
	if run == nil || !run.Status.IsTerminal() || run.SessionID == "" || run.ResolvedConfig == nil {
		return false
	}
	mode := run.ExecutionMode.Normalized()
	if mode != domain.ExecutionModeCodecPipe && mode != domain.ExecutionModeInteractive {
		return false
	}
	transcript := strings.TrimSpace(run.TranscriptPath)
	if transcript == "" && mode == domain.ExecutionModeInteractive && run.ResolvedConfig.RunnerType == domain.RunnerTypeClaudeCode {
		transcript, _ = findClaudeNativeTranscript(run.SessionID)
	}
	if transcript == "" || l.o == nil || l.o.runners == nil {
		return false
	}
	info, err := os.Stat(transcript)
	if err != nil || info.IsDir() {
		return false
	}
	owned, err := l.o.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil {
		return false
	}
	factory, ok := owned.(runner.TranscriptParserFactory)
	if !ok {
		return false
	}
	parser := factory.NewTranscriptParser()
	if setter, ok := parser.(runner.TranscriptModelSetter); ok {
		setter.SetTranscriptModel(runTranscriptModel(run))
	}
	if setter, ok := parser.(runner.TranscriptBillingSetter); ok {
		setter.SetTranscriptBilling(run.Billing)
	}
	_, _, err = readTranscriptTerminalReceipt(ctx, run, transcript, parser, mode == domain.ExecutionModeInteractive)
	return err == nil
}

func meteredWorkflowChildState(run *domain.Run, events []*domain.RunEvent, now time.Time) (workflowruntime.ChildState, error) {
	state := childStateFromRun(run)
	events = terminalWorkflowReceiptProjection(events)
	if run != nil && run.EndedAt != nil {
		state.TerminalObservedAt = *run.EndedAt
	}
	state.Tokens, state.Turns, state.ChargeMicroUSD = 0, 0, 0
	state.ChargeMeasured = false
	terminalAuthority := false
	latestUsage, latestCharge := false, false
	priorInvocationsComplete := true
	chargeObserved, chargeComplete := false, true
	preEffectProven := false
	runnerAcquired := false
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
		if lifecycle, ok := evt.Data.(*domain.LifecycleEventData); ok {
			if lifecycle.Phase == domain.LifecyclePhaseRunnerAcquired {
				runnerAcquired = true
			}
		}
		if failure, ok := evt.Data.(*domain.ErrorEventData); ok && failure.Details != nil {
			known, started := executionEffectEvidence(failure.Details)
			if known && !started {
				preEffectProven = true
			}
		}
		if usage, ok := evt.Data.(*domain.UsageEventData); ok {
			if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.CacheReadTokens < 0 || usage.CacheCreationTokens < 0 || usage.Turns < 0 {
				return state, fmt.Errorf("negative provider usage for run %s", run.ID)
			}
			state.TokensKnown = true
			latestUsage = true
			// An earlier receipt cannot close usage observed after its cut.
			// A later authoritative receipt may establish completion again.
			terminalAuthority = usage.ReconciliationAuthority
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
	// A runner adapter can explicitly prove that launch failed before any
	// provider process existed. In that case zero usage and zero marginal
	// charge are authoritative, even though no provider receipt was emitted.
	// The legacy start-process shape is retained as a read-only compatibility
	// bridge for terminal runs written before execution-effect evidence was
	// added; new failures always use the typed detail above.
	if state.Terminal && !latestUsage && !runnerAcquired && !preEffectProven && legacyPreEffectFailure(run) {
		preEffectProven = true
	}
	if state.Terminal && preEffectProven && !latestUsage {
		state.TokensKnown = true
		state.ChargeMeasured = chargeBasisCanMeasureZero(run.Billing.EffectiveBasis())
	}
	// StopRun can publish cancelled before the final accounting tail arrives.
	// A terminal status plus a prior live reading is not a terminal receipt.
	if state.Terminal && !preEffectProven && (!terminalAuthority || !priorInvocationsComplete) {
		state.TokensKnown = false
	}
	return state, nil
}

func executionEffectEvidence(details map[string]interface{}) (known, started bool) {
	knownValue, ok := details["execution_started_known"].(bool)
	if !ok || !knownValue {
		return false, false
	}
	startedValue, ok := details["execution_started"].(bool)
	if !ok {
		return false, false
	}
	return true, startedValue
}

func chargeBasisCanMeasureZero(basis domain.ChargeBasis) bool {
	switch basis {
	case domain.ChargeBasisMetered, domain.ChargeBasisSubscription, domain.ChargeBasisLocal:
		return true
	default:
		return false
	}
}

func legacyPreEffectFailure(run *domain.Run) bool {
	if run == nil || run.Status != domain.RunStatusFailed || strings.TrimSpace(run.ErrorMsg) == "" {
		return false
	}
	// All supported launchers wrap a failed process creation at this stable
	// boundary. Keep this narrow so an old post-launch/provider failure cannot
	// be mistaken for zero usage merely because its receipt is missing.
	return strings.Contains(strings.ToLower(run.ErrorMsg), "start process:")
}

// An embedded terminal charge covers the same complete invocation as its
// authoritative usage. Earlier unpriced/interim samples remain durable but
// must not permanently poison that original receipt or be charged again.
// No receipt means no substitution; each continuation keeps its own boundary.
// Later observations are outside the receipt's coverage and must survive.
func terminalWorkflowReceiptProjection(events []*domain.RunEvent) []*domain.RunEvent {
	invocation := 0
	selected := map[int]int{}
	for i, item := range events {
		if invocationreadmodel.BeginsRunInvocation(item) {
			invocation++
		}
		if item == nil {
			continue
		}
		if usage, ok := item.Data.(*domain.UsageEventData); ok && usage.ReconciliationAuthority && usage.Charge != nil && usage.Charge.AmountMicroUSD != nil {
			selected[invocation] = i
		}
	}
	if len(selected) == 0 {
		return events
	}
	var out []*domain.RunEvent
	invocation = 0
	for i, item := range events {
		if invocationreadmodel.BeginsRunInvocation(item) {
			invocation++
		}
		if selectedIndex, ok := selected[invocation]; ok && i < selectedIndex && item != nil {
			switch data := item.Data.(type) {
			case *domain.UsageEventData:
				// Invalid observations still fail validation; reconciliation
				// cannot hide malformed retained accounting.
				if data.InputTokens >= 0 && data.OutputTokens >= 0 && data.CacheReadTokens >= 0 && data.CacheCreationTokens >= 0 && data.Turns >= 0 {
					continue
				}
			case *domain.ChargeEventData:
				if data.AmountMicroUSD == nil || *data.AmountMicroUSD >= 0 {
					continue
				}
			}
		}
		out = append(out, item)
	}
	return out
}
