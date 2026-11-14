package codingjobs

import (
	"client-rehearsal/internal/codingjobs"
	"client-rehearsal/internal/logx"
	"client-rehearsal/internal/module"
	"context"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/codingjobs/codingjobs_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger logx.Logger) module.Module {
	if clock == nil {
		clock = schedule.System()
	}
	if logger == nil {
		panic("codingjobs.Module requires Logger")
	}
	if err := codingjobs.EnsureReplayKeySchema(context.Background(), db.Primary()); err != nil {
		panic("coding-jobs replay-key schema initialization failed: " + err.Error())
	}
	service := codingjobs.NewService(codingjobs.NewSQLiteRepository(db), clock.Now)
	path, handler := connect.NewCodingJobServiceHandler(NewConnectHandler(service, logger))
	return module.Module{Name: "coding-jobs", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return codingjobs.Schema() }
