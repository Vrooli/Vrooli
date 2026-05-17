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
	"sync/atomic"
	"time"

	"audio-tools/internal/store"
)

// Stats reports observability counters for the recorder.
type Stats struct {
	// DroppedTotal counts rows dropped because the bounded queue was
	// full at the moment of Enqueue (drop-newest policy, N=1024).
	DroppedTotal uint64
	// EnqueuedTotal counts rows accepted into the queue.
	EnqueuedTotal uint64
	// QueueCapacity is the bounded buffer size (compile-time constant).
	QueueCapacity int
	// QueueDepth is the instantaneous depth of the queue.
	QueueDepth int
}

// QueueCapacity is the bounded buffer size for the async queue. The
// policy is drop-newest: when the queue is full, Enqueue returns
// immediately and increments DroppedTotal. See Stats().
const QueueCapacity = 1024

// seam: Recorder is the usage-recording seam (SEAMS.md row
// "usagereport.Recorder"). Production wires the SQLite-backed recorder;
// tests wire mocks.FakeRecorder or accept the real recorder with a
// fake clock.
//
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
	repo          *store.UsageStore
	logger        *log.Logger
	queue         chan store.UsageRow
	wg            sync.WaitGroup
	retries       []time.Duration
	droppedTotal  atomic.Uint64
	enqueuedTotal atomic.Uint64
}

// New returns an AsyncRecorder running a single drain goroutine.
func New(repo *store.UsageStore, logger *log.Logger) *AsyncRecorder {
	if logger == nil {
		logger = log.Default()
	}
	r := &AsyncRecorder{
		repo:    repo,
		logger:  logger,
		queue:   make(chan store.UsageRow, QueueCapacity),
		retries: []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second},
	}
	r.wg.Add(1)
	go r.drain()
	return r
}

func (r *AsyncRecorder) Enqueue(row store.UsageRow) {
	select {
	case r.queue <- row:
		r.enqueuedTotal.Add(1)
	default:
		r.droppedTotal.Add(1)
		r.logger.Printf("usagereport: queue full, dropping op=%s", row.OperationID)
	}
}

// Stats returns a snapshot of observability counters.
func (r *AsyncRecorder) Stats() Stats {
	return Stats{
		DroppedTotal:  r.droppedTotal.Load(),
		EnqueuedTotal: r.enqueuedTotal.Load(),
		QueueCapacity: cap(r.queue),
		QueueDepth:    len(r.queue),
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
