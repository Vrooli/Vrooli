package motions

import (
	"device-studio/internal/module"
	"device-studio/internal/motions"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions/motions_v1connect"
)

func ModuleWithService(service *motions.RevisionService) module.Module {
	path, handler := connect.NewMotionsServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "motions", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return motions.Schema() }
