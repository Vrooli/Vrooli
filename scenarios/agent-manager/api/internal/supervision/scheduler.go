package supervision

import (
	"context"
	"sync"
	"time"
)

// Scheduler owns one timer for all durable watches. It queries only the due
// index and never creates a polling goroutine per child run.
type Scheduler struct {
	repo      *Repository
	processor *Processor
	kick      chan struct{}
	now       func() time.Time
	onError   func(error)
	once      sync.Once
	policies  *PolicyStore
	lastPrune time.Time
}

func NewScheduler(repo *Repository, processor *Processor, onError func(error)) *Scheduler {
	return &Scheduler{repo: repo, processor: processor, kick: make(chan struct{}, 1), now: time.Now, onError: onError}
}

func (s *Scheduler) SetPolicyStore(p *PolicyStore) { s.policies = p }

func (s *Scheduler) Kick() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.once.Do(func() { go s.loop(ctx) })
}

// RecoverOnce is the synchronous restart boundary used by startup tests and
// by the timer loop. The bounded due query prevents a restart storm.
func (s *Scheduler) RecoverOnce(ctx context.Context) (int, error) {
	if s.policies != nil && (s.lastPrune.IsZero() || s.now().Sub(s.lastPrune) >= time.Hour) {
		if _, err := s.policies.PruneExpired(ctx); err != nil {
			return 0, err
		}
		s.lastPrune = s.now()
	}
	due, err := s.repo.Due(ctx, s.now().UTC(), 100)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, watch := range due {
		if _, err := s.processor.Process(ctx, watch.GetWatchId()); err != nil {
			if s.onError != nil {
				s.onError(err)
			}
			continue
		}
		processed++
	}
	return processed, nil
}

func (s *Scheduler) loop(ctx context.Context) {
	for {
		_, err := s.RecoverOnce(ctx)
		if err != nil && s.onError != nil {
			s.onError(err)
		}
		next, err := s.repo.NextDue(ctx)
		if err != nil && s.onError != nil {
			s.onError(err)
		}
		var timer <-chan time.Time
		var owned *time.Timer
		if s.policies != nil {
			wake := s.lastPrune.Add(time.Hour)
			if next == nil || wake.Before(*next) {
				next = &wake
			}
		}
		if next != nil {
			delay := time.Until(*next)
			if delay < time.Second {
				delay = time.Second // failed due work cannot form a hot loop.
			}
			owned = time.NewTimer(delay)
			timer = owned.C
		}
		select {
		case <-ctx.Done():
			if owned != nil {
				owned.Stop()
			}
			return
		case <-s.kick:
			if owned != nil {
				owned.Stop()
			}
		case <-timer:
		}
	}
}
