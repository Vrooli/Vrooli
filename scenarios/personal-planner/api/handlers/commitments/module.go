package commitments

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/commitments/commitments_v1connect"

	c "personal-planner/internal/commitments"
	"personal-planner/internal/module"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	path, handler := v.NewCommitmentsServiceHandler(NewConnectHandler(Deps{Service: c.NewService(c.NewSQLiteRepository(db, clock)), Logger: logger}))
	return module.Module{Name: "commitments", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return c.Schema() }
