package orchestration

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/findings"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// FrictionPublishScheduler periodically publishes qualifying recurring
// friction. The operation is idempotent in the finding store.
type FrictionPublishScheduler struct {
	orchestrator *Orchestrator
	interval     time.Duration
	cancel       context.CancelFunc
	done         chan struct{}
	mu           sync.Mutex
	sweeps       atomic.Int64
	lastPublish  atomic.Int64
}

func NewFrictionPublishScheduler(orchestrator *Orchestrator, interval time.Duration) *FrictionPublishScheduler {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &FrictionPublishScheduler{orchestrator: orchestrator, interval: interval}
}

func (s *FrictionPublishScheduler) Sweeps() int64 {
	if s == nil {
		return 0
	}
	return s.sweeps.Load()
}
func (s *FrictionPublishScheduler) LastPublishAt() time.Time {
	if s == nil || s.lastPublish.Load() == 0 {
		return time.Time{}
	}
	return time.Unix(0, s.lastPublish.Load()).UTC()
}

func (s *FrictionPublishScheduler) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil || s.orchestrator == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel, s.done = cancel, done
	go func() {
		defer close(done)
		s.sweep(workerCtx)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.sweep(workerCtx)
			case <-workerCtx.Done():
				return
			}
		}
	}()
}

func (s *FrictionPublishScheduler) sweep(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	if summary, err := s.RunOnce(ctx); err != nil {
		obs.Component("friction-publisher").Warn("scheduled friction publication failed", obs.KeyError, err.Error())
	} else if summary.Filed > 0 {
		s.lastPublish.Store(time.Now().UTC().UnixNano())
	}
	s.sweeps.Add(1)
}

func (s *FrictionPublishScheduler) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (s *FrictionPublishScheduler) RunOnce(ctx context.Context) (RecurringFrictionSummary, error) {
	if s == nil || s.orchestrator == nil {
		return RecurringFrictionSummary{}, nil
	}
	summary, err := s.orchestrator.PublishRecurringFriction(ctx, invocationreadmodel.Filter{}, 25)
	if err != nil {
		return summary, err
	}
	if err := s.measurePending(ctx); err != nil {
		return summary, err
	}
	return summary, nil
}

// measurePending records the current recurrence count after the settle window.
// It preserves the original before value so repeated sweeps cannot erase it.
func (s *FrictionPublishScheduler) measurePending(ctx context.Context) error {
	o := s.orchestrator
	projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore)
	if !ok || o.findings == nil {
		return nil
	}
	items, err := o.findings.List(ctx, findings.Filter{Severity: "recurring", Limit: 100})
	if err != nil {
		return err
	}
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	for _, item := range items {
		if item.BeforeValue == nil || item.AfterValue != nil || item.FrictionTopic == "" || item.CreatedAt.After(cutoff) {
			continue
		}
		episodes, err := projection.Episodes(ctx, invocationreadmodel.Filter{Fingerprint: item.Fingerprint}, invocationEvidenceLimit)
		if err != nil {
			return err
		}
		seen := map[string]struct{}{}
		for _, episode := range episodes {
			if _, err := uuid.Parse(episode.RunID); err == nil {
				seen[episode.RunID] = struct{}{}
			}
		}
		after := float64(len(seen))
		if err := o.findings.SetEffectiveness(ctx, item.ID, item.BeforeValue, &after, findings.Effectiveness(item.BeforeValue, &after), item.FrictionTopic); err != nil {
			return err
		}
	}
	return nil
}
