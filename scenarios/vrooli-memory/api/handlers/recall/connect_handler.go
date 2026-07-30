package recall

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"

	internaljournal "vrooli-memory/internal/journal"
	internalrecall "vrooli-memory/internal/recall"
)

type connectHandler struct {
	service *internalrecall.Service
	journal *internaljournal.Service
	logger  *log.Logger
}

func NewConnectHandler(s *internalrecall.Service, l *log.Logger, journals ...*internaljournal.Service) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	h := &connectHandler{service: s, logger: l}
	if len(journals) > 0 {
		h.journal = journals[0]
	}
	return h
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

func (h *connectHandler) ListSiblingEvents(ctx context.Context, req *connect.Request[recallv1.ListSiblingEventsRequest]) (*connect.Response[recallv1.ListSiblingEventsResponse], error) {
	if req.Msg.GetEntryId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("entry_id is required"))
	}
	if h.journal == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("journal source unavailable"))
	}
	entry, err := h.journal.Get(ctx, req.Msg.GetEntryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if entry.Correlation.RunID == "" {
		return connect.NewResponse(&recallv1.ListSiblingEventsResponse{}), nil
	}
	entries, err := h.journal.ListByRun(ctx, entry.Correlation.RunID, 500)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &recallv1.ListSiblingEventsResponse{}
	for _, sibling := range entries {
		if sibling.ID != entry.ID {
			response.Entries = append(response.Entries, &recallv1.RecallHit{EntryId: sibling.ID, FacetId: sibling.FacetID, Text: sibling.Body})
		}
	}
	return connect.NewResponse(response), nil
}

func hitsProto(hits []internalrecall.Hit) []*recallv1.RecallHit {
	out := make([]*recallv1.RecallHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, &recallv1.RecallHit{EntryId: h.Node.EntryID, FacetId: h.Node.FacetID, Text: h.Node.Text, Score: h.Score, Depth: int32(h.Node.Depth), NodeId: h.Node.ID, Summary: h.Node.Summary, Span: int32(h.Node.Span)})
	}
	return out
}
