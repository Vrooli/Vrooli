package orchestration

import (
	"context"
	"errors"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// Standalone runs (for example Swarm goal runs) are not workflow children, so
// workflow recovery never settles their terminal usage. This sweep does, for
// recently ended runs whose live stream lost the harness's final receipt.
const (
	terminalAccountingLookback     = 7 * 24 * time.Hour
	terminalAccountingInspectBatch = 50
	terminalAccountingListLimit    = 500
	terminalAccountingRetryAfter   = 15 * time.Minute
)

// terminalAccountingSweep remembers settled runs and failed attempts so each
// reconcile cycle inspects only runs that still owe accounting.
type terminalAccountingSweep struct {
	mu      sync.Mutex
	settled map[uuid.UUID]struct{}
	retryAt map[uuid.UUID]time.Time
	// exclude proves the run's executor is gone before its transcript is read.
	// Nil uses the control plane's executor-scope exclusion.
	exclude func(context.Context, *domain.Run) error
}

func (s *terminalAccountingSweep) exclusion() func(context.Context, *domain.Run) error {
	if s.exclude != nil {
		return s.exclude
	}
	return maintenance.ExcludeExecutor
}

func (s *terminalAccountingSweep) due(id uuid.UUID, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.settled[id]; ok {
		return false
	}
	at, ok := s.retryAt[id]
	return !ok || !now.Before(at)
}

func (s *terminalAccountingSweep) settle(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled == nil {
		s.settled = map[uuid.UUID]struct{}{}
	}
	s.settled[id] = struct{}{}
	delete(s.retryAt, id)
}

func (s *terminalAccountingSweep) retry(id uuid.UUID, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retryAt == nil {
		s.retryAt = map[uuid.UUID]time.Time{}
	}
	s.retryAt[id] = at
}

// RecoverStandaloneTerminalAccounting settles terminal usage for recently ended
// managed runs from the harness's own retained transcript. It is bounded per
// call; a run whose usage cannot be proven is retried later and stays unknown.
func (o *Orchestrator) RecoverStandaloneTerminalAccounting(ctx context.Context) error {
	if o.runs == nil || o.events == nil || o.runners == nil {
		return nil
	}
	now := o.now()
	from := now.Add(-terminalAccountingLookback)
	sweep := &o.terminalAccounting
	inspected := 0
	var errs []error
	for _, status := range []domain.RunStatus{domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusNeedsReview} {
		status := status
		runs, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: terminalAccountingListLimit}, Status: &status, EndedFrom: &from})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, run := range runs {
			if inspected >= terminalAccountingInspectBatch {
				return errors.Join(errs...)
			}
			if run == nil || !sweep.due(run.ID, now) {
				continue
			}
			mode := run.ExecutionMode.Normalized()
			if run.SessionID == "" || (mode != domain.ExecutionModeCodecPipe && mode != domain.ExecutionModeInteractive) {
				// No retained harness transcript this sweep could read.
				sweep.settle(run.ID)
				continue
			}
			inspected++
			if _, state, err := o.meteredRun(ctx, run.ID); err == nil && state.TokensKnown && state.ChargeMeasured {
				sweep.settle(run.ID)
				continue
			}
			if err := o.recoverTerminalAccounting(ctx, run.ID, sweep.exclusion()); err != nil {
				// Unknown usage is an expected outcome, not a cycle failure.
				sweep.retry(run.ID, now.Add(terminalAccountingRetryAfter))
				continue
			}
			sweep.settle(run.ID)
		}
	}
	return errors.Join(errs...)
}
