package portability

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/portability/portability_v1connect"
	"nutrition-planner/internal/module"
	internalNutrition "nutrition-planner/internal/nutrition"
	internalPlanning "nutrition-planner/internal/planning"
	internalPortability "nutrition-planner/internal/portability"
	internalProfile "nutrition-planner/internal/profile"
	internalRecipe "nutrition-planner/internal/recipe"
	internalShopping "nutrition-planner/internal/shopping"
	internalSupplement "nutrition-planner/internal/supplement"
	internalWorkspace "nutrition-planner/internal/workspace"
)

func Module(recipeService internalRecipe.Service, workspaceService internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, logger *log.Logger) module.Module {
	path, handler := connect.NewPortabilityServiceHandler(NewConnectHandler(recipeService, workspaceService, plans, shoppingRepo, logger))
	return module.Module{Name: "portability", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func ModuleWithRestorer(recipeService internalRecipe.Service, workspaceService internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, targets internalNutrition.TargetRepository, restorer internalPortability.Restorer, logger *log.Logger) module.Module {
	path, handler := connect.NewPortabilityServiceHandler(NewConnectHandlerWithDomains(recipeService, workspaceService, plans, shoppingRepo, targets, restorer, logger))
	return module.Module{Name: "portability", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func ModuleWithProfileRestorer(recipeService internalRecipe.Service, workspaceService internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, targets internalNutrition.TargetRepository, profiles internalProfile.Service, restorer internalPortability.Restorer, logger *log.Logger) module.Module {
	path, handler := connect.NewPortabilityServiceHandler(NewConnectHandlerWithProfile(recipeService, workspaceService, plans, shoppingRepo, targets, profiles, restorer, logger))
	return module.Module{Name: "portability", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func ModuleWithIntake(recipeService internalRecipe.Service, workspaceService internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, targets internalNutrition.TargetRepository, profiles internalProfile.Service, intakes internalNutrition.IntakeRepository, restorer internalPortability.Restorer, logger *log.Logger) module.Module {
	path, handler := connect.NewPortabilityServiceHandler(NewConnectHandlerWithIntake(recipeService, workspaceService, plans, shoppingRepo, targets, profiles, intakes, restorer, logger))
	return module.Module{Name: "portability", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func ModuleWithSchedules(recipeService internalRecipe.Service, workspaceService internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, targets internalNutrition.TargetRepository, profiles internalProfile.Service, intakes internalNutrition.IntakeRepository, supplements internalSupplement.Repository, restorer internalPortability.Restorer, logger *log.Logger) module.Module {
	path, handler := connect.NewPortabilityServiceHandler(NewConnectHandlerWithSchedules(recipeService, workspaceService, plans, shoppingRepo, targets, profiles, intakes, supplements, restorer, logger))
	return module.Module{Name: "portability", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return internalPortability.Schema() }
