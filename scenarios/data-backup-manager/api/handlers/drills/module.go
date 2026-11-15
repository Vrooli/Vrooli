package drills

import (
	"log"

	d "data-backup-manager/internal/drills"
	"data-backup-manager/internal/modulecontract"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	drillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills/drills_v1connect"
)

func Module(svc d.Service, logger *log.Logger) modulecontract.Module {
	path, handler := drillsconnect.NewRecoveryDrillsServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return modulecontract.Module{Name: "drills", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return d.Schema() }
