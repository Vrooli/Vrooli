package findings

import (
	"context"
	"log"
	"time"
)

// surfaceWriter is the narrow persistence surface the recorder needs: the
// Repository satisfies it. It stamps the surfacing time from its own clock.
type surfaceWriter interface {
	RecordSurfaced(ctx context.Context, ids []string) error
}

// DefaultUsageBuffer is the recorder's channel depth: how many surfacing batches
// can be queued before new ones are dropped (best-effort telemetry).
const DefaultUsageBuffer = 256

// UsageRecorder is the OT-P2-001 async surfacing worker. The search read path
// calls Surfaced(ids) — a NON-BLOCKING enqueue — and a background goroutine
// drains the queue and persists the increments. This keeps surfacing telemetry
// entirely off the hot path: a search response never waits on (or is failed by)
// a usage write, and a full buffer drops the event rather than blocking.
type UsageRecorder struct {
	writer surfaceWriter
	ch     chan []string
	logger *log.Logger
}

// NewUsageRecorder constructs the recorder. Call Run in a goroutine to start
// draining; until then Surfaced enqueues up to the buffer depth. The surfacing
// timestamp is stamped by the writer's own clock.
func NewUsageRecorder(w surfaceWriter, buffer int, logger *log.Logger) *UsageRecorder {
	if buffer <= 0 {
		buffer = DefaultUsageBuffer
	}
	if logger == nil {
		logger = log.Default()
	}
	return &UsageRecorder{
		writer: w,
		ch:     make(chan []string, buffer),
		logger: logger,
	}
}

// Surfaced enqueues a surfacing event for the given finding ids without
// blocking. A nil recorder or empty id list is a no-op; a full buffer drops the
// event (telemetry is best-effort, the read path must never stall on it). The
// ids are copied so the caller may reuse its slice.
func (r *UsageRecorder) Surfaced(ids []string) {
	if r == nil || len(ids) == 0 {
		return
	}
	cp := make([]string, len(ids))
	copy(cp, ids)
	select {
	case r.ch <- cp:
	default:
		// Buffer full: drop rather than block the search response.
	}
}

// Run drains the surfacing queue until ctx is cancelled, persisting each batch.
// Each write gets its own bounded context so a slow DB cannot wedge the worker.
func (r *UsageRecorder) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case batch := <-r.ch:
			r.flush(batch)
		}
	}
}

func (r *UsageRecorder) flush(batch []string) {
	wctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.writer.RecordSurfaced(wctx, batch); err != nil {
		r.logger.Printf("findings.UsageRecorder: record surfaced (%d ids): %v", len(batch), err)
	}
}
