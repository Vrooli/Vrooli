package runtimeactivity

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/maintenance"
)

type observation struct {
	state maintenance.Inventory
	err   error
}

type countingObservation struct {
	calls atomic.Int32
	state maintenance.Inventory
}

func (o *countingObservation) ObserveActivity(context.Context) (maintenance.Inventory, error) {
	o.calls.Add(1)
	return o.state, nil
}

func TestReadReusesRecentPhysicalObservationAndRefreshesAfterExpiry(t *testing.T) {
	at := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	observer := &countingObservation{state: maintenance.Inventory{ObservedAt: at, PhysicalComplete: true, ObservedExecutors: 2}}
	reader := NewWithCacheTTL(observer, time.Minute)
	reader.now = func() time.Time { return at }

	first := reader.Read(t.Context())
	second := reader.Read(t.Context())
	if observer.calls.Load() != 1 {
		t.Fatalf("rapid reads launched %d physical observations, want 1", observer.calls.Load())
	}
	if first.ExecutingRuns == nil || second.ExecutingRuns == nil || *first.ExecutingRuns != *second.ExecutingRuns || !second.ObservedAt.Equal(at) {
		t.Fatalf("cached snapshot lost source evidence: first=%+v second=%+v", first, second)
	}
	reader.now = func() time.Time { return at.Add(time.Minute + time.Nanosecond) }
	third := reader.Read(t.Context())
	if observer.calls.Load() != 2 || !third.Complete {
		t.Fatalf("expired cache did not refresh: calls=%d snapshot=%+v", observer.calls.Load(), third)
	}
}

type gatedObservation struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
	state   maintenance.Inventory
}

func (o *gatedObservation) ObserveActivity(context.Context) (maintenance.Inventory, error) {
	if o.calls.Add(1) == 1 {
		close(o.started)
	}
	<-o.release
	return o.state, nil
}

func TestConcurrentReadsShareOnePhysicalObservation(t *testing.T) {
	at := time.Now().UTC()
	observer := &gatedObservation{
		started: make(chan struct{}), release: make(chan struct{}),
		state: maintenance.Inventory{ObservedAt: at, PhysicalComplete: true, ObservedExecutors: 1},
	}
	reader := New(observer)
	const readers = 8
	results := make(chan Snapshot, readers)
	for range readers {
		go func() { results <- reader.Read(t.Context()) }()
	}
	<-observer.started
	time.Sleep(10 * time.Millisecond)
	close(observer.release)
	for range readers {
		if snapshot := <-results; !snapshot.Complete {
			t.Fatalf("coalesced read lost complete evidence: %+v", snapshot)
		}
	}
	if got := observer.calls.Load(); got != 1 {
		t.Fatalf("concurrent reads launched %d physical observations, want 1", got)
	}
}

func (o observation) ObserveActivity(context.Context) (maintenance.Inventory, error) {
	return o.state, o.err
}
func TestReadPreservesPhysicalCountAndUnknownRatherThanLogicalDrainCount(t *testing.T) {
	at := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	logical := 64
	for _, count := range []int{0, 3, 100} {
		snapshot := New(observation{state: maintenance.Inventory{ObservedAt: at, PhysicalComplete: true, ObservedExecutors: count, LogicalQueuedRuns: 17, Remaining: &logical}}).Read(t.Context())
		if !snapshot.Complete || snapshot.ExecutingRuns == nil || *snapshot.ExecutingRuns != uint32(count) || snapshot.ObservedExecutingRuns != uint32(count) || snapshot.LogicalQueuedRuns != 17 || !snapshot.ObservedAt.Equal(at) {
			t.Fatalf("wrong runtime projection: %+v", snapshot)
		}
	}
	for _, o := range []observation{
		{state: maintenance.Inventory{ObservedAt: at, ObservedExecutors: 0}},
		{state: maintenance.Inventory{ObservedAt: at, ObservedExecutors: 3, Unknown: []string{"scope unknown"}}, err: errors.New("partial physical scope")},
		{state: maintenance.Inventory{ObservedAt: at, PhysicalComplete: true}, err: errors.New("owner failed")},
		{state: maintenance.Inventory{PhysicalComplete: true}},
		{state: maintenance.Inventory{ObservedAt: at, PhysicalComplete: true, ObservedExecutors: -1}},
	} {
		snapshot := New(o).Read(t.Context())
		if snapshot.Complete || snapshot.ExecutingRuns != nil || len(snapshot.Unknown) == 0 {
			t.Fatalf("incomplete observation became an activity total: %+v", snapshot)
		}
	}
}
func TestReadBoundsUnknownAndHonorsCancellation(t *testing.T) {
	reasons := make([]string, 70)
	for i := range reasons {
		reasons[i] = strings.Repeat("観", 700)
	}
	snapshot := New(observation{state: maintenance.Inventory{ObservedAt: time.Now(), Unknown: reasons}}).Read(t.Context())
	if len(snapshot.Unknown) != 64 || len([]rune(snapshot.Unknown[0])) > 530 {
		t.Fatalf("unbounded reasons: %d/%d", len(snapshot.Unknown), len([]rune(snapshot.Unknown[0])))
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	snapshot = New(observation{state: maintenance.Inventory{ObservedAt: time.Now(), PhysicalComplete: true}}).Read(ctx)
	if snapshot.Complete || snapshot.ExecutingRuns != nil {
		t.Fatal("cancelled read established a complete count")
	}
}
