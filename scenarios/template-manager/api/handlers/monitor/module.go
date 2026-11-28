package monitor

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"
	monitoring "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/monitor"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	monitorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/monitor/monitor_v1connect"
)

func Module(service *monitoring.Service) module.Module {
	path, handler := monitorconnect.NewMonitorServiceHandler(NewConnectHandler(Deps{Service: service}))
	return module.Module{
		Name: "monitor",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
