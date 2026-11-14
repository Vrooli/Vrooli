package runtimeactivity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/maintenance"
)

type observation struct {
	state maintenance.Inventory
	err   error
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
