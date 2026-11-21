package audio_runtime

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	audioruntimeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime/audio_runtime_v1connect"

	"web-console/internal/module"
)

// Module wires the audio_runtime domain into the API server.
func Module(deps Deps) module.Module {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	connectPath, connectHandler := audioruntimeconnect.NewAudioRuntimeServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "audio_runtime",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
