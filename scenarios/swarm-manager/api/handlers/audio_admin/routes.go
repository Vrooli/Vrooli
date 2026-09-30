// Package audio_admin mounts the AudioAdminService Connect-RPC handler.
//
// Wire proto: packages/proto/schemas/swarm-manager/v1/audio_admin/audio_admin.proto.
// Mount path: /vrooli.swarm_manager.v1.audio_admin.AudioAdminService/...
package audio_admin

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	audioadminconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/audio_admin/audio_admin_v1connect"
)

// RegisterRoutes mounts the AudioAdminService handler on the given mux router.
func RegisterRoutes(router *mux.Router, deps Deps) {
	path, handler := audioadminconnect.NewAudioAdminServiceHandler(NewConnectHandler(deps))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
