package measures

import (
	"context"
	"sync"
)

// Projection is a read-model fold over the event log: an adopter implements
// Apply (incrementally update aggregate state from one event) and Reset (clear
// it for a full rebuild). It is the seam through which a measure turns an
// append-only log into a queryable aggregate. The projection owns its own
// query surface — measures-go only drives the watermark fold.
type Projection interface {
	// Apply folds a single event into the projection's aggregate state. It is
	// called in ID order, exactly once per event, under the ReadModel's write
	// lock.
	Apply(e Event)
	// Reset clears all aggregate state ahead of a full Rebuild.
	Reset()
}

// DefaultReadModelBatch is the page size used when folding events from the log.
const DefaultReadModelBatch = 5000

// ReadModel drives the watermark-based incremental aggregation pattern
// generalized from swarm-manager's stats engine: Rebuild replays the whole log
// once at startup; Refresh folds only events appended since the last watermark.
// The adopter supplies a Projection; ReadModel owns the watermark, the batching,
// and the locking.
type ReadModel struct {
	mu        sync.RWMutex
	watermark int64
	log       EventLog
	proj      Projection
	batch     int
}

// ReadModelOption configures a ReadModel.
type ReadModelOption func(*ReadModel)

// WithBatchSize sets the event page size for Rebuild/Refresh (default
// DefaultReadModelBatch).
func WithBatchSize(n int) ReadModelOption {
	return func(m *ReadModel) {
		if n > 0 {
			m.batch = n
		}
	}
}

// NewReadModel binds a Projection to an EventLog. Call Rebuild once at startup,
// then Refresh before each read.
func NewReadModel(log EventLog, proj Projection, opts ...ReadModelOption) *ReadModel {
	m := &ReadModel{log: log, proj: proj, batch: DefaultReadModelBatch}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Rebuild resets the projection and replays every event from scratch, advancing
// the watermark to the log's MaxID. Call once at startup.
func (m *ReadModel) Rebuild(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.proj.Reset()
	m.watermark = 0
	for {
		events, err := m.log.Since(ctx, m.watermark, m.batch)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}
		for i := range events {
			m.proj.Apply(events[i])
		}
		m.watermark = events[len(events)-1].ID
		if len(events) < m.batch {
			break
		}
	}
	return nil
}

// Refresh folds events appended since the last watermark into the projection.
// Call before each read so the aggregate reflects all appended events.
func (m *ReadModel) Refresh(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		events, err := m.log.Since(ctx, m.watermark, m.batch)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		for i := range events {
			m.proj.Apply(events[i])
		}
		m.watermark = events[len(events)-1].ID
		if len(events) < m.batch {
			return nil
		}
	}
}

// Watermark returns the ID of the last event folded into the projection.
func (m *ReadModel) Watermark() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.watermark
}

// Read runs fn under the read lock, giving a measure's compute func a consistent
// view of the projection while Refresh/Rebuild are excluded. The projection
// itself is captured by the adopter's closure; ReadModel only guards concurrency.
func (m *ReadModel) Read(fn func()) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fn()
}
