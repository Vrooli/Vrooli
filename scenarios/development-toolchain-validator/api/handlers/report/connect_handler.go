package report

import (
	"context"
	"log"

	report "development-toolchain-validator/internal/report"

	"connectrpc.com/connect"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"
)

// Deps wires the seams the Connect report handler needs.
type Deps struct {
	Service report.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for ReportService.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetGoldenSummary(ctx context.Context, req *connect.Request[reportv1.GetGoldenSummaryRequest]) (*connect.Response[reportv1.GetGoldenSummaryResponse], error) {
	s, err := h.deps.Service.GetGoldenSummary(ctx, req.Msg.GoldenSlug)
	if err != nil {
		connectErr := report.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("report.GetGoldenSummary(%q): %v", req.Msg.GoldenSlug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&reportv1.GetGoldenSummaryResponse{Summary: summaryToProto(s)}), nil
}

func (h *connectHandler) GetTupleHistory(ctx context.Context, req *connect.Request[reportv1.GetTupleHistoryRequest]) (*connect.Response[reportv1.GetTupleHistoryResponse], error) {
	hist, err := h.deps.Service.GetTupleHistory(ctx,
		tupleKindProtoToDomain(req.Msg.TupleKind),
		req.Msg.SubjectId, req.Msg.GoldenSlug,
		int(req.Msg.PageSize), req.Msg.PageToken)
	if err != nil {
		connectErr := report.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("report.GetTupleHistory: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&reportv1.GetTupleHistoryResponse{History: historyToProto(hist)}), nil
}

func (h *connectHandler) GetCoverage(ctx context.Context, req *connect.Request[reportv1.GetCoverageRequest]) (*connect.Response[reportv1.GetCoverageResponse], error) {
	cov, err := h.deps.Service.GetCoverage(ctx, req.Msg.GoldenSlug)
	if err != nil {
		connectErr := report.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("report.GetCoverage(%q): %v", req.Msg.GoldenSlug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&reportv1.GetCoverageResponse{Coverage: coverageToProto(cov)}), nil
}
