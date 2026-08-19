package personas

import (
	"persona/internal/module"
	"persona/internal/personas"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	personasconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/personas/personas_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock) module.Module {
	return ModuleWithService(personas.NewService(personas.NewSQLiteRepository(db, clock)))
}

func ModuleWithService(service personas.Service) module.Module {
	path, handler := personasconnect.NewPersonasServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "personas", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return personas.Schema() }
