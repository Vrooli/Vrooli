package audio_admin

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	audioadminconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_admin/audio_admin_v1connect"

	"web-console/internal/module"
)

// Module wires the audio_admin domain into the API server.
func Module(deps Deps) module.Module {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	connectPath, connectHandler := audioadminconnect.NewAudioAdminServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "audio_admin",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
