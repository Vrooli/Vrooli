package metrics

import (
	"context"
	"errors"
	"io"
	"log"
	"math"
	"sort"
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

// EmbeddingTelemetryPage is one bounded metadata-only slice of the rolling
// telemetry table. RetainedFrom/To describe the whole currently retained
// window; they do not assert that older operations never existed.
type EmbeddingTelemetryPage struct {
	Rows                  []EmbeddingTelemetryRow
	Truncated             bool
	RetainedFrom          time.Time
	RetainedTo            time.Time
	HasRetainedWindow     bool
	ObserverDroppedTotal  uint64
	ObserverWriteFailures uint64
}

// EmbeddingTelemetryProfile groups a returned window by operation, role and
// model. Complete is false when the enclosing bounded read was truncated.
type EmbeddingTelemetryProfile struct {
	Kind          string
	Role          string
	Model         string
	SampleCount   int64
	FailedCount   int64
	DurationP50US int64
	DurationP95US int64
	Complete      bool
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

// ReadWindow returns observations in [from,to), in timestamp order, with an
// explicit row-cap signal and the current rolling-retention bounds. It keeps
// only allowlisted metadata; request content is never present in this store.
func (s *EmbeddingTelemetryStore) ReadWindow(ctx context.Context, from, to time.Time, limit int) (EmbeddingTelemetryPage, error) {
	if s == nil || s.db == nil {
		return EmbeddingTelemetryPage{}, errors.New("embedding telemetry store is unavailable")
	}
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return EmbeddingTelemetryPage{}, errors.New("embedding telemetry window must have a non-zero start before its end")
	}
	if limit < 1 {
		limit = 1000
	}
	if limit > EmbeddingTelemetryLimit {
		limit = EmbeddingTelemetryLimit
	}

	page := EmbeddingTelemetryPage{Rows: make([]EmbeddingTelemetryRow, 0, min(limit, 256))}
	page.ObserverDroppedTotal, page.ObserverWriteFailures = s.Counters()

	// RFC3339Nano emits variable fractional precision. Its strings do not sort
	// chronologically at subsecond boundaries (for example, `.5Z` sorts before
	// `Z`), so read the bounded retained table and compare parsed instants.
	rows, err := s.db.QueryContext(ctx, `SELECT operation_kind, role, model, duration_us, vector_count, failed, created_at FROM embedding_telemetry ORDER BY id ASC`)
	if err != nil {
		return EmbeddingTelemetryPage{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var row EmbeddingTelemetryRow
		var failed int
		var created string
		if err := rows.Scan(&row.Kind, &row.Role, &row.Model, &row.DurationUS, &row.VectorCount, &failed, &created); err != nil {
			return EmbeddingTelemetryPage{}, err
		}
		row.Failed = failed != 0
		row.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return EmbeddingTelemetryPage{}, err
		}
		if !page.HasRetainedWindow || row.CreatedAt.Before(page.RetainedFrom) {
			page.RetainedFrom = row.CreatedAt
		}
		if !page.HasRetainedWindow || row.CreatedAt.After(page.RetainedTo) {
			page.RetainedTo = row.CreatedAt
		}
		page.HasRetainedWindow = true
		if !row.CreatedAt.Before(from) && row.CreatedAt.Before(to) && len(page.Rows) < limit+1 {
			page.Rows = append(page.Rows, row)
		}
	}
	if err := rows.Err(); err != nil {
		return EmbeddingTelemetryPage{}, err
	}
	sort.SliceStable(page.Rows, func(i, j int) bool { return page.Rows[i].CreatedAt.Before(page.Rows[j].CreatedAt) })
	if len(page.Rows) > limit {
		page.Truncated = true
		page.Rows = page.Rows[:limit]
	}
	return page, nil
}

// ProfileEmbeddingTelemetry derives descriptive nearest-rank latency summaries
// from exactly the rows returned by ReadWindow. A truncated page marks every
// group incomplete because the omitted rows may belong to any group.
func ProfileEmbeddingTelemetry(rows []EmbeddingTelemetryRow, complete bool) []EmbeddingTelemetryProfile {
	type profileKey struct{ kind, role, model string }
	groups := make(map[profileKey]*EmbeddingTelemetryProfile)
	durations := make(map[profileKey][]int64)
	for _, row := range rows {
		key := profileKey{row.Kind, row.Role, row.Model}
		profile := groups[key]
		if profile == nil {
			profile = &EmbeddingTelemetryProfile{Kind: row.Kind, Role: row.Role, Model: row.Model, Complete: complete}
			groups[key] = profile
		}
		profile.SampleCount++
		if row.Failed {
			profile.FailedCount++
		}
		durations[key] = append(durations[key], row.DurationUS)
	}
	out := make([]EmbeddingTelemetryProfile, 0, len(groups))
	for key, profile := range groups {
		values := durations[key]
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		profile.DurationP50US = nearestRank(values, 0.50)
		profile.DurationP95US = nearestRank(values, 0.95)
		out = append(out, *profile)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		return out[i].Model < out[j].Model
	})
	return out
}

func nearestRank(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(p*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
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
