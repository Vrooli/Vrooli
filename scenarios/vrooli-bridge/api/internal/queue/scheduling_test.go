package queue_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/queue"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-P1-004] Queued and running jobs are visible per node with accurate
// state, counts, and FIFO positions — the control-plane view the QueueService
// surfaces.
func TestSnapshot_ShowsRunningAndQueuedPerNode(t *testing.T) {
	s, _, _ := newScheduler(t, 1)
	ctx := context.Background()

	require.NoError(t, submit(s, ctx, "r1", "n1")) // running on n1
	require.NoError(t, submit(s, ctx, "r2", "n1")) // queued behind r1
	require.NoError(t, submit(s, ctx, "r3", "n1")) // queued behind r2
	require.NoError(t, submit(s, ctx, "r4", "n2")) // running on n2

	snap := s.Snapshot("")
	require.Len(t, snap, 2, "both active nodes appear")

	byNode := map[string]queue.NodeQueue{}
	for _, nq := range snap {
		byNode[nq.NodeID] = nq
	}

	n1 := byNode["n1"]
	require.Equal(t, 1, n1.ConcurrencyLimit)
	require.Equal(t, 1, n1.Running)
	require.Equal(t, 2, n1.Queued)
	require.Len(t, n1.Entries, 3)
	// Running first, then queued in FIFO order with positions.
	require.Equal(t, "r1", n1.Entries[0].Job.RunID)
	require.Equal(t, queue.StateRunning, n1.Entries[0].State)
	require.Equal(t, -1, n1.Entries[0].Position)
	require.Equal(t, "r2", n1.Entries[1].Job.RunID)
	require.Equal(t, queue.StateQueued, n1.Entries[1].State)
	require.Equal(t, 0, n1.Entries[1].Position)
	require.Equal(t, "r3", n1.Entries[2].Job.RunID)
	require.Equal(t, 1, n1.Entries[2].Position)

	require.Equal(t, 1, byNode["n2"].Running)
	require.Equal(t, 0, byNode["n2"].Queued)
}

// [REQ:BRG-P1-004] A queued job transitions to running (an accurate state
// transition surfaced in the view) when the slot ahead of it frees.
func TestSnapshot_QueuedJobTransitionsToRunning(t *testing.T) {
	s, _, _ := newScheduler(t, 1)
	ctx := context.Background()

	require.NoError(t, submit(s, ctx, "r1", "n1"))
	require.NoError(t, submit(s, ctx, "r2", "n1"))

	// r2 is queued.
	require.Equal(t, queue.StateQueued, findEntry(t, s, "n1", "r2").State)

	// r1 finishes; r2 is promoted to running.
	s.Complete(ctx, "n1", "r1")
	require.Equal(t, queue.StateRunning, findEntry(t, s, "n1", "r2").State)
	require.Equal(t, -1, findEntry(t, s, "n1", "r2").Position)
}

// [REQ:BRG-P1-004] Filtering the view to one node returns only that node.
func TestSnapshot_FiltersByNode(t *testing.T) {
	s, _, _ := newScheduler(t, 1)
	ctx := context.Background()
	require.NoError(t, submit(s, ctx, "r1", "n1"))
	require.NoError(t, submit(s, ctx, "r2", "n2"))

	snap := s.Snapshot("n2")
	require.Len(t, snap, 1)
	require.Equal(t, "n2", snap[0].NodeID)
}

// [REQ:BRG-P1-004] When a node becomes unreachable, promoting its queued jobs
// aborts each undeliverable run and drains the queue rather than wedging it.
func TestComplete_UnreachableNodeAbortsQueuedRunsAndDrains(t *testing.T) {
	s, pusher, aborter := newScheduler(t, 1)
	ctx := context.Background()

	require.NoError(t, submit(s, ctx, "r1", "n1")) // running
	require.NoError(t, submit(s, ctx, "r2", "n1")) // queued
	require.NoError(t, submit(s, ctx, "r3", "n1")) // queued

	// The node drops; subsequent pushes (promotions) do not land.
	pusher.failNodes["n1"] = true

	s.Complete(ctx, "n1", "r1")

	require.ElementsMatch(t, []string{"r2", "r3"}, aborter.abortedRuns(),
		"each undeliverable queued run is aborted")
	require.Empty(t, s.Snapshot("n1"), "the queue drains rather than wedging")
}
