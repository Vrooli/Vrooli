// Package catalog publishes this scenario's capability catalog over the shared
// cross-scenario media-operation contract (common.v1). It is the read-only
// bridge a workflow builder uses to discover image steps; it never orchestrates
// or mirrors a peer's catalog (see docs/concepts/COMPOSITION.md).
package catalog

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"image-tools/internal/capability"
	"image-tools/internal/module"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/catalog/catalog_v1connect"
)

// Module returns the catalog domain's contribution.
func Module() module.Module {
	path, handler := catalogconnect.NewCatalogServiceHandler(&Handler{})
	return module.Module{
		Name:      "catalog",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

// Handler serves the published catalog.
type Handler struct{}

// GetOperationCatalog projects the capability manifest set onto the shared
// contract. An unknown family filter is rejected rather than silently returning
// an empty catalog, which a builder would read as "this scenario offers nothing".
func (h *Handler) GetOperationCatalog(_ context.Context, req *connect.Request[catalogv1.GetOperationCatalogRequest]) (*connect.Response[commonv1.MediaOperationCatalog], error) {
	family := req.Msg.GetFamily()
	if family != "" && !capability.ValidFamily(family) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown family %q; expected generate|edit|enhance|restore|extract|compose", family))
	}
	return connect.NewResponse(&commonv1.MediaOperationCatalog{
		ContractVersion: capability.ContractVersion,
		Scenario:        capability.ScenarioSlug,
		Operations:      capability.Published(family),
	}), nil
}
