package devices

import (
	domain "device-studio/internal/devices"
	"device-studio/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices/devices_v1connect"
)

func ModuleWithService(service Service) module.Module {
	path, handler := connect.NewDevicesServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "devices", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return domain.Schema() }
