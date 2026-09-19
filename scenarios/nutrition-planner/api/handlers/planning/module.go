package planning

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning/planning_v1connect"

	feedback "nutrition-planner/internal/feedback"
	"nutrition-planner/internal/module"
	internal "nutrition-planner/internal/planning"
	profile "nutrition-planner/internal/profile"
	recipe "nutrition-planner/internal/recipe"
	shopping "nutrition-planner/internal/shopping"
	workspace "nutrition-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithServices(workspace.NewService(workspace.NewSQLiteRepository(db, clock)), recipe.NewService(recipe.NewSQLiteRepository(db, clock)), profile.NewService(profile.NewSQLiteRepository(db, clock)), internal.NewSQLiteRepository(db, clock), shopping.NewSQLiteRepository(db, clock), feedback.NewSQLiteRepository(db, clock), logger)
}

func ModuleWithServices(workspaces workspace.Service, recipes recipe.Service, profiles profile.Service, plans internal.Repository, shoppingRepo shopping.Repository, feedbackRepo feedback.Repository, logger *log.Logger) module.Module {
	path, handler := connect.NewPlanningServiceHandler(NewConnectHandler(workspaces, recipes, profiles, plans, shoppingRepo, feedbackRepo, logger))
	return module.Module{Name: "planning", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() + "\n" + shopping.Schema() + "\n" + feedback.Schema() }
