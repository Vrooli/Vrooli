package main

import (
	"sync"
	"time"
)

// R3FEvent is a single performance telemetry sample posted from the UI.
type R3FEvent struct {
	Tier        int       `json:"tier"`
	FPS         float64   `json:"fps"`
	TS          time.Time `json:"ts"`
	Route       string    `json:"route,omitempty"`
	FrameDrops  int       `json:"frame_drops,omitempty"`
	DowngradeTo *int      `json:"downgrade_to,omitempty"`
}

// StatsBuffer is a bounded ring buffer of R3F events with a soft TTL.
type StatsBuffer struct {
	mu       sync.Mutex
	events   []R3FEvent
	capacity int
	ttl      time.Duration
}

// NewStatsBuffer returns a buffer bounded by capacity events; events older
// than ttl are evicted on every Append.
func NewStatsBuffer(capacity int, ttl time.Duration) *StatsBuffer {
	if capacity <= 0 {
		capacity = 1024
	}
	return &StatsBuffer{
		events:   make([]R3FEvent, 0, capacity),
		capacity: capacity,
		ttl:      ttl,
	}
}

// Append records an event and prunes expired entries.
func (b *StatsBuffer) Append(e R3FEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if e.TS.IsZero() {
		e.TS = time.Now().UTC()
	}
	b.events = append(b.events, e)

	cutoff := time.Now().Add(-b.ttl)
	idx := 0
	for i, ev := range b.events {
		if ev.TS.After(cutoff) {
			idx = i
			break
		}
		idx = i + 1
	}
	if idx > 0 {
		b.events = append([]R3FEvent(nil), b.events[idx:]...)
	}

	if len(b.events) > b.capacity {
		overflow := len(b.events) - b.capacity
		b.events = append([]R3FEvent(nil), b.events[overflow:]...)
	}
}

// Snapshot returns a copy of the buffer contents, newest-first.
func (b *StatsBuffer) Snapshot() []R3FEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]R3FEvent, len(b.events))
	for i, ev := range b.events {
		out[len(b.events)-1-i] = ev
	}
	return out
}

// Len returns the current buffer size (for tests).
func (b *StatsBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}
