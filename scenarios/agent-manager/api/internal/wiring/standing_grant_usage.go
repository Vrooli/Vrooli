package wiring

import (
	"context"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/vrooli/api-core/authn"
)

// standingGrantUsageReader measures a standing grant against existing run reads
// rather than new accounting. Concurrency is the firm, binding limit: it counts
// the owner's non-terminal runs via the orchestrator's existing ListRuns. Budget
// is enforced as the committed-spend floor already attributed to those runs —
// see StandingGrantUsage below for the exact approximation.
type standingGrantUsageReader struct {
	orch *orchestration.Orchestrator
}

// standingGrantConcurrencyStatuses are the non-terminal run states that count
// toward an owner's concurrency. These are every state that still holds, or can
// resume to hold, a live agent for the owner — i.e. not IsTerminal(). Listing by
// explicit status reuses the indexed status filter instead of scanning all runs.
var standingGrantConcurrencyStatuses = []domain.RunStatus{
	domain.RunStatusPending,
	domain.RunStatusStarting,
	domain.RunStatusRunning,
	domain.RunStatusNeedsReview,
	domain.RunStatusParked,
}

func (r standingGrantUsageReader) StandingGrantUsage(ctx context.Context, owner string) (authn.StandingGrantUsage, error) {
	usage, _, err := r.read(ctx, owner, 0)
	return usage, err
}

// StandingGrantUsageWithRuns returns the same aggregate usage as StandingGrantUsage
// plus a bounded list of the exact rows counted. The diagnostic uses this method
// only after its existing investigator authorization succeeds.
func (r standingGrantUsageReader) StandingGrantUsageWithRuns(ctx context.Context, owner string, limit int) (authn.StandingGrantUsage, []domain.Run, error) {
	return r.read(ctx, owner, limit)
}

func (r standingGrantUsageReader) read(ctx context.Context, owner string, runLimit int) (authn.StandingGrantUsage, []domain.Run, error) {
	if r.orch == nil || owner == "" {
		return authn.StandingGrantUsage{}, nil, nil
	}
	if runLimit < 0 {
		runLimit = 0
	}
	concurrent := 0
	committedUSD := 0.0
	var countedRuns []domain.Run
	if runLimit > 0 {
		countedRuns = make([]domain.Run, 0, runLimit)
	}
	for i := range standingGrantConcurrencyStatuses {
		status := standingGrantConcurrencyStatuses[i]
		runs, err := r.orch.ListRuns(ctx, orchestration.RunListOptions{Status: &status})
		if err != nil {
			return authn.StandingGrantUsage{}, nil, err
		}
		for _, run := range runs {
			if run == nil || run.OwnerSubject != owner {
				continue
			}
			concurrent++
			if len(countedRuns) < runLimit {
				countedRuns = append(countedRuns, domain.Run{ID: run.ID, Status: run.Status})
			}
			// Committed spend uses the ONLY per-run USD figure the domain already
			// records: the run's accumulated cost estimate (RunSummary.CostEstimate,
			// derived from metered usage). There is no admission-time per-run USD
			// budget field to sum (RunConfig carries only token/turn ceilings), so we
			// do NOT invent an estimator — we sum the spend already attributed to the
			// owner's live runs. The pruned List() column set omits `summary`, so each
			// owner run is loaded by id for its cost; the set is bounded by the owner's
			// concurrency, so this is a small, bounded read, not a scan.
			full, err := r.orch.GetRun(ctx, run.ID)
			if err != nil {
				return authn.StandingGrantUsage{}, nil, err
			}
			if full != nil && full.Summary != nil {
				committedUSD += full.Summary.CostEstimate
			}
		}
	}

	// CommittedBudgetUSD is the committed-spend FLOOR already attributed to the
	// owner's non-terminal runs. It is honest (never an estimate) but an
	// APPROXIMATION of true committed spend in two bounded ways:
	//   1. A run that has not yet projected a cost (just-started, summary not yet
	//      derived) contributes 0, so the floor can lag live in-flight spend.
	//   2. create-run supplies no per-request projected spend, so RequestBudgetUSD
	//      stays 0; the budget is enforced on already-committed owner spend only.
	// It errs toward admitting (a floor, not a ceiling), so it never refuses a
	// caller that is actually under budget. The concurrency bound remains the firm
	// limit; the budget bound now additionally refuses an owner whose live runs
	// have ALREADY committed at or beyond MaxRunBudgetUSD.
	return authn.StandingGrantUsage{ConcurrentRuns: concurrent, CommittedBudgetUSD: committedUSD}, countedRuns, nil
}
