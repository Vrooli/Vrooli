package forest

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"

	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest"

	internalforest "vrooli-memory/internal/forest"
)

type connectHandler struct {
	service *internalforest.Service
	logger  *log.Logger
}

func NewConnectHandler(s *internalforest.Service, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{service: s, logger: l}
}

func (h *connectHandler) RunCompactionPass(ctx context.Context, _ *connect.Request[forestv1.RunCompactionPassRequest]) (*connect.Response[forestv1.RunCompactionPassResponse], error) {
	// Compaction is server-owned background work: it can span several local
	// generations, so a client transport disconnect must not interrupt it
	// halfway through a pressure pass. The service serializes concurrent runs.
	passCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Minute)
	defer cancel()
	h.logger.Printf("forest compaction pass started")
	r, e := h.service.Run(passCtx)
	if e != nil {
		h.logger.Printf("forest compaction pass failed: %v", e)
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	h.logger.Printf("forest compaction pass completed: compacted=%d eligible_before=%d eligible_after=%d", r.CompactedCount, r.EligibleFrontierBefore, r.EligibleFrontierAfter)
	return connect.NewResponse(&forestv1.RunCompactionPassResponse{CompactedCount: int32(r.CompactedCount)}), nil
}

func (h *connectHandler) GetFrontier(ctx context.Context, _ *connect.Request[forestv1.GetFrontierRequest]) (*connect.Response[forestv1.GetFrontierResponse], error) {
	nodes, e := h.service.Frontier(ctx)
	if e != nil {
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	return connect.NewResponse(&forestv1.GetFrontierResponse{Nodes: nodesProto(nodes)}), nil
}

func (h *connectHandler) GetNode(ctx context.Context, req *connect.Request[forestv1.GetNodeRequest]) (*connect.Response[forestv1.GetNodeResponse], error) {
	nodes, e := h.service.Frontier(ctx)
	if e != nil {
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	for _, n := range nodes {
		if n.ID == req.Msg.GetId() {
			return connect.NewResponse(&forestv1.GetNodeResponse{Node: nodesProto([]internalforest.Node{n})[0]}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, context.Canceled)
}

func (h *connectHandler) RebuildForest(ctx context.Context, _ *connect.Request[forestv1.RebuildForestRequest]) (*connect.Response[forestv1.RebuildForestResponse], error) {
	passCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Minute)
	defer cancel()
	h.logger.Printf("forest rebuild started")
	r, e := h.service.Rebuild(passCtx)
	if e != nil {
		h.logger.Printf("forest rebuild failed: %v", e)
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	h.logger.Printf("forest rebuild completed: compacted=%d eligible_before=%d eligible_after=%d", r.CompactedCount, r.EligibleFrontierBefore, r.EligibleFrontierAfter)
	return connect.NewResponse(&forestv1.RebuildForestResponse{NodeCount: int32(r.EligibleFrontierAfter)}), nil
}

func nodesProto(nodes []internalforest.Node) []*forestv1.Node {
	out := make([]*forestv1.Node, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, &forestv1.Node{Id: n.ID, EntryId: n.EntryID, FacetId: n.FacetID, Depth: int32(n.Depth)})
	}
	return out
}
