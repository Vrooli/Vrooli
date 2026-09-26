package probes

import (
	"context"
	"log"

	"tunnel-manager/internal/probes"

	"connectrpc.com/connect"

	probesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes"
)

// Deps wires the seams the Connect probes handler needs.
type Deps struct {
	Service probes.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) RunProbes(ctx context.Context, _ *connect.Request[probesv1.RunProbesRequest]) (*connect.Response[probesv1.RunProbesResponse], error) {
	results, err := h.deps.Service.RunProbes(ctx)
	if err != nil {
		h.deps.Logger.Printf("probes.RunProbes: %v", err)
		return nil, probes.ToConnectError(err)
	}
	resp := &probesv1.RunProbesResponse{Results: make([]*probesv1.ProbeResult, 0, len(results))}
	for _, r := range results {
		resp.Results = append(resp.Results, resultToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListProbes(ctx context.Context, req *connect.Request[probesv1.ListProbesRequest]) (*connect.Response[probesv1.ListProbesResponse], error) {
	results, err := h.deps.Service.ListProbes(ctx, req.Msg.Subdomain, int(req.Msg.Limit))
	if err != nil {
		h.deps.Logger.Printf("probes.ListProbes: %v", err)
		return nil, probes.ToConnectError(err)
	}
	resp := &probesv1.ListProbesResponse{Results: make([]*probesv1.ProbeResult, 0, len(results))}
	for _, r := range results {
		resp.Results = append(resp.Results, resultToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) Classify(ctx context.Context, _ *connect.Request[probesv1.ClassifyRequest]) (*connect.Response[probesv1.ClassifyResponse], error) {
	classifications, err := h.deps.Service.Classify(ctx)
	if err != nil {
		h.deps.Logger.Printf("probes.Classify: %v", err)
		return nil, probes.ToConnectError(err)
	}
	resp := &probesv1.ClassifyResponse{Classifications: make([]*probesv1.RouteClassification, 0, len(classifications))}
	for _, c := range classifications {
		resp.Classifications = append(resp.Classifications, classificationToProto(c))
	}
	return connect.NewResponse(resp), nil
}
