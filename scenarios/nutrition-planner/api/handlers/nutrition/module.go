package nutrition

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition/nutrition_v1connect"
	"nutrition-planner/internal/module"
	internal "nutrition-planner/internal/nutrition"
	"nutrition-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithRepositories(internal.NewSQLiteTargetRepository(db, clock), internal.NewSQLiteIntakeRepository(db), workspace.NewService(workspace.NewSQLiteRepository(db, clock)), logger)
}
func ModuleWithRepository(repo internal.TargetRepository, ws workspace.Service, logger *log.Logger) module.Module {
	return ModuleWithRepositories(repo, nil, ws, logger)
}
func ModuleWithRepositories(repo internal.TargetRepository, intakes internal.IntakeRepository, ws workspace.Service, logger *log.Logger) module.Module {
	path, handler := connect.NewNutritionServiceHandler(NewConnectHandler(repo, intakes, ws, logger))
	return module.Module{Name: "nutrition", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }
