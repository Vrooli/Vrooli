package workspace

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace/workspace_v1connect"

	"personal-planner/internal/module"
	d "personal-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	path, handler := c.NewWorkspaceServiceHandler(NewConnectHandler(Deps{Service: d.NewService(d.NewSQLiteRepository(db, clock)), Logger: logger}))
	return module.Module{Name: "workspace", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return d.Schema() }
