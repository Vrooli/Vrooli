package projects

import (
	"device-studio/internal/module"
	domain "device-studio/internal/projects"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
	"log"
)

func Module(db *database.RoutedDB, logger *log.Logger) module.Module {
	return ModuleWithRepository(domain.NewSQLiteRepository(db, nil), logger)
}
func ModuleWithRepository(repo domain.Repository, logger *log.Logger) module.Module {
	path, svc := projectsconnect.NewProjectsServiceHandler(NewConnectHandler(Deps{Service: domain.NewService(repo, nil), Logger: logger}))
	return module.Module{Name: "projects", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc}) }, Endpoints: Endpoints}
}
func Schema() string { return domain.Schema() }
