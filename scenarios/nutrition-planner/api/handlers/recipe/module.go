package recipe

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe/recipe_v1connect"

	"nutrition-planner/internal/module"
	internal "nutrition-planner/internal/recipe"
	workspace "nutrition-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	s := internal.NewService(internal.NewSQLiteRepository(db, clock))
	w := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	return ModuleWithService(s, w, logger)
}

func ModuleWithService(s internal.Service, w workspace.Service, logger *log.Logger) module.Module {
	p, h := connect.NewRecipeServiceHandler(NewConnectHandler(s, w, logger))
	return module.Module{Name: "recipe", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: p, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }
