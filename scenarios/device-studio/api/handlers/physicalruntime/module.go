package physicalruntime

import (
	"device-studio/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	physicalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical/physical_v1connect"
)

func Module(service Service) module.Module {
	path, handler := physicalconnect.NewPhysicalStarterServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "physical", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
