package tunnel

import (
	"context"
	"log"
	"time"

	"tunnel-manager/internal/tunnel"

	"connectrpc.com/connect"

	tunnelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel"
)

// Deps wires the seams the Connect tunnel handler needs.
type Deps struct {
	Service tunnel.Service
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

func (h *connectHandler) GetStatus(ctx context.Context, _ *connect.Request[tunnelv1.GetStatusRequest]) (*connect.Response[tunnelv1.GetStatusResponse], error) {
	status, latest, err := h.deps.Service.GetStatus(ctx)
	if err != nil {
		h.deps.Logger.Printf("tunnel.GetStatus: %v", err)
		return nil, tunnel.ToConnectError(err)
	}
	return connect.NewResponse(&tunnelv1.GetStatusResponse{
		Status:        statusToProto(status),
		LatestMetrics: sampleToProto(latest),
	}), nil
}

func (h *connectHandler) ListMetrics(ctx context.Context, req *connect.Request[tunnelv1.ListMetricsRequest]) (*connect.Response[tunnelv1.ListMetricsResponse], error) {
	var from, to time.Time
	if req.Msg.From != nil {
		from = req.Msg.From.AsTime()
	}
	if req.Msg.To != nil {
		to = req.Msg.To.AsTime()
	}
	samples, err := h.deps.Service.ListMetrics(ctx, from, to)
	if err != nil {
		h.deps.Logger.Printf("tunnel.ListMetrics: %v", err)
		return nil, tunnel.ToConnectError(err)
	}
	resp := &tunnelv1.ListMetricsResponse{Samples: make([]*tunnelv1.MetricsSample, 0, len(samples))}
	for i := range samples {
		resp.Samples = append(resp.Samples, sampleToProto(&samples[i]))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) Scrape(ctx context.Context, _ *connect.Request[tunnelv1.ScrapeRequest]) (*connect.Response[tunnelv1.ScrapeResponse], error) {
	sample, err := h.deps.Service.Scrape(ctx)
	if err != nil {
		h.deps.Logger.Printf("tunnel.Scrape: %v", err)
		return nil, tunnel.ToConnectError(err)
	}
	return connect.NewResponse(&tunnelv1.ScrapeResponse{Sample: sampleToProto(&sample)}), nil
}
