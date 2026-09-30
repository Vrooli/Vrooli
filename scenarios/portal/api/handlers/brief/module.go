package brief

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	briefconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/brief/brief_v1connect"
	"portal/internal/brief"
	"portal/internal/module"
)

func Module(service *brief.Service) module.Module {
	path, handler := briefconnect.NewBriefServiceHandler(NewHandler(service))
	return module.Module{Name: "brief", Endpoints: Endpoints, Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }}
}
func Schema() string { return brief.Schema() }
