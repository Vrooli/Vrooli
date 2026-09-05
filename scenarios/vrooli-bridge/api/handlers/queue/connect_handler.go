package queue

import (
	"context"
	"log"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/queue"

	"connectrpc.com/connect"

	queuev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue"
)

// Snapshotter is the narrow read seam the QueueService handler depends on: the
// live scheduler view. The queue.Scheduler satisfies it.
type Snapshotter interface {
	Snapshot(nodeID string) []queue.NodeQueue
}

// Deps wires the seams the Connect queue handler needs. ListQueue is owner-gated.
type Deps struct {
	Scheduler Snapshotter
	Logger    *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the handler, defaulting the logger.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ListQueue returns the live scheduler view (running + queued jobs per node).
// Owner-gated, read-only.
func (h *connectHandler) ListQueue(ctx context.Context, req *connect.Request[queuev1.ListQueueRequest]) (*connect.Response[queuev1.ListQueueResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	snapshot := h.deps.Scheduler.Snapshot(req.Msg.NodeId)
	resp := &queuev1.ListQueueResponse{Nodes: make([]*queuev1.NodeQueue, 0, len(snapshot))}
	for _, nq := range snapshot {
		resp.Nodes = append(resp.Nodes, nodeQueueToProto(nq))
	}
	return connect.NewResponse(resp), nil
}
