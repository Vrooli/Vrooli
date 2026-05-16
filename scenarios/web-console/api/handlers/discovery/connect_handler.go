package discovery

import (
	"context"
	"log"
	"strings"

	"connectrpc.com/connect"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/discovery"
)

type Deps struct {
	AudioTools AudioToolsResolver
	Logger     *log.Logger
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

func (h *connectHandler) GetAudioToolsEndpoint(ctx context.Context, _ *connect.Request[discoveryv1.GetAudioToolsEndpointRequest]) (*connect.Response[discoveryv1.GetAudioToolsEndpointResponse], error) {
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
// operator-readable tokens that the browser banner consumes.
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
