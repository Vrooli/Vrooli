package discovery

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery/discovery_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/aisearch"
	skillsdomain "prompt-manager/internal/skills"
)

type connectHandler struct {
	discoveryconnect.UnimplementedDiscoveryServiceHandler
	legacy *domain.Handlers
	skills *skillsdomain.Handlers
}

// NewConnectMount exposes discovery and its telemetry reports as one cohesive
// generated service. The legacy handlers remain in-process domain adapters and
// are not mounted as REST routes.
func NewConnectMount(legacy *domain.Handlers, skills *skillsdomain.Handlers) (string, http.Handler) {
	return discoveryconnect.NewDiscoveryServiceHandler(&connectHandler{legacy: legacy, skills: skills})
}

func (h *connectHandler) Discover(ctx context.Context, req *connect.Request[discoveryv1.DiscoverRequest]) (*connect.Response[discoveryv1.DiscoverResponse], error) {
	return transportbridge.InvokeProto(ctx, req.Header(), h.legacy.Discover, http.MethodPost, "/discover", req.Msg, &discoveryv1.DiscoverResponse{})
}

func (h *connectHandler) ListDiscoveryGaps(ctx context.Context, req *connect.Request[discoveryv1.ListDiscoveryGapsRequest]) (*connect.Response[discoveryv1.ListDiscoveryGapsResponse], error) {
	q := url.Values{"since": {req.Msg.GetSince()}, "type": {req.Msg.GetType()}}
	return transportbridge.InvokeProto(ctx, req.Header(), h.legacy.DiscoveryGaps, http.MethodGet, "/discovery-gaps?"+q.Encode(), nil, &discoveryv1.ListDiscoveryGapsResponse{})
}

func (h *connectHandler) GetDiscoveryMetrics(ctx context.Context, req *connect.Request[discoveryv1.GetDiscoveryMetricsRequest]) (*connect.Response[discoveryv1.GetDiscoveryMetricsResponse], error) {
	q := url.Values{"since": {req.Msg.GetSince()}, "type": {req.Msg.GetType()}}
	return transportbridge.InvokeProto(ctx, req.Header(), h.legacy.DiscoveryMetrics, http.MethodGet, "/discovery-metrics?"+q.Encode(), nil, &discoveryv1.GetDiscoveryMetricsResponse{})
}

func (h *connectHandler) GetSkillUsage(ctx context.Context, req *connect.Request[discoveryv1.GetSkillUsageRequest]) (*connect.Response[discoveryv1.GetSkillUsageResponse], error) {
	q := url.Values{"since": {req.Msg.GetSince()}}
	return transportbridge.InvokeProto(ctx, req.Header(), h.skills.SkillUsage, http.MethodGet, "/skill-usage?"+q.Encode(), nil, &discoveryv1.GetSkillUsageResponse{})
}
