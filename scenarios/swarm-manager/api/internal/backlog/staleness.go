package backlog

import (
	"strings"
	"time"

	"swarm-manager/internal/projectroot"
)

const defaultStaleAfter = 14 * 24 * time.Hour

// IsStale is deliberately a read-time calculation: stale state is an
// operator hint, not durable workflow state. A recent accepted review is a
// freshness signal independent of a content update. A malformed timestamp is
// stale because the item cannot establish its freshness deterministically.
func IsStale(item BacklogItem, repoRoot string, now time.Time) bool {
	freshness, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Updated))
	if item.LastReview != nil {
		if reviewedAt, reviewErr := time.Parse(time.RFC3339, strings.TrimSpace(item.LastReview.ReviewedAt)); reviewErr == nil && reviewedAt.After(freshness) {
			freshness, err = reviewedAt, nil
		}
	}
	if err != nil || now.Sub(freshness) >= defaultStaleAfter {
		return true
	}
	if item.PlanRef != nil && strings.TrimSpace(item.PlanRef.PlanID) == "" {
		return true
	}
	if len(item.AcceptanceAllow) == 0 || strings.TrimSpace(repoRoot) == "" {
		return false
	}
	_, err = projectroot.ValidateAcceptance(repoRoot, item.AcceptanceAllow, item.Creates)
	return err != nil
}
