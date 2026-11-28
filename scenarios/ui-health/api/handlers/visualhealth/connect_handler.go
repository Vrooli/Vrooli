package visualhealth

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	vhdomain "ui-health/internal/visualhealth"

	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

type connectHandler struct {
	analyzer     vhdomain.Analyzer
	observations vhdomain.ObservationRepository
}

func NewConnectHandler(analyzer vhdomain.Analyzer) *connectHandler {
	return &connectHandler{analyzer: analyzer}
}

func (h *connectHandler) AnalyzeArtifacts(ctx context.Context, req *connect.Request[visualpb.AnalyzeArtifactsRequest]) (*connect.Response[visualpb.AnalyzeArtifactsResponse], error) {
	_ = ctx
	return connect.NewResponse(h.analyzer.Analyze(req.Msg)), nil
}

func (h *connectHandler) CompareArtifacts(ctx context.Context, req *connect.Request[visualpb.CompareArtifactsRequest]) (*connect.Response[visualpb.CompareArtifactsResponse], error) {
	_ = ctx
	return connect.NewResponse(h.analyzer.Compare(req.Msg)), nil
}

func (h *connectHandler) ListRules(ctx context.Context, req *connect.Request[visualpb.ListRulesRequest]) (*connect.Response[visualpb.ListRulesResponse], error) {
	_ = ctx
	_ = req
	return connect.NewResponse(&visualpb.ListRulesResponse{Rules: vhdomain.Rules()}), nil
}

func (h *connectHandler) RecordBrowserObservation(ctx context.Context, req *connect.Request[visualpb.RecordBrowserObservationRequest]) (*connect.Response[visualpb.RecordBrowserObservationResponse], error) {
	if h.observations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("browser observation storage unavailable"))
	}
	row, err := h.observations.Record(ctx, req.Msg.GetObservation())
	if err != nil {
		if errors.Is(err, vhdomain.ErrInvalidObservation) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&visualpb.RecordBrowserObservationResponse{Observation: row}), nil
}

func (h *connectHandler) ListBrowserObservations(ctx context.Context, req *connect.Request[visualpb.ListBrowserObservationsRequest]) (*connect.Response[visualpb.ListBrowserObservationsResponse], error) {
	if h.observations == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("browser observation storage unavailable"))
	}
	rows, err := h.observations.List(ctx, req.Msg.GetScenario(), req.Msg.GetClientId(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&visualpb.ListBrowserObservationsResponse{Observations: rows}), nil
}
