package integrations

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	integrationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations/integrations_v1connect"

	"portal/internal/clock"
	"portal/internal/integrations/registry"
	"portal/internal/module"
)

func Module(registryService *registry.Service) module.Module {
	connectPath, connectHandler := integrationsconnect.NewIntegrationsServiceHandler(NewHandler(registryService))
	return module.Module{
		Name: "integrations",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func NewRegistry(db *database.RoutedDB, clk clock.Clock) *registry.Service {
	return registry.NewService(registry.Config{
		Clock: clk,
		Store: registry.NewStore(db),
	})
}

func Schema() string { return registry.Schema() }
