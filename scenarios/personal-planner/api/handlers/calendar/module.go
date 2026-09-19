package calendar

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar/calendar_v1connect"

	d "personal-planner/internal/calendar"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	path, handler := c.NewCalendarServiceHandler(NewConnectHandler(Deps{Service: d.NewService(d.NewSQLiteRepository(db, clock)), Logger: logger, Clock: clock}))
	return module.Module{Name: "calendar", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return d.Schema() }
