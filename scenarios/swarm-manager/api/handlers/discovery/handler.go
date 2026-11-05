// Package discovery exposes the swarm-manager DiscoveryService over
// Connect-RPC. The single RPC, GetAudioToolsEndpoint, resolves the
// audio-tools base URL server-side so the swarm-manager UI never
// composes scenario URLs on its own.
//
// Proto: packages/proto/schemas/swarm-manager/v1/discovery/discovery.proto
// Mount: /vrooli.swarm_manager.v1.discovery.DiscoveryService/...
package discovery

import (
	"context"
	"log"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/discovery/discovery_v1connect"
)

// AudioToolsResolver is the seam the handler depends on. The Server
// adapter (api/discovery_adapter.go) implements it by wrapping the
// audiotools.URLResolver field. Tests pass an in-memory fake.
type AudioToolsResolver interface {
	Resolve(ctx context.Context) (string, error)
}

// Deps carries the handler's collaborators.
type Deps struct {
	AudioTools AudioToolsResolver
	Logger     *log.Logger
}

// Handler implements vrooli.swarm_manager.v1.discovery.DiscoveryService.
type Handler struct {
	deps Deps
}

// NewHandler constructs a Connect handler with the given deps.
func NewHandler(d Deps) *Handler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &Handler{deps: d}
}

// GetAudioToolsEndpoint returns the resolved audio-tools URLs, or a
// stable unavailable_reason token when the resolver fails. Never
// returns a Connect-level error — unreachable audio-tools is normal
// runtime state, not an exceptional path.
func (h *Handler) GetAudioToolsEndpoint(ctx context.Context, _ *connect.Request[discoveryv1.GetAudioToolsEndpointRequest]) (*connect.Response[discoveryv1.GetAudioToolsEndpointResponse], error) {
	resp := &discoveryv1.GetAudioToolsEndpointResponse{}
	if h.deps.AudioTools == nil {
		resp.Available = false
		resp.UnavailableReason = "resolver_not_configured"
		return connect.NewResponse(resp), nil
	}
	url, err := h.deps.AudioTools.Resolve(ctx)
	if err != nil {
		resp.Available = false
		resp.UnavailableReason = classifyResolveError(err)
		return connect.NewResponse(resp), nil
	}
	base := strings.TrimRight(url, "/")
	resp.Available = true
	resp.BaseUrl = base
	resp.WsBaseUrl = httpToWS(base)
	return connect.NewResponse(resp), nil
}

// RegisterRoutes mounts the Connect handler on the given mux router.
// Caller wires the resolver and (optional) logger at construction time.
func RegisterRoutes(router *mux.Router, resolver AudioToolsResolver, logger *log.Logger) {
	path, handler := discoveryconnect.NewDiscoveryServiceHandler(NewHandler(Deps{
		AudioTools: resolver,
		Logger:     logger,
	}))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

func httpToWS(httpURL string) string {
	switch {
	case strings.HasPrefix(httpURL, "https://"):
		return "wss://" + strings.TrimPrefix(httpURL, "https://")
	case strings.HasPrefix(httpURL, "http://"):
		return "ws://" + strings.TrimPrefix(httpURL, "http://")
	}
	return httpURL
}

// classifyResolveError maps resolver errors to a small set of stable
// operator-readable tokens the UI banner consumes.
func classifyResolveError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not set"), strings.Contains(msg, "no default"):
		return "env_misconfigured"
	case strings.Contains(msg, "not found"), strings.Contains(msg, "not running"):
		return "scenario_not_running"
	default:
		return "discovery_failed"
	}
}
