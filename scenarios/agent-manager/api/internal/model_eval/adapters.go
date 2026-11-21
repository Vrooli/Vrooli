package model_eval

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"
	capacityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity/capacityconnect"
)

// DiscoveredGateway uses the public AI Gateway Connect service. It does not
// call a provider directly and does not persist returned output.
type DiscoveredGateway struct {
	Client interface {
		PreviewRoute(context.Context, *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error)
		ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error)
	}
	HTTPClient connect.HTTPClient
	ResolveURL func(context.Context) (string, error)
}

func (g *DiscoveredGateway) Preview(ctx context.Context, request *sharedv1.GatewayRequest) (*routingv1.PreviewRouteResponse, error) {
	client, err := g.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.PreviewRoute(ctx, connect.NewRequest(&routingv1.PreviewRouteRequest{Request: request}))
	if err != nil || response == nil || response.Msg == nil {
		return nil, errors.New("AI Gateway route preview is unavailable")
	}
	return response.Msg, nil
}

func (g *DiscoveredGateway) Execute(ctx context.Context, request *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
	client, err := g.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.ExecuteRoute(ctx, connect.NewRequest(request))
	if err != nil || response == nil || response.Msg == nil {
		return nil, errors.New("AI Gateway execution returned no metadata receipt")
	}
	return response.Msg, nil
}

func (g *DiscoveredGateway) client(ctx context.Context) (interface {
	PreviewRoute(context.Context, *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error)
	ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error)
}, error) {
	if g != nil && g.Client != nil {
		return g.Client, nil
	}
	resolver := g.ResolveURL
	if resolver == nil {
		resolver = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		}
	}
	baseURL, err := resolver(ctx)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("AI Gateway service discovery failed")
	}
	httpClient := connect.HTTPClient(&http.Client{Timeout: 3 * time.Minute})
	if g.HTTPClient != nil {
		httpClient = g.HTTPClient
	}
	return routingconnect.NewRoutingServiceClient(httpClient, baseURL), nil
}

// SystemMonitorCapacityGate reads the fresh measured host GPU inventory. The
// operation-scoped, no-reclaim claim is still taken by AI Gateway on each call.
type SystemMonitorCapacityGate struct {
	Client     capacityconnect.CapacityServiceClient
	HTTPClient connect.HTTPClient
	ResolveURL func(context.Context) (string, error)
	Now        func() time.Time
}

func (g SystemMonitorCapacityGate) MeasureAndCheck(ctx context.Context, required int64) (CapacityObservation, error) {
	client := g.Client
	if client == nil {
		resolver := g.ResolveURL
		if resolver == nil {
			resolver = func(ctx context.Context) (string, error) {
				return discovery.ResolveScenarioURLDefault(ctx, "system-monitor")
			}
		}
		baseURL, err := resolver(ctx)
		if err != nil || strings.TrimSpace(baseURL) == "" {
			return CapacityObservation{}, errors.New("System Monitor capacity discovery failed")
		}
		httpClient := connect.HTTPClient(&http.Client{Timeout: 20 * time.Second})
		if g.HTTPClient != nil {
			httpClient = g.HTTPClient
		}
		client = capacityconnect.NewCapacityServiceClient(httpClient, baseURL)
	}
	response, err := client.GetCapacityOverview(ctx, connect.NewRequest(&capacitypb.GetCapacityOverviewRequest{}))
	if err != nil || response == nil || response.Msg == nil || !response.Msg.GetSuccess() || !response.Msg.GetSensingAvailable() {
		return CapacityObservation{}, errors.New("live GPU sensing is unavailable")
	}
	var available int64
	for _, gpu := range response.Msg.GetGpus() {
		if gpu.GetFreeBytes() > available {
			available = gpu.GetFreeBytes()
		}
	}
	now := time.Now().UTC()
	if g.Now != nil {
		now = g.Now().UTC()
	}
	return CapacityObservation{ObservedAt: now, Available: available, Fits: available >= required}, nil
}

// FilesystemDiskGate measures the filesystem that holds the LM2 research
// workspace. No prompts, outputs, or per-case artifacts are written there.
type FilesystemDiskGate struct {
	Path string
	Now  func() time.Time
}

func (g FilesystemDiskGate) Measure(context.Context) (DiskObservation, error) {
	if strings.TrimSpace(g.Path) == "" {
		return DiskObservation{}, errors.New("disk measurement path is not configured")
	}
	usage, err := readFilesystemBytes(g.Path)
	if err != nil {
		return DiskObservation{}, errors.New("model-eval filesystem measurement failed")
	}
	now := time.Now().UTC()
	if g.Now != nil {
		now = g.Now().UTC()
	}
	return DiskObservation{ObservedAt: now, FreeBytes: usage, ArtifactBytes: 0}, nil
}
