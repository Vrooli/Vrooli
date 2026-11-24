package metrics

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/schedule"
	"search-hub/internal/aigateway"
)

// EmbeddingTelemetryLimit is the hard row bound for the metadata-only
// embedding observer. The table is a rolling operational window, not an
// evidence archive; experiment receipts must be exported separately.
const EmbeddingTelemetryLimit = 10000

const embeddingTelemetryQueueLimit = 512

// EmbeddingTelemetryRow is safe to export: it contains no input, output,
// provenance, vector data, or provider error text.
type EmbeddingTelemetryRow struct {
	Kind        string
	Role        string
	Model       string
	DurationUS  int64
	VectorCount int
	Failed      bool
	CreatedAt   time.Time
}

// EmbeddingTelemetryStore persists client-visible embedding observations at
// the Search Hub metrics owner. Record failures are best effort and never
// change embedding behavior.
type EmbeddingTelemetryStore struct {
	db       SQLExecutor
	clock    schedule.Clock
	logger   *log.Logger
	queue    chan embeddingTelemetryEvent
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	closed   bool
	dropped  atomic.Uint64
	failures atomic.Uint64
}

type embeddingTelemetryEvent struct {
	row *EmbeddingTelemetryRow
	ack chan struct{}
}

func NewEmbeddingTelemetryStore(db SQLExecutor, clock schedule.Clock, logger *log.Logger) *EmbeddingTelemetryStore {
	if clock == nil {
		clock = schedule.System()
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	s := &EmbeddingTelemetryStore{db: db, clock: clock, logger: logger, queue: make(chan embeddingTelemetryEvent, embeddingTelemetryQueueLimit), stop: make(chan struct{}), done: make(chan struct{})}
	go s.run()
	return s
}

// Observe returns an observer compatible with the Search Hub-owned typed
// embedding client. Labels are allowlisted/capped before persistence.
func (s *EmbeddingTelemetryStore) Observe(o aigateway.EmbeddingObservation) {
	if s == nil || s.db == nil {
		return
	}
	kind := strings.TrimSpace(o.Kind)
	switch kind {
	case "query", "document", "generic":
	default:
		kind = "other"
	}
	role := boundedLabel(o.Role, 80)
	model := boundedLabel(o.Model, 120)
	durationUS := o.Duration.Microseconds()
	if durationUS < 0 {
		durationUS = 0
	}
	if durationUS > int64((24*time.Hour)/time.Microsecond) {
		durationUS = int64((24 * time.Hour) / time.Microsecond)
	}
	vectors := o.VectorCount
	if vectors < 0 {
		vectors = 0
	}
	if vectors > 4096 {
		vectors = 4096
	}
	failed := 0
	if o.Failed {
		failed = 1
	}
	row := &EmbeddingTelemetryRow{Kind: kind, Role: role, Model: model, DurationUS: durationUS, VectorCount: vectors, Failed: failed != 0, CreatedAt: s.clock.Now().UTC()}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	select {
	case s.queue <- embeddingTelemetryEvent{row: row}:
	default:
		// Keep telemetry from adding queueing latency to the embedding caller.
		s.dropped.Add(1)
	}
}

func (s *EmbeddingTelemetryStore) run() {
	defer close(s.done)
	for {
		select {
		case event := <-s.queue:
			s.write(event)
		case <-s.stop:
			for {
				select {
				case event := <-s.queue:
					s.write(event)
				default:
					return
				}
			}
		}
	}
}

func (s *EmbeddingTelemetryStore) write(event embeddingTelemetryEvent) {
	if event.row != nil {
		row := event.row
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		_, err := s.db.ExecContext(ctx, `INSERT INTO embedding_telemetry (operation_kind, role, model, duration_us, vector_count, failed, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, row.Kind, row.Role, row.Model, row.DurationUS, row.VectorCount, boolInt(row.Failed), row.CreatedAt.Format(time.RFC3339Nano))
		cancel()
		if err != nil {
			// Do not include SQL args or any caller-derived content in logs.
			s.failures.Add(1)
			s.logger.Printf("embedding telemetry write failed: %T", err)
		}
	}
	if event.ack != nil {
		close(event.ack)
	}
}

// Counters exposes in-process missingness so a busy or unhealthy sink cannot
// silently masquerade as a complete latency sample set.
func (s *EmbeddingTelemetryStore) Counters() (dropped, writeFailures uint64) {
	if s == nil {
		return 0, 0
	}
	return s.dropped.Load(), s.failures.Load()
}

// Flush waits until all events queued before the barrier have been persisted.
func (s *EmbeddingTelemetryStore) Flush(ctx context.Context) error {
	ack := make(chan struct{})
	s.mu.Lock()
	if s.closed {
		done := s.done
		s.mu.Unlock()
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	select {
	case s.queue <- embeddingTelemetryEvent{ack: ack}:
		s.mu.Unlock()
	case <-ctx.Done():
		s.mu.Unlock()
		return ctx.Err()
	}
	select {
	case <-ack:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops accepting observations and drains the bounded queue.
func (s *EmbeddingTelemetryStore) Close(ctx context.Context) error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.stop)
	}
	s.mu.Unlock()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RecentEmbeddings returns a bounded newest-first window for owner diagnostics
// and retained run receipts.
func (s *EmbeddingTelemetryStore) RecentEmbeddings(ctx context.Context, limit int) ([]EmbeddingTelemetryRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("embedding telemetry store is unavailable")
	}
	if limit < 1 {
		limit = 1
	}
	if limit > EmbeddingTelemetryLimit {
		limit = EmbeddingTelemetryLimit
	}
	rows, err := s.db.QueryContext(ctx, `SELECT operation_kind, role, model, duration_us, vector_count, failed, created_at FROM embedding_telemetry ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]EmbeddingTelemetryRow, 0, limit)
	for rows.Next() {
		var row EmbeddingTelemetryRow
		var failed int
		var created string
		if err := rows.Scan(&row.Kind, &row.Role, &row.Model, &row.DurationUS, &row.VectorCount, &failed, &created); err != nil {
			return nil, err
		}
		row.Failed = failed != 0
		row.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func boundedLabel(value string, max int) string {
	value = strings.TrimSpace(value)
	var out strings.Builder
	for _, r := range value {
		if out.Len() >= max {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', strings.ContainsRune("._:/-", r):
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	return out.String()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
