package eligibility

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/eligibility/eligibility_v1connect"

	"nutrition-planner/internal/module"
	workspace "nutrition-planner/internal/workspace"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithWorkspace(workspace.NewService(workspace.NewSQLiteRepository(db, clock)), logger)
}

func ModuleWithWorkspace(workspaces workspace.Service, logger *log.Logger) module.Module {
	path, handler := connect.NewEligibilityServiceHandler(NewConnectHandler(workspaces, logger))
	return module.Module{Name: "eligibility", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return "" }
