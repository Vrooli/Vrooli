package systemevents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

type Store interface {
	UpsertSystemEvents(ctx context.Context, events []Event) (inserted int, deduped int, err error)
	UpsertSystemEventSource(ctx context.Context, source SourceStatus) error
	ListSystemEvents(ctx context.Context, filters Filters) (*Response, error)
	GetSystemEventSources(ctx context.Context) ([]SourceStatus, error)
	CursorStore
}

// ExecAvoider is implemented by collectors that track journalctl invocations
// skipped via incremental ingestion (already-scanned boots + cursor reads).
type ExecAvoider interface {
	ExecsAvoided() int64
}

type Collector interface {
	Collect(ctx context.Context) ([]Event, []SourceStatus)
}

// DefaultIngestInterval throttles tick-driven kernel-signal ingestion. The
// 60s tick re-greps the journal far more often than host events change; gating
// behind a coarser interval (mirroring the 30s host-inventory cache) gives an
// immediate exec reduction with zero feature loss. Overridable via
// AUTOHEAL_SYSTEMEVENTS_INTERVAL (clamped to [MinIngestInterval, MaxIngestInterval]).
const (
	DefaultIngestInterval = 300 * time.Second
	MinIngestInterval     = 300 * time.Second
	MaxIngestInterval     = 600 * time.Second
)

type Service struct {
	store      Store
	collectors []Collector
	now        func() time.Time

	// interval gates IngestIfDue (the tick path). lastIngest is the last time
	// an ingest actually ran. mu guards lastIngest against the (rare) concurrent
	// tick/refresh overlap.
	interval   time.Duration
	mu         sync.Mutex
	lastIngest time.Time
}

func NewService(store Store, plat *platform.Capabilities) *Service {
	return &Service{
		store: store,
		collectors: []Collector{
			NewHostCollectorWithCursors(plat, checks.DefaultExecutor, journal.NewPlatformReader(), store),
		},
		now:      func() time.Time { return time.Now().UTC() },
		interval: DefaultIngestInterval,
	}
}

func NewServiceWithCollectors(store Store, collectors []Collector, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, collectors: collectors, now: now, interval: DefaultIngestInterval}
}

// SetIngestInterval overrides the tick-path ingest throttle, clamped to
// [MinIngestInterval, MaxIngestInterval]. A non-positive value resets to the
// default.
func (s *Service) SetIngestInterval(d time.Duration) {
	if d <= 0 {
		d = DefaultIngestInterval
	}
	if d < MinIngestInterval {
		d = MinIngestInterval
	}
	if d > MaxIngestInterval {
		d = MaxIngestInterval
	}
	s.mu.Lock()
	s.interval = d
	s.mu.Unlock()
}

// IngestIfDue runs Ingest only if at least the configured interval has elapsed
// since the last ingest. It is the tick-path entry point: the 60s tick calls
// it, but the underlying journalctl work runs at most once per interval. The
// explicit refresh endpoint and startup still call Ingest directly to force a
// fresh read. The bool reports whether an ingest actually ran.
func (s *Service) IngestIfDue(ctx context.Context) (*IngestSummary, bool, error) {
	if s == nil {
		return nil, false, fmt.Errorf("system event service unavailable")
	}
	s.mu.Lock()
	interval := s.interval
	if interval <= 0 {
		interval = DefaultIngestInterval
	}
	if !s.lastIngest.IsZero() && s.now().Sub(s.lastIngest) < interval {
		s.mu.Unlock()
		return nil, false, nil
	}
	s.mu.Unlock()

	summary, err := s.Ingest(ctx)
	return summary, true, err
}

// ExecsAvoided sums the journalctl execs avoided across collectors that track
// it (the host collector's already-scanned-boot skips + incremental reads).
func (s *Service) ExecsAvoided() int64 {
	if s == nil {
		return 0
	}
	var total int64
	for _, collector := range s.collectors {
		if avoider, ok := collector.(ExecAvoider); ok {
			total += avoider.ExecsAvoided()
		}
	}
	return total
}

func (s *Service) Ingest(ctx context.Context) (*IngestSummary, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("system event service unavailable")
	}
	start := s.now()
	var all []Event
	var sources []SourceStatus
	for _, collector := range s.collectors {
		if collector == nil {
			continue
		}
		events, statuses := collector.Collect(ctx)
		all = append(all, events...)
		sources = append(sources, statuses...)
	}
	for i := range all {
		normalizeEvent(&all[i], start)
	}
	inserted, deduped, err := s.store.UpsertSystemEvents(ctx, all)
	if err != nil {
		return nil, err
	}
	for _, status := range sources {
		if status.LastIngestedAt.IsZero() {
			status.LastIngestedAt = start
		}
		if err := s.store.UpsertSystemEventSource(ctx, status); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	s.lastIngest = s.now()
	s.mu.Unlock()
	return &IngestSummary{
		Ingested:     inserted,
		Deduped:      deduped,
		Sources:      sources,
		DurationMs:   s.now().Sub(start).Milliseconds(),
		ExecsAvoided: s.ExecsAvoided(),
	}, nil
}

func normalizeEvent(event *Event, ingestedAt time.Time) {
	event.Source = strings.TrimSpace(event.Source)
	event.Platform = strings.TrimSpace(event.Platform)
	event.Category = strings.TrimSpace(event.Category)
	event.Title = strings.TrimSpace(event.Title)
	event.Summary = strings.TrimSpace(event.Summary)
	if event.Source == "" {
		event.Source = "unknown"
	}
	if event.Platform == "" {
		event.Platform = "other"
	}
	if event.Category == "" {
		event.Category = "system"
	}
	if event.Severity == "" {
		event.Severity = SeverityInfo
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = ingestedAt
	}
	event.OccurredAt = event.OccurredAt.UTC()
	event.IngestedAt = ingestedAt.UTC()
	if event.Fingerprint == "" {
		event.Fingerprint = Fingerprint(*event)
	}
}

func Fingerprint(event Event) string {
	details, _ := json.Marshal(event.Details)
	parts := []string{
		event.Source,
		event.Platform,
		event.Category,
		string(event.Severity),
		event.Title,
		event.Summary,
		event.BootID,
		event.OccurredAt.UTC().Format(time.RFC3339Nano),
		string(details),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:16])
}

func BuildCorrelations(events []Event) []Correlation {
	if len(events) == 0 {
		return nil
	}
	ordered := append([]Event(nil), events...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].OccurredAt.Before(ordered[j].OccurredAt) })
	var out []Correlation
	add := func(title, summary, rationale string, a, b Event) {
		delta := b.OccurredAt.Sub(a.OccurredAt)
		out = append(out, Correlation{
			Title:        title,
			Summary:      summary,
			Rationale:    rationale,
			EventIDs:     []int64{a.ID, b.ID},
			EventSources: []string{a.Source, b.Source},
			TimeDelta:    delta.Round(time.Second).String(),
			Confidence:   "temporal",
		})
	}
	firstCrash := firstMatching(ordered, func(e Event) bool {
		return e.Category == "crash" || strings.Contains(strings.ToLower(e.Summary), "unclean")
	})
	if firstCrash != nil {
		if kernel := nearestBefore(ordered, *firstCrash, func(e Event) bool { return e.Category == "kernel" }); kernel != nil {
			add("Kernel change before crash", "A kernel event occurred before the first crash/reset event in this window.", "Temporal proximity only; verify boot IDs, hardware signals, and module state before treating this as causal.", *kernel, *firstCrash)
		}
		if driver := nearestBefore(ordered, *firstCrash, func(e Event) bool { return e.Category == "driver" }); driver != nil {
			add("Driver change before crash", "A driver/module event occurred before the first crash/reset event in this window.", "Temporal proximity only; driver updates can expose lower-level hardware or firmware faults without being the sole root cause.", *driver, *firstCrash)
		}
		if firmware := nearestAfter(ordered, *firstCrash, func(e Event) bool { return e.Category == "firmware" }); firmware != nil {
			add("Firmware changed after crashes began", "A firmware package event occurred after the first crash/reset event in this window.", "This weakens firmware update as the initial trigger for this window, though it may affect later behavior.", *firstCrash, *firmware)
		}
	}
	if missing := firstMatching(ordered, func(e Event) bool {
		text := strings.ToLower(e.Summary + " " + e.Title)
		return strings.Contains(text, "nvidia") && strings.Contains(text, "module") && strings.Contains(text, "missing")
	}); missing != nil {
		if repair := nearestAfter(ordered, *missing, func(e Event) bool {
			text := strings.ToLower(e.Summary + " " + e.Title)
			return e.Category == "driver" && strings.Contains(text, "nvidia") && (strings.Contains(text, "install") || strings.Contains(text, "upgrade"))
		}); repair != nil {
			add("NVIDIA module gap later repaired", "A missing NVIDIA module signal was followed by a NVIDIA package/module install or upgrade.", "This identifies a repair sequence, not proof that NVIDIA was the original reset cause.", *missing, *repair)
		}
	}
	return out
}

func firstMatching(events []Event, fn func(Event) bool) *Event {
	for i := range events {
		if fn(events[i]) {
			return &events[i]
		}
	}
	return nil
}

func nearestBefore(events []Event, pivot Event, fn func(Event) bool) *Event {
	var candidate *Event
	for i := range events {
		if !events[i].OccurredAt.Before(pivot.OccurredAt) || !fn(events[i]) {
			continue
		}
		candidate = &events[i]
	}
	return candidate
}

func nearestAfter(events []Event, pivot Event, fn func(Event) bool) *Event {
	for i := range events {
		if events[i].OccurredAt.After(pivot.OccurredAt) && fn(events[i]) {
			return &events[i]
		}
	}
	return nil
}
