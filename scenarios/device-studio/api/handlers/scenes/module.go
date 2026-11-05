package scenes

import (
	"device-studio/internal/module"
	"device-studio/internal/photographic"
	"device-studio/internal/scenes"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes/scenes_v1connect"
)

func ModuleWithService(service *scenes.RevisionService, devices ...DeviceRevisionResolver) module.Module {
	path, handler := connect.NewScenesServiceHandler(NewConnectHandler(service, devices...))
	return module.Module{Name: "scenes", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func ModuleWithPreparedPhotographs(service *scenes.RevisionService, source PreparedPhotographSource, library photographic.Publisher, devices ...DeviceRevisionResolver) module.Module {
	path, handler := connect.NewScenesServiceHandler(NewConnectHandlerWithPreparedPhotographs(service, source, library, devices...))
	return module.Module{Name: "scenes", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return scenes.Schema() }
