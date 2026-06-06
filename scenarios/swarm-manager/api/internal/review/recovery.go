package review

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

// envDuration reads a time.Duration env var (e.g. "30m", "5m") and falls back
// to the default on parse error. A bare integer is interpreted as seconds for
// ergonomics. Mirrors the feedback package helper of the same name.
func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return fallback
}

// HasLiveReviewRound reports whether the item's latest review round is still
// actively gathering — i.e. a real review is in flight. The recover-review
// endpoint uses this to refuse (409) manual recovery that would short-circuit a
// legitimate review.
//
// "Live" means: the latest round is gathering/pending AND its agent run is not
// terminal. If there is no inspector or the run is unreachable, the run is
// treated as NOT live so recovery isn't blocked by a dead run — recovery is a
// deliberate operator action and the sweeper would reclaim such an item anyway.
func (s *Service) HasLiveReviewRound(kind, name string) bool {
	rounds, err := LoadRounds(s.resolveItemDir(kind, name))
	if err != nil || len(rounds) == 0 {
		return false
	}
	latest := rounds[len(rounds)-1]
	if latest.Status != RoundStatusGathering && latest.Status != RoundStatusPending {
		return false
	}
	if s.inspector == nil || strings.TrimSpace(latest.RunID) == "" {
		return false
	}
	state, stateErr := s.inspector.GetRunState(context.Background(), latest.RunID)
	if stateErr != nil {
		return false // run unreachable: not a live review, don't block recovery
	}
	// Live only while the run has not reached a terminal state.
	return mapRunStatusToRoundStatus(state.Status) == ""
}

// ClassifyOrphan decides whether a backlog item currently in `in_review` is
// orphaned — stranded with no review round that will ever advance it — and
// returns a human-readable reason. The review sweeper calls this for every
// in_review item; itemUpdated is the item's spec `updated` timestamp, used to
// avoid racing a freshly-set in_review whose review agent is still spawning.
//
// An item is orphaned when:
//   - it has no review round at all and has been in_review past maxAge, or
//   - its latest round is already terminal (the in_review→review_pending flip
//     never happened — e.g. the handler was missed across a restart), or
//   - its latest round is gathering but the run is terminal/unreachable, or the
//     round has aged past maxAge (the review run died).
//
// A live gathering round younger than maxAge is healthy → not orphaned.
func (s *Service) ClassifyOrphan(kind, name, itemUpdated string, maxAge time.Duration) (bool, string) {
	rounds, err := LoadRounds(s.resolveItemDir(kind, name))
	if err != nil {
		return false, "" // never act on a read error
	}
	now := s.now()
	if len(rounds) == 0 {
		if olderThan(itemUpdated, now, maxAge) {
			return true, "in_review with no review round"
		}
		return false, ""
	}
	latest := rounds[len(rounds)-1]
	switch latest.Status {
	case RoundStatusComplete, RoundStatusFailed:
		return true, "review round is terminal but the item was never advanced out of in_review"
	case RoundStatusGathering, RoundStatusPending:
		if s.inspector != nil && strings.TrimSpace(latest.RunID) != "" {
			if state, stateErr := s.inspector.GetRunState(context.Background(), latest.RunID); stateErr == nil {
				if mapRunStatusToRoundStatus(state.Status) != "" {
					return true, "review run finished but the round was never finalized"
				}
				// Run is live; orphan only if it has aged out.
				if olderThan(latest.GeneratedAt, now, maxAge) {
					return true, "review run exceeded max age"
				}
				return false, ""
			}
			// Inspector error (run unreachable): orphan only once aged out.
			if olderThan(latest.GeneratedAt, now, maxAge) {
				return true, "review run is unreachable and exceeded max age"
			}
			return false, ""
		}
		// No inspector / no run id: fall back to age of the round.
		if olderThan(latest.GeneratedAt, now, maxAge) {
			return true, "gathering review round exceeded max age with no tracked run"
		}
		return false, ""
	}
	return false, ""
}

// olderThan reports whether an RFC3339 timestamp is more than maxAge before
// now. An unparseable timestamp is treated as not-older so malformed data is
// never force-recovered on a clock guess.
func olderThan(ts string, now time.Time, maxAge time.Duration) bool {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(ts))
	if err != nil {
		return false
	}
	return now.Sub(parsed) > maxAge
}
