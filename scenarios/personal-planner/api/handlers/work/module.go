package work

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	workconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work/work_v1connect"

	"personal-planner/internal/module"
	internalwork "personal-planner/internal/work"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	repo := internalwork.NewSQLiteRepository(db, clock)
	path, handler := workconnect.NewWorkServiceHandler(NewConnectHandler(Deps{Service: internalwork.NewService(repo), Logger: logger}))
	return module.Module{Name: "work", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return internalwork.Schema() }
