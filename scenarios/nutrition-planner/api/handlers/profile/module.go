package profile

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/profile/profile_v1connect"
	"nutrition-planner/internal/module"
	internal "nutrition-planner/internal/profile"
	recipe "nutrition-planner/internal/recipe"
	workspace "nutrition-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, l *log.Logger) module.Module {
	return ModuleWithServices(internal.NewService(internal.NewSQLiteRepository(db, clock)), workspace.NewService(workspace.NewSQLiteRepository(db, clock)), recipe.NewService(recipe.NewSQLiteRepository(db, clock)), l)
}

func ModuleWithServices(s internal.Service, w workspace.Service, recipes recipe.Service, l *log.Logger) module.Module {
	p, h := connect.NewProfileServiceHandler(NewConnectHandler(s, w, recipes, l))
	return module.Module{Name: "profile", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: p, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }
