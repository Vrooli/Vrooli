package registry

import (
	"log"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/registry/registry_v1connect"
)

func Module(db *database.RoutedDB, logger *log.Logger) module.Module {
	repo := catalog.NewSQLiteRepository(db)
	path, handler := registryconnect.NewRegistryServiceHandler(NewConnectHandler(Deps{
		Repository: repo,
		Logger:     logger,
	}))
	return module.Module{
		Name: "registry",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return catalog.Schema() }
