package workflowruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

const meteredStopPrefix = "metered_budget_stop:"

type meteredSettlement struct {
	RunID              string       `json:"runId"`
	Tokens             int          `json:"tokens"`
	OvershootTokens    int          `json:"overshootTokens"`
	TerminalAuthority  bool         `json:"terminalAuthority"`
	MeterCadence       MeterCadence `json:"meterCadence"`
	StopLatencySeconds *float64     `json:"stopLatencySeconds,omitempty"`
}

// Cancellation uses the same accounting barrier as budget stops. A stop RPC
// acknowledgement is insufficient, and an unbound dispatch cannot be presumed
// absent: it may be a lost response from an owner that already started work.
func (e *Engine) reconcileMeteredCleanup(ctx context.Context, x *domain.WorkflowExecution, journal []*domain.WorkflowJournalEntry) ([]*domain.WorkflowNodeAttempt, []meteredSettlement, error) {
	if e.Catalog == nil {
		return nil, nil, fmt.Errorf("pinned workflow catalog unavailable during cleanup")
	}
	r, err := e.Catalog.GetByDigest(ctx, x.DefinitionDigest)
	if err != nil {
		return nil, nil, err
	}
	if r == nil {
		return nil, nil, fmt.Errorf("pinned workflow revision unavailable during cleanup")
	}
	if r.Definition.Budgets.Enforcement != domain.WorkflowBudgetMeteredCancellation {
		return e.reconcileOrdinaryCleanup(ctx, x, journal)
	}
	meter, ok := e.Children.(MeteredChildLauncher)
	if !ok {
		return nil, nil, fmt.Errorf("live usage inspection unavailable during cleanup")
	}
	attempts, err := e.Store.ListAttempts(ctx, x.ID)
	if err != nil {
		return nil, nil, err
	}
	var settled []*domain.WorkflowNodeAttempt
	var settlements []meteredSettlement
	usage := x.BudgetUsage
	for _, a := range attempts {
		if a.Status == domain.WorkflowAttemptCompleted || a.Status == domain.WorkflowAttemptFailed {
			continue
		}
		if a.RunID == nil {
			return nil, nil, fmt.Errorf("unbound attempt %s requires dispatch reconciliation before cleanup", a.ID)
		}
		state, err := meter.InspectMetered(ctx, *a.RunID)
		if err != nil {
			return nil, nil, err
		}
		if !state.Terminal || !state.TokensKnown {
			return nil, nil, fmt.Errorf("child %s requires terminal usage before cleanup", a.RunID)
		}
		remaining := r.Definition.Budgets.MaxTokens - usage.Tokens
		overshoot := state.Tokens - remaining
		if overshoot < 0 {
			overshoot = 0
		}
		var stopLatency *float64
		if intentAt := stopIntentAt(journal, a.ID); !intentAt.IsZero() && !state.TerminalObservedAt.IsZero() {
			seconds := state.TerminalObservedAt.Sub(intentAt).Seconds()
			if seconds < 0 {
				seconds = 0
			}
			stopLatency = &seconds
		}
		settlements = append(settlements, meteredSettlement{RunID: a.RunID.String(), Tokens: state.Tokens, OvershootTokens: overshoot, TerminalAuthority: state.MeterCadence.TerminalAuthority, MeterCadence: state.MeterCadence, StopLatencySeconds: stopLatency})
		now := e.now()
		a.Status, a.ErrorCode = domain.WorkflowAttemptFailed, "cancelled"
		a.UpdatedAt, a.CompletedAt = now, &now
		a.Version++
		usage.Tokens += state.Tokens
		usage.Turns += state.Turns
		usage.ChargeMicroUSD += state.ChargeMicroUSD
		usage.ChargeMeasured = usage.ChargeMeasured || state.ChargeMeasured
		usage.AccountingComplete = usage.AccountingComplete && state.TokensKnown && state.ChargeMeasured
		settled = append(settled, a)
	}
	x.BudgetUsage = usage
	return settled, settlements, nil
}

func stopIntentAt(journal []*domain.WorkflowJournalEntry, attemptID uuid.UUID) time.Time {
	for _, entry := range journal {
		if entry == nil || entry.Kind != domain.WorkflowJournalDiagnostic || entry.AttemptID == nil || *entry.AttemptID != attemptID {
			continue
		}
		var payload struct {
			Code            string `json:"code"`
			StopRequestedAt string `json:"stopRequestedAt"`
		}
		if json.Unmarshal(entry.Payload, &payload) != nil || payload.Code != "metered_cancellation_requested" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, payload.StopRequestedAt)
		if err == nil {
			return at
		}
	}
	return time.Time{}
}

// Persist intent before requesting a stop. Neither a successful stop request nor
// an exhausted meter proves the process stopped or its final usage is known.
// Recovery re-enters this path and retries; terminal settlement happens once in
// advanceAgent's versioned commit, including any in-flight overshoot.
func (e *Engine) meterActiveChild(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, a *domain.WorkflowNodeAttempt, state ChildState, journal []*domain.WorkflowJournalEntry) (*domain.WorkflowExecution, error) {
	budget := ""
	limits := r.Definition.Budgets
	now := e.now()
	switch {
	case strings.HasPrefix(a.ErrorCode, meteredStopPrefix):
		budget = strings.TrimPrefix(a.ErrorCode, meteredStopPrefix)
	case state.TokensKnown && state.Tokens >= limits.MaxTokens-x.BudgetUsage.Tokens:
		budget = "tokens"
	case state.Turns >= limits.MaxTurns-x.BudgetUsage.Turns:
		budget = "turns"
	case state.ChargeMeasured && state.ChargeMicroUSD >= limits.MaxChargeMicroUSD-x.BudgetUsage.ChargeMicroUSD:
		budget = "charge_micro_usd"
	case now.Sub(x.CreatedAt)-waitedDuration(journal, now) >= time.Duration(limits.WallTimeSeconds)*time.Second:
		budget = "wall_time"
	}
	if budget == "" {
		return x, nil
	}
	if !strings.HasPrefix(a.ErrorCode, meteredStopPrefix) {
		a.ErrorCode, a.UpdatedAt = meteredStopPrefix+budget, now
		a.Version++
		x.Version++
		x.UpdatedAt = now
		payload, _ := json.Marshal(map[string]any{
			"code": "metered_cancellation_requested", "budget": budget,
			"observedChildTokens": state.Tokens, "overshootPossible": true,
			"meterSamples":             state.MeterCadence.Samples,
			"meterMeanIntervalSeconds": state.MeterCadence.MeanIntervalSeconds,
			"meterMaxIntervalSeconds":  state.MeterCadence.MaxIntervalSeconds,
			"stopRequestedAt":          now.UTC().Format(time.RFC3339Nano),
		})
		entry := nextJournal(x.ID, journal, domain.WorkflowJournalDiagnostic, a.NodeID, &a.ID, payload, now)
		ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: a, Journal: []*domain.WorkflowJournalEntry{entry}})
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrConcurrentAdvance
		}
	}
	return x, e.Children.Stop(ctx, *a.RunID)
}
