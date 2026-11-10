package rehearsals

import (
	"log"

	"client-rehearsal/internal/module"
	internal "client-rehearsal/internal/rehearsals"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals/rehearsals_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	if clock == nil {
		clock = schedule.System()
	}
	service := internal.NewService(internal.NewSQLiteRepository(db), internal.SyntheticEngine{}, clock.Now)
	path, handler := connect.NewRehearsalServiceHandler(NewConnectHandler(service, logger))
	return module.Module{Name: "rehearsals", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }
