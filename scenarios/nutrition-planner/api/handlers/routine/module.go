package routine

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/routine/routine_v1connect"

	"nutrition-planner/internal/module"
	internal "nutrition-planner/internal/routine"
	"nutrition-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithRepository(internal.NewSQLiteRepository(db, clock), workspace.NewService(workspace.NewSQLiteRepository(db, clock)), logger)
}

func ModuleWithRepository(repo internal.Repository, ws workspace.Service, logger *log.Logger) module.Module {
	path, handler := connect.NewRoutineServiceHandler(NewConnectHandler(repo, ws, logger))
	return module.Module{Name: "routine", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }
