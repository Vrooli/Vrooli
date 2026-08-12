package queue_test

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/queue"
	"vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func watchdog(t *testing.T, entry queue.DurableEntry, aborter *fakeAborter) (*queue.Watchdog, *fakeDurableStore, *queue.Scheduler, time.Time) {
	t.Helper()
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	clk := mocks.NewFakeClock(now)
	store := &fakeDurableStore{entries: []queue.DurableEntry{entry}}
	scheduler, err := queue.NewSchedulerWithStore(&fakePusher{failNodes: map[string]bool{}, available: map[string]bool{"n1": false}}, aborter, clk, 1, store)
	require.NoError(t, err)
	w := queue.NewWatchdog(store, scheduler, aborter, clk, queue.WatchdogConfig{
		DeliveryLease: time.Second, Interval: time.Second, MaxAttempts: 3,
		StartDeadline: 10 * time.Second, DeadlineGrace: time.Second,
	}, nil)
	return w, store, scheduler, now
}

func TestWatchdog_ExpiredLeaseRedeliversBeforeTerminalFailure(t *testing.T) {
	aborter := &fakeAborter{}
	w, store, scheduler, now := watchdog(t, queue.DurableEntry{
		Job: job("lease-1", "n1"), State: queue.StateRunning,
		LeaseExpiresAt: time.Unix(1, 0), DeliveryAttempts: 1,
	}, aborter)
	store.entries[0].LeaseExpiresAt = now.Add(-time.Second)

	require.NoError(t, w.Sweep(context.Background()))
	require.Equal(t, []string{"lease-1"}, store.queued)
	require.Empty(t, store.failed)
	require.Empty(t, aborter.abortedRuns())
	require.Len(t, scheduler.Snapshot("n1"), 1)
	require.Equal(t, 1, scheduler.Snapshot("n1")[0].Queued)
}

func TestWatchdog_MaxDeliveryAttemptsTerminalizesLostNode(t *testing.T) {
	aborter := &fakeAborter{}
	w, store, scheduler, _ := watchdog(t, queue.DurableEntry{
		Job: job("lease-max", "n1"), State: queue.StateRunning,
		LeaseExpiresAt: time.Unix(1, 0), DeliveryAttempts: 3,
	}, aborter)
	require.NoError(t, w.Sweep(context.Background()))
	require.Equal(t, []string{"lease-max"}, store.failed)
	require.Empty(t, scheduler.Snapshot("n1"))
}

func TestWatchdog_AckWithoutStartTerminalizesNoStart(t *testing.T) {
	aborter := &fakeAborter{}
	w, store, scheduler, _ := watchdog(t, queue.DurableEntry{
		Job: job("no-start", "n1"), State: queue.StateRunning,
		Acked: true, AckedAt: time.Unix(1, 0), DeliveryAttempts: 1,
	}, aborter)
	require.NoError(t, w.Sweep(context.Background()))
	require.Equal(t, []string{"no-start"}, store.failed)
	require.Empty(t, scheduler.Snapshot("n1"))
}

func TestWatchdog_ExecutionDeadlineAbortsNodeAndRemovesSlot(t *testing.T) {
	aborter := &fakeAborter{}
	w, store, scheduler, _ := watchdog(t, queue.DurableEntry{
		Job: queue.Job{RunID: "deadline", NodeID: "n1", TimeoutSeconds: 10}, State: queue.StateRunning,
		StartedAt: time.Unix(1, 0), Acked: true, AckedAt: time.Unix(1, 0),
	}, aborter)
	require.NoError(t, w.Sweep(context.Background()))
	require.Equal(t, []string{"deadline"}, aborter.abortedRuns())
	require.Empty(t, store.failed)
	require.Empty(t, scheduler.Snapshot("n1"))
}
