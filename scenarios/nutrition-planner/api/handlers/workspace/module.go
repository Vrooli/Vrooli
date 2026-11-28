package workspace

import (
	"log"

	"nutrition-planner/internal/module"
	internal "nutrition-planner/internal/workspace"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace/workspace_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	svc := internal.NewService(internal.NewSQLiteRepository(db, clock))
	return ModuleWithService(svc, logger)
}

func ModuleWithService(svc internal.Service, logger *log.Logger) module.Module {
	path, h := workspaceconnect.NewWorkspaceServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "workspace", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }
