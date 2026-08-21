package worldscale

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	worldscalev1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/worldscale"
	worldscaleconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/worldscale/worldscale_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/worldscale"
)

type connectHandler struct {
	worldscaleconnect.UnimplementedWorldScaleServiceHandler
	get http.HandlerFunc
	put http.HandlerFunc
}

func NewConnectMount(configDir string) (string, http.Handler) {
	return worldscaleconnect.NewWorldScaleServiceHandler(&connectHandler{get: domain.HandleGet(configDir), put: domain.HandlePut(configDir)})
}

func (h *connectHandler) GetWorldScale(ctx context.Context, req *connect.Request[worldscalev1.GetWorldScaleRequest]) (*connect.Response[worldscalev1.WorldScale], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.get, http.MethodGet, "/world-scale", nil, nil, &worldscalev1.WorldScale{})
}

func (h *connectHandler) SetWorldScale(ctx context.Context, req *connect.Request[worldscalev1.SetWorldScaleRequest]) (*connect.Response[worldscalev1.WorldScale], error) {
	body, err := transportbridge.ProtoBody(req.Msg.GetScale())
	if err != nil {
		return nil, err
	}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.put, http.MethodPut, "/world-scale", body, nil, &worldscalev1.WorldScale{})
}
