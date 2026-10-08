package integrations

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/integrations/integrations_v1connect"

	d "personal-planner/internal/integrations"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	path, handler := c.NewIntegrationsServiceHandler(NewConnectHandler(Deps{Service: d.NewService(d.NewSQLiteRepository(db, clock)), Logger: logger}))
	return module.Module{Name: "integrations", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return d.Schema() }
