package recall

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"

	internalfacets "source-ledger/internal/facets"
	internaljournal "source-ledger/internal/journal"
	"source-ledger/internal/policy"
	internalrecall "source-ledger/internal/recall"
)

type connectHandler struct {
	service *internalrecall.Service
	journal *internaljournal.Service
	usage   *internalfacets.Service
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

func (h *connectHandler) SetUsageRecorder(usage *internalfacets.Service) { h.usage = usage }

func (h *connectHandler) Recall(ctx context.Context, req *connect.Request[recallv1.RecallRequest]) (*connect.Response[recallv1.RecallResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	hits, err := h.service.Recall(ctx, req.Msg.GetQuery(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if h.usage != nil {
		ids := make([]string, 0, len(hits))
		seen := make(map[string]struct{}, len(hits))
		for _, hit := range hits {
			if hit.Node.EntryID == "" {
				continue
			}
			if _, ok := seen[hit.Node.EntryID]; ok {
				continue
			}
			seen[hit.Node.EntryID] = struct{}{}
			ids = append(ids, hit.Node.EntryID)
		}
		if err := h.usage.RecordRecall(ctx, ids); err != nil {
			h.logger.Printf("record recall usage: %v", err)
		}
	}
	return connect.NewResponse(&recallv1.RecallResponse{Hits: hitsProto(hits)}), nil
}

func (h *connectHandler) Wake(ctx context.Context, req *connect.Request[recallv1.WakeRequest]) (*connect.Response[recallv1.WakeResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	wake, err := h.service.Wake(ctx, int(req.Msg.GetTokenBudget()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&recallv1.WakeResponse{Hits: hitsProto(wake.Hits), Overflow: wake.Overflow}), nil
}

func (h *connectHandler) Zoom(ctx context.Context, req *connect.Request[recallv1.ZoomRequest]) (*connect.Response[recallv1.ZoomResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
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
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
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
