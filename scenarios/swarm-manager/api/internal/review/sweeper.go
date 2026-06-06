package review

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"swarm-manager/internal/backlog"
)

// DefaultSweepInterval is how often the review sweeper ticker runs. Mirrors
// feedback.DefaultSweepInterval.
const DefaultSweepInterval = 5 * time.Minute

// BacklogReviewLister is the minimal surface the sweeper needs to enumerate
// backlog items currently in a review-gated status. Implemented by
// backlog.Store (LoadAll).
type BacklogReviewLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// OrphanRecoverFn flips an orphaned `in_review` item to `review_pending` with an
// audit record. Implemented by the backlog handler (RecoverOrphanedReview) so
// the sweeper reuses the same recovery + audit path as the manual endpoint.
type OrphanRecoverFn func(ctx context.Context, kind, name, reason string) error

// Sweeper periodically scans every `in_review` backlog item and recovers those
// that are orphaned — stranded with no review round that will ever advance
// them (work done out-of-band, a review run that died, or a premature mark).
// It is the engine-agnostic safety net that guarantees no item can sit in
// `in_review` forever, complementing the finalization source fix (which
// prevents the common case) and the poller's max-age backstop (which handles
// tracked rounds whose runs die).
type Sweeper struct {
	Service  *Service
	Backlog  BacklogReviewLister
	Recover  OrphanRecoverFn
	MaxAge   time.Duration
	Interval time.Duration
}

// NewSweeper constructs a review Sweeper with sane defaults. The max age
// matches the service's abandoned-round threshold so the sweeper and poller
// agree on what "too old" means.
func NewSweeper(svc *Service, store BacklogReviewLister, recover OrphanRecoverFn) *Sweeper {
	maxAge := DefaultRoundMaxAge
	if svc != nil {
		maxAge = svc.maxRoundAge()
	}
	return &Sweeper{
		Service:  svc,
		Backlog:  store,
		Recover:  recover,
		MaxAge:   maxAge,
		Interval: envDuration("SWARM_MANAGER_REVIEW_SWEEP_INTERVAL", DefaultSweepInterval),
	}
}

// RunOnce performs a single sweep pass and returns the number of items
// recovered. Safe to call at boot (synchronous) and from the ticker. Logs and
// continues on per-item errors so one bad item can't block the rest.
func (s *Sweeper) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.Service == nil || s.Backlog == nil || s.Recover == nil {
		return 0, nil
	}
	items, err := s.Backlog.LoadAll(nil)
	if err != nil {
		return 0, fmt.Errorf("list backlog items: %w", err)
	}
	maxAge := s.maxAge()
	recovered := 0
	for _, item := range items {
		if item.Status != backlog.StatusInReview || backlog.IsArchived(item) {
			continue
		}
		orphaned, reason := s.Service.ClassifyOrphan(string(item.Kind), item.Name, item.Updated, maxAge)
		if !orphaned {
			continue
		}
		if rErr := s.Recover(ctx, string(item.Kind), item.Name, reason); rErr != nil {
			slog.Warn("review: sweeper: recover failed",
				"err", rErr, "kind", item.Kind, "name", item.Name)
			continue
		}
		slog.Info("review: sweeper: recovered orphaned in_review item",
			"kind", item.Kind, "name", item.Name, "reason", reason)
		recovered++
	}
	if recovered > 0 {
		slog.Info("review: sweeper: pass complete", "recovered", recovered)
	}
	return recovered, nil
}

// Start runs RunOnce on a ticker until ctx is cancelled. Run in its own
// goroutine; recovers from panics so a transient disk error doesn't kill the
// safety net.
func (s *Sweeper) Start(ctx context.Context) {
	if s == nil || s.Interval <= 0 {
		return
	}
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.runWithRecover(ctx)
		}
	}
}

func (s *Sweeper) runWithRecover(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("review: sweeper: panic recovered", "panic", rec)
		}
	}()
	if _, err := s.RunOnce(ctx); err != nil {
		slog.Warn("review: sweeper: run-once failed", "err", err)
	}
}

func (s *Sweeper) maxAge() time.Duration {
	if s.MaxAge > 0 {
		return s.MaxAge
	}
	return DefaultRoundMaxAge
}
