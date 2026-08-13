package coverage

import (
	"context"
	"log"

	internalcoverage "meta-optimization-manager/internal/coverage"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/coverage"
)

// Deps wires the seams the Connect coverage handler needs.
type Deps struct {
	Service internalcoverage.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the CoverageService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetStatus(ctx context.Context, req *connect.Request[coveragev1.GetStatusRequest]) (*connect.Response[coveragev1.GetStatusResponse], error) {
	status, err := h.deps.Service.GetStatus(ctx, projFromProto(req.Msg.GetProjection()))
	if err != nil {
		h.deps.Logger.Printf("coverage.GetStatus: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &coveragev1.GetStatusResponse{
		Projections:         make([]*coveragev1.ProjectionCoverage, 0, len(status.Projections)),
		ComputedAt:          timestamppb.New(status.ComputedAt),
		DeterminismChecked:  status.DeterminismChecked,
		Deterministic:       status.Deterministic,
		DeterminismEvidence: status.DeterminismEvidence,
	}
	for _, pc := range status.Projections {
		resp.Projections = append(resp.Projections, &coveragev1.ProjectionCoverage{
			Projection:            projToProto(pc.Projection),
			NowCount:              int32(pc.NowCount),
			InReachCount:          int32(pc.InReachCount),
			MissingCount:          int32(pc.MissingCount),
			TotalCells:            int32(pc.TotalCells),
			CoverageRatio:         pc.CoverageRatio,
			DenominatorConfidence: confToProto(pc.DenominatorConfidence),
			ConfidenceRationale:   pc.ConfidenceRationale,
			Available:             pc.Available,
			UnavailableReason:     pc.UnavailableReason,
		})
		for _, condition := range pc.ConditionCounts {
			resp.Projections[len(resp.Projections)-1].ConditionCounts = append(resp.Projections[len(resp.Projections)-1].ConditionCounts, &coveragev1.ConditionCount{Condition: string(condition.Condition), Count: int32(condition.Count)})
		}
	}
	if t := status.LatestTrialTrend; t != nil {
		resp.LatestTrialTrend = &coveragev1.EmpiricalTrendPoint{
			SuccessRate:      t.SuccessRate,
			MedianTokens:     t.MedianTokens,
			MedianDurationMs: t.MedianDurationMs,
			At:               timestamppb.New(t.At),
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListCells(ctx context.Context, req *connect.Request[coveragev1.ListCellsRequest]) (*connect.Response[coveragev1.ListCellsResponse], error) {
	cells, err := h.deps.Service.ListCells(ctx, projFromProto(req.Msg.GetProjection()), statusFromProto(req.Msg.GetStatus()))
	if err != nil {
		h.deps.Logger.Printf("coverage.ListCells: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &coveragev1.ListCellsResponse{Cells: make([]*coveragev1.Cell, 0, len(cells))}
	for _, c := range cells {
		resp.Cells = append(resp.Cells, cellToProto(c))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ExplainCell(ctx context.Context, req *connect.Request[coveragev1.ExplainCellRequest]) (*connect.Response[coveragev1.ExplainCellResponse], error) {
	cell, err := h.deps.Service.ExplainCell(ctx, req.Msg.GetCellId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&coveragev1.ExplainCellResponse{Cell: cellToProto(cell)}), nil
}

func (h *connectHandler) ValidateBaseDocs(ctx context.Context, req *connect.Request[coveragev1.ValidateBaseDocsRequest]) (*connect.Response[coveragev1.ValidateBaseDocsResponse], error) {
	report, err := h.deps.Service.ValidateBaseDocs(ctx, projFromProto(req.Msg.GetProjection()))
	if err != nil {
		h.deps.Logger.Printf("coverage.ValidateBaseDocs: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &coveragev1.ValidateBaseDocsResponse{
		Ok:     report.OK,
		Issues: make([]*coveragev1.BaseDocIssue, 0, len(report.Issues)),
	}
	for _, is := range report.Issues {
		resp.Issues = append(resp.Issues, &coveragev1.BaseDocIssue{
			Projection: projToProto(is.Projection),
			Code:       is.Code,
			Message:    is.Message,
			Location:   is.Location,
			Severity:   severityToProto(is.Severity),
		})
	}
	return connect.NewResponse(resp), nil
}

// cellToProto translates a domain Cell to its proto wire form.
func cellToProto(c internalcoverage.Cell) *coveragev1.Cell {
	out := &coveragev1.Cell{
		Id:          c.ID,
		Projection:  projToProto(c.Projection),
		Question:    c.Question,
		Owner:       c.Owner,
		Status:      statusToProto(c.Status),
		Condition:   string(c.Condition),
		Basis:       basisToProto(c.Basis),
		Sufficiency: sufficiencyToProto(c.Sufficiency),
		Notes:       c.Notes,
		Citations:   make([]*coveragev1.Citation, 0, len(c.Citations)),
	}
	for _, ci := range c.Citations {
		out.Citations = append(out.Citations, &coveragev1.Citation{
			Locator: ci.Locator,
			Kind:    ci.Kind,
			Note:    ci.Note,
		})
	}
	return out
}
