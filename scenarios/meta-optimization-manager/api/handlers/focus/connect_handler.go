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
	result, err := h.deps.Service.GetFocus(ctx, int(req.Msg.GetLimit()), projFromProto(req.Msg.GetProjection()))
	if err != nil {
		h.deps.Logger.Printf("focus.GetFocus: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &focusv1.GetFocusResponse{
		Items:          make([]*focusv1.FocusItem, 0, len(result.Items)),
		Degraded:       result.Degraded,
		DegradedReason: result.DegradedReason,
	}
	for _, it := range result.Items {
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

func (h *connectHandler) ListCondition(ctx context.Context, _ *connect.Request[focusv1.ListConditionRequest]) (*connect.Response[focusv1.ListConditionResponse], error) {
	report, err := h.deps.Service.ListConditionReport(ctx)
	if err != nil {
		h.deps.Logger.Printf("focus.ListCondition: %v", err)
	}
	resp := &focusv1.ListConditionResponse{Gaps: make([]*focusv1.Gap, 0, len(report.Gaps)), Instrumentation: &focusv1.ConditionInstrumentation{
		Healthy: int32(report.Instrumentation.Healthy), Degraded: int32(report.Instrumentation.Degraded),
		Dormant: int32(report.Instrumentation.Dormant), Uninstrumented: int32(report.Instrumentation.Uninstrumented),
		Unavailable: int32(report.Instrumentation.Unavailable), Instrumented: int32(report.Instrumentation.Instrumented),
		Total: int32(report.Instrumentation.Total), FilteredOut: int32(report.Instrumentation.FilteredOut),
		LedgerExercise: exerciseBasisToProto(report.Instrumentation.LedgerExercise), ReceiptExercise: exerciseBasisToProto(report.Instrumentation.ReceiptExercise),
	}}
	for _, gap := range report.Gaps {
		resp.Gaps = append(resp.Gaps, gapToProto(gap))
	}
	return connect.NewResponse(resp), err
}

func exerciseBasisToProto(basis internalfocus.ExerciseBasisInstrumentation) *focusv1.ExerciseBasisInstrumentation {
	return &focusv1.ExerciseBasisInstrumentation{Basis: basis.Basis, Instrumented: int32(basis.Instrumented), Total: int32(basis.Total), Invocations: basis.Invocations}
}

func (h *connectHandler) ExplainCondition(ctx context.Context, req *connect.Request[focusv1.ExplainConditionRequest]) (*connect.Response[focusv1.ExplainConditionResponse], error) {
	gap, err := h.deps.Service.ExplainCondition(ctx, req.Msg.GetProviderId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&focusv1.ExplainConditionResponse{Gap: gapToProto(gap)}), nil
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
		ProviderIds:        g.ProviderIDs,
		ConditionStatus:    g.ConditionStatus,
		MaturityFindings:   maturityFindingsToProto(g.MaturityFindings),
		CauseKey:           g.CauseKey,
		AffectedCellIds:    g.AffectedCellIDs,
		AffectedCellCount:  int32(g.AffectedCellCount),
	}
}

func maturityFindingsToProto(findings []internalfocus.MaturityFinding) []*focusv1.MaturityFinding {
	out := make([]*focusv1.MaturityFinding, 0, len(findings))
	for _, finding := range findings {
		out = append(out, &focusv1.MaturityFinding{
			Code:          finding.Code,
			Message:       finding.Message,
			Location:      finding.Location,
			Remediation:   finding.Remediation,
			FixClass:      finding.FixClass,
			RepairCommand: finding.RepairCommand,
		})
	}
	return out
}
