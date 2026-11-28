// Package asyncpool provides the shared server-lifetime worker pool used by
// background domain executors.
package asyncpool

import (
	"context"
	"sync"
)

// JobFunc executes one queued job.
type JobFunc[Job any] func(context.Context, Job)

// Pool is a bounded worker pool. The pool owns a derived cancellation context
// so Shutdown does not depend on the caller cancelling the server context.
type Pool[Job any] struct {
	workers int
	queue   chan Job
	run     JobFunc[Job]
	baseCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New constructs a pool. workers and capacity are clamped to safe defaults.
func New[Job any](workers, capacity int) *Pool[Job] {
	if workers < 1 {
		workers = 1
	}
	if capacity < 1 {
		capacity = 1
	}
	return &Pool[Job]{workers: workers, queue: make(chan Job, capacity)}
}

// Bind wires the job function and starts the workers.
func (p *Pool[Job]) Bind(baseCtx context.Context, run JobFunc[Job]) {
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	p.baseCtx, p.cancel = context.WithCancel(baseCtx)
	p.run = run
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.loop()
	}
}

func (p *Pool[Job]) loop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.baseCtx.Done():
			return
		case job := <-p.queue:
			p.run(p.baseCtx, job)
		}
	}
}

// Submit queues a job unless the pool is shutting down.
func (p *Pool[Job]) Submit(job Job) {
	select {
	case p.queue <- job:
	case <-p.baseCtx.Done():
	}
}

// Shutdown cancels the workers and waits for them to finish or for ctx to
// expire.
func (p *Pool[Job]) Shutdown(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
