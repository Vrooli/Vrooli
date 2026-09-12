// Package jobs owns admission for heavy catalog work.
package jobs

import (
	"context"
	"database/sql"
	"sync"
)

// Runner serializes heavyweight catalog jobs. The database is retained here
// as the job-owned handle so future job implementations cannot accidentally
// fall back to the serving pool.
type Runner struct {
	db  *sql.DB
	sem chan struct{}
}

func New(db *sql.DB) *Runner { return &Runner{db: db, sem: make(chan struct{}, 1)} }

// Acquire admits one matrix and honors cancellation while waiting. Callers
// must pair a successful Acquire with Release.
func (r *Runner) Acquire(ctx context.Context) error {
	if r == nil {
		return nil
	}
	select {
	case r.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) Release() {
	if r != nil {
		<-r.sem
	}
}

func (r *Runner) Database() *sql.DB {
	if r == nil {
		return nil
	}
	return r.db
}

// RunMatrix admits one bounded matrix and runs its indexes with a small
// worker pool. The callback receives the request context so cancellation
// reaches the actual gate work, not only the dispatcher.
func (r *Runner) RunMatrix(ctx context.Context, count, workers int, run func(context.Context, int) error) error {
	if err := r.Acquire(ctx); err != nil {
		return err
	}
	defer r.Release()
	if count == 0 {
		return nil
	}
	if workers < 1 {
		workers = 1
	}
	if workers > count {
		workers = count
	}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var once sync.Once
	var firstErr error
	var mu sync.Mutex
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				if workCtx.Err() != nil {
					return
				}
				if err := run(workCtx, index); err != nil {
					once.Do(func() {
						mu.Lock()
						firstErr = err
						mu.Unlock()
					})
					cancel()
					return
				}
			}
		}()
	}
	for index := 0; index < count; index++ {
		select {
		case jobs <- index:
		case <-workCtx.Done():
			close(jobs)
			group.Wait()
			mu.Lock()
			err := firstErr
			mu.Unlock()
			if err != nil {
				return err
			}
			return ctx.Err()
		}
	}
	close(jobs)
	group.Wait()
	mu.Lock()
	defer mu.Unlock()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}
