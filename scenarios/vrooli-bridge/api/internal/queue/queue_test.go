package queue_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/internal/queue"
	"vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// fakePusher records the jobs it successfully pushed, in order. Nodes in
// failNodes report non-delivery (0) so the scheduler treats them as unreachable.
type fakePusher struct {
	mu        sync.Mutex
	pushed    []queue.Job
	failNodes map[string]bool
	err       error
}

func (f *fakePusher) Push(_ context.Context, job queue.Job) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return 0, f.err
	}
	if f.failNodes[job.NodeID] {
		return 0, nil
	}
	f.pushed = append(f.pushed, job)
	return 1, nil
}

func (f *fakePusher) pushedRunIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.pushed))
	for _, j := range f.pushed {
		out = append(out, j.RunID)
	}
	return out
}

// fakeAborter records the runs it was asked to abort.
type fakeAborter struct {
	mu      sync.Mutex
	aborted []string
}

func (f *fakeAborter) Abort(_ context.Context, runID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.aborted = append(f.aborted, runID)
	return nil
}

func (f *fakeAborter) abortedRuns() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.aborted...)
}

func newScheduler(t *testing.T, limit int) (*queue.Scheduler, *fakePusher, *fakeAborter) {
	t.Helper()
	pusher := &fakePusher{failNodes: map[string]bool{}}
	aborter := &fakeAborter{}
	clk := mocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	return queue.NewScheduler(pusher, aborter, clk, limit), pusher, aborter
}

func job(runID, nodeID string) queue.Job {
	return queue.Job{RunID: runID, NodeID: nodeID, Scenario: "web-search", Verb: "scenario test", Args: []string{"web-search"}}
}

// [REQ:BRG-P1-004] A second job for a busy node QUEUES rather than running
// concurrently beyond the node's bound (default 1); only the first is pushed.
func TestSubmit_SecondJobForBusyNodeQueues(t *testing.T) {
	s, pusher, _ := newScheduler(t, 1)
	ctx := context.Background()

	out, delivered, err := s.Submit(ctx, job("r1", "n1"))
	require.NoError(t, err)
	require.Equal(t, queue.OutcomePushed, out)
	require.Equal(t, 1, delivered)

	out, delivered, err = s.Submit(ctx, job("r2", "n1"))
	require.NoError(t, err)
	require.Equal(t, queue.OutcomeQueued, out, "the busy node's second job queues")
	require.Equal(t, 1, delivered, "queued is accepted (delivered=1) so dispatch keeps the run")

	require.Equal(t, []string{"r1"}, pusher.pushedRunIDs(), "only the first job is pushed")

	snap := s.Snapshot("n1")
	require.Len(t, snap, 1)
	require.Equal(t, 1, snap[0].Running)
	require.Equal(t, 1, snap[0].Queued)
}

// [REQ:BRG-P1-004] When the running job completes, the next queued job is
// promoted and pushed — fair FIFO order across a sequence.
func TestComplete_PromotesNextInFIFOOrder(t *testing.T) {
	s, pusher, _ := newScheduler(t, 1)
	ctx := context.Background()

	require.NoError(t, submit(s, ctx, "r1", "n1"))
	require.NoError(t, submit(s, ctx, "r2", "n1"))
	require.NoError(t, submit(s, ctx, "r3", "n1"))
	require.Equal(t, []string{"r1"}, pusher.pushedRunIDs())

	s.Complete(ctx, "n1", "r1")
	require.Equal(t, []string{"r1", "r2"}, pusher.pushedRunIDs(), "r2 promoted after r1")

	s.Complete(ctx, "n1", "r2")
	require.Equal(t, []string{"r1", "r2", "r3"}, pusher.pushedRunIDs(), "r3 promoted after r2")

	s.Complete(ctx, "n1", "r3")
	require.Empty(t, s.Snapshot("n1"), "queue drains empty")
}

// [REQ:BRG-P1-004] A larger concurrency bound runs up to N jobs at once; the
// (N+1)th queues.
func TestSubmit_ConcurrencyBoundGreaterThanOne(t *testing.T) {
	s, pusher, _ := newScheduler(t, 2)
	ctx := context.Background()

	require.NoError(t, submit(s, ctx, "r1", "n1"))
	require.NoError(t, submit(s, ctx, "r2", "n1"))
	require.NoError(t, submit(s, ctx, "r3", "n1"))

	require.Equal(t, []string{"r1", "r2"}, pusher.pushedRunIDs(), "two run at the bound")
	snap := s.Snapshot("n1")
	require.Equal(t, 2, snap[0].Running)
	require.Equal(t, 1, snap[0].Queued)
}

// [REQ:BRG-P1-004] Different nodes run concurrently — the bound is PER node.
func TestSubmit_DifferentNodesRunConcurrently(t *testing.T) {
	s, pusher, _ := newScheduler(t, 1)
	ctx := context.Background()

	require.NoError(t, submit(s, ctx, "r1", "n1"))
	require.NoError(t, submit(s, ctx, "r2", "n2"))

	require.ElementsMatch(t, []string{"r1", "r2"}, pusher.pushedRunIDs(), "both nodes' jobs run")
}

// [REQ:BRG-P1-004] A job whose immediate push does not land reports delivered=0
// (so dispatch aborts the run) and frees the slot it optimistically took.
func TestSubmit_ImmediatePushFailureReportsZeroAndFreesSlot(t *testing.T) {
	s, pusher, _ := newScheduler(t, 1)
	pusher.failNodes["n1"] = true
	ctx := context.Background()

	_, delivered, err := s.Submit(ctx, job("r1", "n1"))
	require.NoError(t, err)
	require.Equal(t, 0, delivered, "non-delivery surfaces as 0 so the caller aborts the run")
	require.Empty(t, s.Snapshot("n1"), "the optimistically-taken slot is freed")
}

func submit(s *queue.Scheduler, ctx context.Context, runID, nodeID string) error {
	_, _, err := s.Submit(ctx, job(runID, nodeID))
	return err
}

// findEntry returns the scheduler entry for (nodeID, runID) from the live
// snapshot, failing the test if absent.
func findEntry(t *testing.T, s *queue.Scheduler, nodeID, runID string) queue.Entry {
	t.Helper()
	for _, nq := range s.Snapshot(nodeID) {
		for _, e := range nq.Entries {
			if e.Job.RunID == runID {
				return e
			}
		}
	}
	t.Fatalf("entry %q not found on node %q", runID, nodeID)
	return queue.Entry{}
}
