package hub

import (
	"context"
	"time"
)

// closeWait bounds how long Close waits for the worker to finish what it is
// doing before it gives up and reports it.
const closeWait = 15 * time.Second

// serviceLifecycle owns the Service's background worker. The worker starts in
// New and runs until Close; nothing else starts or stops it.
type serviceLifecycle struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func newServiceLifecycle() serviceLifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return serviceLifecycle{ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

// Close stops the background worker and waits for the unit of work it is in the
// middle of (a delivery, a sweep) to finish. It is idempotent. Work already
// begun is never cancelled mid-flight, so a delivery is neither lost nor
// recorded as failed because the process is shutting down; a notification left
// pending is picked up by the next boot's recovery pass. Close the Service
// before closing the database it uses.
func (s *Service) Close() {
	s.life.cancel()
	select {
	case <-s.life.done:
	case <-time.After(closeWait):
		s.log.Printf("notification hub: background worker still busy after %s; closing anyway", closeWait)
	}
}

// closing reports whether Close was called.
func (s *Service) closing() bool { return s.life.ctx.Err() != nil }

// runWorker is the Service's only background loop: it routes dispatched
// notifications as they arrive and, once a minute, runs the maintenance pass.
func (s *Service) runWorker() {
	defer close(s.life.done)
	// Work runs on a context that Close does not cancel; Close is observed
	// between units of work.
	work := context.WithoutCancel(s.life.ctx)
	s.safely("recover pending notifications", func() { s.recoverPending(work) })
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.life.ctx.Done():
			return
		case item := <-s.worker:
			ctx := work
			if item.ctx != nil {
				ctx = item.ctx
			}
			s.safely("process notification "+item.id, func() {
				if err := s.Process(ctx, item.id); err != nil {
					s.log.Printf("notification %s: %v", item.id, err)
				}
			})
		case <-ticker.C:
			s.maintain(work)
		}
	}
}

// maintain is the once-a-minute pass: recover stranded notifications, advance
// asks, publish ask events, apply retention (once a day), and sweep unreferenced
// attachment blobs. Each step is independent, so one failing never skips the
// rest. Retention comes before the sweep so the blobs it frees go in the same pass.
func (s *Service) maintain(ctx context.Context) {
	s.safely("recover pending notifications", func() { s.recoverPending(ctx) })
	s.safely("process ask escalations", func() {
		if err := s.ProcessEscalations(ctx); err != nil {
			s.log.Printf("process ask escalations: %v", err)
		}
	})
	s.safely("publish ask events", func() {
		if err := s.publishAskEvents(ctx); err != nil {
			s.log.Printf("publish ask events: %v", err)
		}
	})
	s.safely("apply retention", func() { s.runRetention(ctx) })
	s.safely("sweep attachment blobs", func() {
		if _, err := s.SweepAttachmentBlobs(ctx); err != nil {
			s.log.Printf("sweep attachment blobs: %v", err)
		}
	})
}

// safely runs one background step. A panic (for example from a database that
// was closed under it) is logged and ends the step, never the process: the
// worker has no caller to report to and the next tick tries again. A step is
// skipped once the Service is closing.
func (s *Service) safely(what string, step func()) {
	if s.closing() {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			s.log.Printf("notification hub: %s: recovered from %v", what, r)
		}
	}()
	step()
}
