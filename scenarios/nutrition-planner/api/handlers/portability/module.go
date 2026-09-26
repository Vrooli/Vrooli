package portability

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/portability/portability_v1connect"
	"nutrition-planner/internal/module"
	internalPlanning "nutrition-planner/internal/planning"
	internalPortability "nutrition-planner/internal/portability"
	internalRecipe "nutrition-planner/internal/recipe"
	internalShopping "nutrition-planner/internal/shopping"
	internalWorkspace "nutrition-planner/internal/workspace"
)

func Module(recipeService internalRecipe.Service, workspaceService internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, logger *log.Logger) module.Module {
	path, handler := connect.NewPortabilityServiceHandler(NewConnectHandler(recipeService, workspaceService, plans, shoppingRepo, logger))
	return module.Module{Name: "portability", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func ModuleWithRestorer(recipeService internalRecipe.Service, workspaceService internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, restorer internalPortability.Restorer, logger *log.Logger) module.Module {
	path, handler := connect.NewPortabilityServiceHandler(NewConnectHandlerWithRestorer(recipeService, workspaceService, plans, shoppingRepo, restorer, logger))
	return module.Module{Name: "portability", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return internalPortability.Schema() }
