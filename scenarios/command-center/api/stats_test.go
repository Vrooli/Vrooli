package main

import (
	"testing"
	"time"
)

func TestStatsBuffer_AppendAndSnapshot(t *testing.T) {
	b := NewStatsBuffer(10, time.Hour)
	now := time.Now().UTC()
	b.Append(R3FEvent{Tier: 1, FPS: 30.0, TS: now.Add(-3 * time.Second)})
	b.Append(R3FEvent{Tier: 2, FPS: 60.0, TS: now.Add(-1 * time.Second)})

	snap := b.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 events, got %d", len(snap))
	}
	// Newest-first
	if snap[0].FPS != 60.0 {
		t.Errorf("expected newest first, got FPS=%v", snap[0].FPS)
	}
}

func TestStatsBuffer_DropsExpired(t *testing.T) {
	b := NewStatsBuffer(100, 500*time.Millisecond)
	b.Append(R3FEvent{Tier: 1, FPS: 10.0, TS: time.Now().Add(-2 * time.Second)})
	b.Append(R3FEvent{Tier: 1, FPS: 20.0, TS: time.Now()})
	if got := b.Len(); got != 1 {
		t.Errorf("expected eviction of stale entry, got len=%d", got)
	}
}

func TestStatsBuffer_CapacityTrims(t *testing.T) {
	b := NewStatsBuffer(3, time.Hour)
	for i := 0; i < 10; i++ {
		b.Append(R3FEvent{Tier: i, FPS: float64(i), TS: time.Now().UTC()})
	}
	if got := b.Len(); got != 3 {
		t.Errorf("expected cap=3, got %d", got)
	}
}

func TestStatsBuffer_DefaultTS(t *testing.T) {
	b := NewStatsBuffer(10, time.Hour)
	b.Append(R3FEvent{Tier: 1, FPS: 60.0})
	snap := b.Snapshot()
	if snap[0].TS.IsZero() {
		t.Error("expected TS to default to now when zero")
	}
}
