package queue

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/queue"

	"github.com/vrooli/api-core/scheduletest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	queuev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

// okPusher always delivers, so Submit pushes immediately when a slot is free.
type okPusher struct{}

func (okPusher) Push(context.Context, queue.Job) (int, error) { return 1, nil }

type noopAborter struct{}

func (noopAborter) Abort(context.Context, string, string) error { return nil }

func newScheduler(t *testing.T) *queue.Scheduler {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	return queue.NewScheduler(okPusher{}, noopAborter{}, clk, 1)
}

// [REQ:BRG-P1-004] ListQueue is owner-gated: no identity → Unauthenticated.
func TestQueueHandler_ListQueueRequiresOwner(t *testing.T) {
	h := NewConnectHandler(Deps{Scheduler: newScheduler(t)})
	_, err := h.ListQueue(context.Background(), connect.NewRequest(&queuev1.ListQueueRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P1-004] ListQueue surfaces the live per-node view: a busy node shows
// one running + one queued job with accurate states.
func TestQueueHandler_ListQueueShowsRunningAndQueued(t *testing.T) {
	s := newScheduler(t)
	ctx := context.Background()
	_, _, _ = s.Submit(ctx, queue.Job{RunID: "r1", NodeID: "n1", Verb: "scenario test"})
	_, _, _ = s.Submit(ctx, queue.Job{RunID: "r2", NodeID: "n1", Verb: "scenario test"})

	h := NewConnectHandler(Deps{Scheduler: s})
	resp, err := h.ListQueue(ownerCtx(), connect.NewRequest(&queuev1.ListQueueRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Nodes, 1)

	nq := resp.Msg.Nodes[0]
	require.Equal(t, "n1", nq.NodeId)
	require.Equal(t, int32(1), nq.Running)
	require.Equal(t, int32(1), nq.Queued)
	require.Len(t, nq.Entries, 2)
	require.Equal(t, queuev1.QueueState_QUEUE_STATE_RUNNING, nq.Entries[0].State)
	require.Equal(t, queuev1.QueueState_QUEUE_STATE_QUEUED, nq.Entries[1].State)
}
