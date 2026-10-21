package scenes

import (
	"device-studio/internal/module"
	"device-studio/internal/scenes"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes/scenes_v1connect"
)

func ModuleWithService(service *scenes.RevisionService) module.Module {
	path, handler := connect.NewScenesServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "scenes", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return scenes.Schema() }
