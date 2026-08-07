package focus

import (
	"context"
	"log"

	internalfocus "meta-optimization-manager/internal/focus"

	"connectrpc.com/connect"

	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus"
)

// Deps wires the seams the Connect focus handler needs.
type Deps struct {
	Service internalfocus.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the FocusService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetFocus(ctx context.Context, req *connect.Request[focusv1.GetFocusRequest]) (*connect.Response[focusv1.GetFocusResponse], error) {
	items, err := h.deps.Service.GetFocus(ctx, int(req.Msg.GetLimit()), projFromProto(req.Msg.GetProjection()))
	if err != nil {
		h.deps.Logger.Printf("focus.GetFocus: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &focusv1.GetFocusResponse{Items: make([]*focusv1.FocusItem, 0, len(items))}
	for _, it := range items {
		resp.Items = append(resp.Items, &focusv1.FocusItem{
			Gap:           gapToProto(it.Gap),
			Impact:        it.Impact,
			Importance:    it.Importance,
			PriorityScore: it.Priority,
			Rationale:     it.Rationale,
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListGaps(ctx context.Context, req *connect.Request[focusv1.ListGapsRequest]) (*connect.Response[focusv1.ListGapsResponse], error) {
	gaps, err := h.deps.Service.ListGaps(ctx, internalfocus.GapFilter{
		Projection: projFromProto(req.Msg.GetProjection()),
		CellID:     req.Msg.GetCellId(),
		Status:     statusFromProto(req.Msg.GetStatus()),
	})
	if err != nil {
		h.deps.Logger.Printf("focus.ListGaps: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &focusv1.ListGapsResponse{Gaps: make([]*focusv1.Gap, 0, len(gaps))}
	for _, g := range gaps {
		resp.Gaps = append(resp.Gaps, gapToProto(g))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetGap(ctx context.Context, req *connect.Request[focusv1.GetGapRequest]) (*connect.Response[focusv1.GetGapResponse], error) {
	gap, err := h.deps.Service.GetGap(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&focusv1.GetGapResponse{Gap: gapToProto(gap)}), nil
}

func (h *connectHandler) AddGapNote(ctx context.Context, req *connect.Request[focusv1.AddGapNoteRequest]) (*connect.Response[focusv1.AddGapNoteResponse], error) {
	gap, err := h.deps.Service.AddGapNote(ctx, req.Msg.GetId(), req.Msg.GetApproach())
	if err != nil {
		h.deps.Logger.Printf("focus.AddGapNote: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&focusv1.AddGapNoteResponse{Gap: gapToProto(gap)}), nil
}

// gapToProto translates a domain Gap to its proto wire form.
func gapToProto(g internalfocus.Gap) *focusv1.Gap {
	return &focusv1.Gap{
		Id:                 g.ID,
		Projection:         projToProto(g.Projection),
		Title:              g.Title,
		Status:             statusToProto(g.Status),
		SourceCellId:       g.SourceCellID,
		Global:             g.Global,
		Notes:              g.Notes,
		Approaches:         g.Approaches,
		FollowUps:          g.FollowUps,
		Axis:               axisToProto(g.Axis),
		Recurrence:         int32(g.Recurrence),
		EvidenceSource:     g.EvidenceSource,
		EvidenceLocator:    g.EvidenceLocator,
		AvailabilityReason: g.AvailabilityReason,
	}
}
