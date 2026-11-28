package catalog

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"ai-gateway/internal/capabilitycatalog"
	"ai-gateway/internal/gatewaycatalog"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type connectHandler struct {
	profiles *gatewaycatalog.ProfileCatalog
	roles    rolecatalog.RoleCatalog
	specs    []providers.ProviderSpec
}

func NewConnectHandler(profiles *gatewaycatalog.ProfileCatalog, roles rolecatalog.RoleCatalog, specs []providers.ProviderSpec) *connectHandler {
	return &connectHandler{profiles: profiles, roles: roles, specs: append([]providers.ProviderSpec(nil), specs...)}
}

func (h *connectHandler) GetCapabilityCatalog(_ context.Context, _ *connect.Request[catalogv1.GetCapabilityCatalogRequest]) (*connect.Response[commonv1.AIGatewayCapabilityCatalog], error) {
	if h.profiles == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI Gateway profile catalog is unavailable"))
	}
	return connect.NewResponse(capabilitycatalog.Build(h.profiles, h.roles, h.specs)), nil
}
