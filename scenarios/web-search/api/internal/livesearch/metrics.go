package livesearch

import (
	"sync"
	"time"
)

// MetricKind identifies one bounded live-search event. Values are deliberately
// owner-defined and low-cardinality so source text never becomes a metric label.
type MetricKind string

const (
	MetricCacheHit        MetricKind = "cache_hit"
	MetricCacheMiss       MetricKind = "cache_miss"
	MetricGovernorRefusal MetricKind = "governor_refusal"
	MetricUpstreamCall    MetricKind = "upstream_call"
)

const maxMetricEvents = 8192

type metricEvent struct {
	kind MetricKind
	at   time.Time
}

// Metrics is the live-search owner's bounded operational event recorder. It is
// intentionally ephemeral: fixed-window readings describe the current process
// and report unavailable when the recorder has discarded history.
type Metrics struct {
	mu        sync.Mutex
	events    []metricEvent
	truncated bool
	now       func() time.Time
}

func NewMetrics(now ...func() time.Time) *Metrics {
	m := &Metrics{}
	if len(now) > 0 {
		m.now = now[0]
	}
	return m
}

func (m *Metrics) RecordNow(kind MetricKind) {
	if m == nil {
		return
	}
	at := time.Now
	if m.now != nil {
		at = m.now
	}
	m.Record(kind, at())
}

func (m *Metrics) Record(kind MetricKind, at time.Time) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) >= maxMetricEvents {
		copy(m.events, m.events[1:])
		m.events = m.events[:len(m.events)-1]
		m.truncated = true
	}
	m.events = append(m.events, metricEvent{kind: kind, at: at.UTC()})
}

// Snapshot returns counts for the half-open [from,to) window and whether the
// bounded history can still be treated as complete.
func (m *Metrics) Snapshot(from, to time.Time) (map[MetricKind]int, bool) {
	counts := map[MetricKind]int{}
	if m == nil {
		return counts, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, event := range m.events {
		if !event.at.Before(from) && event.at.Before(to) {
			counts[event.kind]++
		}
	}
	return counts, !m.truncated
}
