// Package catalog publishes the AI Gateway's read-only capability inventory.
// It is a native-catalog projection for discovery, never a source of routing
// authority or consumer dispatch policy.
package catalog

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"ai-gateway/internal/gatewaycatalog"
	"ai-gateway/internal/module"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog/catalog_v1connect"
)

var ProtoFile = catalogv1.File_ai_gateway_v1_catalog_catalog_proto

// Module returns the catalog domain's Connect contribution.
func Module(profiles *gatewaycatalog.ProfileCatalog, roles rolecatalog.RoleCatalog) module.Module {
	path, handler := catalogconnect.NewCatalogServiceHandler(NewConnectHandler(profiles, roles, providers.DefaultProviderSpecs()))
	return module.Module{
		Name: "catalog",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
