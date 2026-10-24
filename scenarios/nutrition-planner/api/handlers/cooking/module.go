package cooking

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cooking/cooking_v1connect"

	"nutrition-planner/internal/cooking"
	"nutrition-planner/internal/module"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, recipes recipe.Service, workspaces workspace.Service, logger *log.Logger) module.Module {
	path, handler := connect.NewCookingServiceHandler(NewConnectHandler(cooking.NewSQLiteRepository(db), recipes, workspaces, logger))
	return module.Module{Name: "cooking", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
