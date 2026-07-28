package recall

import (
	"context"
	"log"

	"connectrpc.com/connect"

	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"

	internalrecall "vrooli-memory/internal/recall"
)

type connectHandler struct {
	service *internalrecall.Service
	logger  *log.Logger
}

func NewConnectHandler(s *internalrecall.Service, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{service: s, logger: l}
}

func (h *connectHandler) Recall(ctx context.Context, req *connect.Request[recallv1.RecallRequest]) (*connect.Response[recallv1.RecallResponse], error) {
	hits, err := h.service.Recall(ctx, req.Msg.GetQuery(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&recallv1.RecallResponse{Hits: hitsProto(hits)}), nil
}

func (h *connectHandler) Wake(ctx context.Context, req *connect.Request[recallv1.WakeRequest]) (*connect.Response[recallv1.WakeResponse], error) {
	wake, err := h.service.Wake(ctx, int(req.Msg.GetTokenBudget()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&recallv1.WakeResponse{Hits: hitsProto(wake.Hits), Overflow: wake.Overflow}), nil
}

func (h *connectHandler) Zoom(ctx context.Context, req *connect.Request[recallv1.ZoomRequest]) (*connect.Response[recallv1.ZoomResponse], error) {
	nodes, err := h.service.Zoom(ctx, req.Msg.GetNodeId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	hits := make([]internalrecall.Hit, len(nodes))
	for i, n := range nodes {
		hits[i] = internalrecall.Hit{Node: n}
	}
	return connect.NewResponse(&recallv1.ZoomResponse{Constituents: hitsProto(hits)}), nil
}

func (h *connectHandler) ListSiblingEvents(context.Context, *connect.Request[recallv1.ListSiblingEventsRequest]) (*connect.Response[recallv1.ListSiblingEventsResponse], error) {
	return connect.NewResponse(&recallv1.ListSiblingEventsResponse{}), nil
}

func hitsProto(hits []internalrecall.Hit) []*recallv1.RecallHit {
	out := make([]*recallv1.RecallHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, &recallv1.RecallHit{EntryId: h.Node.EntryID, FacetId: h.Node.FacetID, Text: h.Node.Text, Score: h.Score, Depth: int32(h.Node.Depth), NodeId: h.Node.ID, Summary: h.Node.Summary, Span: int32(h.Node.Span)})
	}
	return out
}
