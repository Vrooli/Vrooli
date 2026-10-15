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
// is best-effort — see the TODO below.
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
	if r.orch == nil || owner == "" {
		return authn.StandingGrantUsage{}, nil
	}
	concurrent := 0
	for i := range standingGrantConcurrencyStatuses {
		status := standingGrantConcurrencyStatuses[i]
		runs, err := r.orch.ListRuns(ctx, orchestration.RunListOptions{Status: &status})
		if err != nil {
			return authn.StandingGrantUsage{}, err
		}
		for _, run := range runs {
			if run != nil && run.OwnerSubject == owner {
				concurrent++
			}
		}
	}
	// TODO(auth-consolidation Phase 2b/6): committed per-owner spend has no cheap
	// existing read (BillingSnapshot carries plan/mode metadata, not accumulated
	// USD). Concurrency is the binding limit for now; budget is enforced only
	// against a request's own projected spend when a caller supplies it, which
	// create-run does not yet. MaxRunBudgetUSD therefore admits until a real
	// per-owner spend read exists. Flagged to the orchestrator; do not build an
	// estimator here.
	return authn.StandingGrantUsage{ConcurrentRuns: concurrent}, nil
}
