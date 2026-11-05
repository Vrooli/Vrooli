package export

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrExportCapacity = errors.New("export capacity unavailable or runner is draining")

// JobRunner owns bounded work for one API process. Durable rows are reconciled
// on startup; interruption is explicit failure, never automatic render replay.
type JobRunner struct {
	mu       sync.Mutex
	jobs     map[uuid.UUID]context.CancelFunc
	limit    int
	timeout  time.Duration
	draining bool
	done     chan struct{}
}

func NewJobRunner(limit int, timeout time.Duration) *JobRunner {
	if limit < 1 || timeout <= 0 {
		panic("export runner requires positive capacity and timeout")
	}
	return &JobRunner{jobs: make(map[uuid.UUID]context.CancelFunc), limit: limit, timeout: timeout, done: make(chan struct{})}
}

// Start reserves capacity before persisting admission. Context values (including
// test routing and entitlement) survive, but the caller does not own job lifetime.
func (r *JobRunner) Start(inherited context.Context, id uuid.UUID, admit func(context.Context) error, work func(context.Context)) error {
	r.mu.Lock()
	if r.draining || len(r.jobs) >= r.limit {
		r.mu.Unlock()
		return ErrExportCapacity
	}
	if _, exists := r.jobs[id]; exists {
		r.mu.Unlock()
		return fmt.Errorf("export already active")
	}
	if err := inherited.Err(); err != nil {
		r.mu.Unlock()
		return err
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(inherited), r.timeout)
	r.jobs[id] = cancel
	r.mu.Unlock()
	finish := func() {
		cancel()
		r.mu.Lock()
		delete(r.jobs, id)
		if r.draining && len(r.jobs) == 0 {
			close(r.done)
		}
		r.mu.Unlock()
	}
	admission, admissionCancel := context.WithTimeout(inherited, 5*time.Second)
	stop := context.AfterFunc(ctx, admissionCancel)
	err := admit(admission)
	stop()
	admissionCancel()
	if err != nil {
		finish()
		return err
	}
	go func() { defer finish(); work(ctx) }()
	return nil
}

// Cancel is idempotent while an export remains active. Its worker persists the
// existing failed state using its independent bounded finalization context.
func (r *JobRunner) Cancel(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cancel, ok := r.jobs[id]
	if ok {
		cancel()
	}
	return ok
}
func (r *JobRunner) Drain(ctx context.Context) error {
	r.mu.Lock()
	if !r.draining {
		r.draining = true
		for _, cancel := range r.jobs {
			cancel()
		}
		if len(r.jobs) == 0 {
			close(r.done)
		}
	}
	done := r.done
	r.mu.Unlock()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
