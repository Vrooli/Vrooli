// Package usagereport owns the async pipeline that persists usage rows
// from the provider chains. Chains call Recorder.Enqueue after each
// Execute; a single drain goroutine flushes to store.UsageStore with
// a 500ms/1s/2s retry policy. Insert is idempotent on operation_id so
// retries are safe.
package usagereport

import (
	"context"
	"log"
	"sync"
	"time"

	"audio-tools/internal/store"
)

// Recorder accepts usage rows and persists them.
type Recorder interface {
	// Enqueue is non-blocking and never returns an error; failures are
	// retried and ultimately dropped after the configured policy is
	// exhausted (logged).
	Enqueue(row store.UsageRow)
	// Record persists synchronously. Used by tests and CLI paths.
	Record(ctx context.Context, row store.UsageRow) error
	// Close stops the drain goroutine and flushes pending rows.
	Close()
}

// AsyncRecorder writes through a buffered channel.
type AsyncRecorder struct {
	repo    *store.UsageStore
	logger  *log.Logger
	queue   chan store.UsageRow
	wg      sync.WaitGroup
	retries []time.Duration
}

// New returns an AsyncRecorder running a single drain goroutine.
func New(repo *store.UsageStore, logger *log.Logger) *AsyncRecorder {
	if logger == nil {
		logger = log.Default()
	}
	r := &AsyncRecorder{
		repo:    repo,
		logger:  logger,
		queue:   make(chan store.UsageRow, 1024),
		retries: []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second},
	}
	r.wg.Add(1)
	go r.drain()
	return r
}

func (r *AsyncRecorder) Enqueue(row store.UsageRow) {
	select {
	case r.queue <- row:
	default:
		r.logger.Printf("usagereport: queue full, dropping op=%s", row.OperationID)
	}
}

func (r *AsyncRecorder) Record(ctx context.Context, row store.UsageRow) error {
	return r.repo.Insert(ctx, row)
}

func (r *AsyncRecorder) Close() {
	close(r.queue)
	r.wg.Wait()
}

func (r *AsyncRecorder) drain() {
	defer r.wg.Done()
	for row := range r.queue {
		r.flush(row)
	}
}

func (r *AsyncRecorder) flush(row store.UsageRow) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.repo.Insert(ctx, row); err == nil {
		return
	}
	for _, d := range r.retries {
		time.Sleep(d)
		ctx2, c2 := context.WithTimeout(context.Background(), 5*time.Second)
		err := r.repo.Insert(ctx2, row)
		c2()
		if err == nil {
			return
		}
	}
	r.logger.Printf("usagereport: dropping op=%s after retries", row.OperationID)
}
