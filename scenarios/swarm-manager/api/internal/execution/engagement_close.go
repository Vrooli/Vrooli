package execution

import (
	"context"
	"log/slog"
	"strings"
)

// Engagement close at review-decide (plan P-c).
//
// The atomic accept/reject of an item happens at review-decide, not at
// finalization, so the owner's whole engagement set is promoted (accept) or
// abandoned (reject) as a unit only when the user decides — never before. A
// followup leaves the set open so the next run continues under the same owner.
// The backlog→execution adapter (main.CloseEngagementsOnReviewDecide) maps the
// review decision onto an EngagementCloseDecision and calls CloseOwnerEngagements.

// CloseOwnerEngagements applies a terminal decision to the engagement set owned
// by a backlog item. It is best-effort and self-dispatching: promote/abandon
// shell out to git-control-tower (minutes), so the GCT work runs in a detached
// goroutine and the function returns immediately. A no-op when the owner has no
// open set or the engagement machinery is off.
func (s *Service) CloseOwnerEngagements(ctx context.Context, kind, name string, decision EngagementCloseDecision) {
	if s.engagementStore == nil || s.baselineEngagementRunner == nil {
		return
	}
	if decision == EngagementLeaveOpen {
		return // work continues under this owner; the set stays
	}

	owner := ownerKeyFor(kind, name)
	set, ok, err := s.engagementStore.Remove(owner)
	if err != nil {
		slog.Warn("baseline engagement: failed to load owner set for close",
			"owner", owner, "err", err)
		return
	}
	if !ok || len(set.Engagements) == 0 {
		return // nothing engaged under this owner
	}

	excludeRun := s.latestRunIDForItem(kind, name)
	go func() {
		bg := context.WithoutCancel(ctx)
		switch decision {
		case EngagementPromote:
			s.promoteOwnerSet(bg, set, excludeRun)
		case EngagementAbandon:
			s.abandonOwnerSet(bg, set)
		}
	}()
}

// latestRunIDForItem returns the RunID of the most recently created execution
// record for a backlog item, used as the promote drain self-guard's
// --exclude-run (memory part 12). Empty when no record carries a run id.
func (s *Service) latestRunIDForItem(kind, name string) string {
	records, err := s.store.Load()
	if err != nil {
		return ""
	}
	latestCreated := ""
	runID := ""
	for _, r := range records {
		if r.BacklogKind != kind || r.BacklogName != name {
			continue
		}
		if strings.TrimSpace(r.RunID) == "" {
			continue
		}
		if r.CreatedAt >= latestCreated {
			latestCreated = r.CreatedAt
			runID = r.RunID
		}
	}
	return runID
}
