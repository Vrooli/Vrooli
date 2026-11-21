package forecasts

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/forecasts/forecasts_v1connect"

	f "personal-planner/internal/forecasts"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	path, handler := v.NewForecastsServiceHandler(NewConnectHandler(Deps{Service: f.NewService(db, clock), Logger: logger}))
	return module.Module{Name: "forecasts", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return f.Schema() }
