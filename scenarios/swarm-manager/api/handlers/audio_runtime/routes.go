// Package audio_runtime mounts the AudioRuntimeService Connect-RPC handler.
//
// Wire proto: packages/proto/schemas/swarm-manager/v1/audio_runtime/audio_runtime.proto.
// Mount path: /vrooli.swarm_manager.v1.audio_runtime.AudioRuntimeService/...
package audio_runtime

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	audioruntimeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/audio_runtime/audio_runtime_v1connect"
)

// RegisterRoutes mounts the AudioRuntimeService handler on the given mux router.
func RegisterRoutes(router *mux.Router, deps Deps) {
	path, handler := audioruntimeconnect.NewAudioRuntimeServiceHandler(NewConnectHandler(deps))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
