package visualhealth

import (
	"context"

	"connectrpc.com/connect"

	vhdomain "ui-health/internal/visualhealth"

	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

type connectHandler struct {
	analyzer vhdomain.Analyzer
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
