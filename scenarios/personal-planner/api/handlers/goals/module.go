package goals

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals/goals_v1connect"

	g "personal-planner/internal/goals"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	path, handler := gc.NewGoalsServiceHandler(NewConnectHandler(Deps{Service: g.NewService(g.NewSQLiteRepository(db, clock)), Logger: logger}))
	return module.Module{Name: "goals", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return g.Schema() }
