package channel

import (
	"log"
	"net/http"

	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	presenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
)

// Module returns the channel domain's contribution to the API: the dial-out SSE
// stream (a REST ops-probe edge the node's EventSource holds open) plus the
// PresenceService Connect handler the node heartbeats to. Both feed the shared
// presence hub; the heartbeat handler also persists last-seen via the registry
// seam. channel owns no tables of its own.
func Module(hub *presence.Hub, lastSeen LastSeenRecorder, verifier *nodeauth.Verifier, logger *log.Logger) module.Module {
	sse := newSSEHandler(sseDeps{Hub: hub, Verifier: verifier, Logger: logger})
	path, handler := presenceconnect.NewPresenceServiceHandler(NewHeartbeatHandler(HeartbeatDeps{
		Hub:      hub,
		LastSeen: lastSeen,
		Verifier: verifier,
		Logger:   logger,
	}))
	return module.Module{
		Name: "channel",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
			r.HandleFunc("/api/v1/channel/events", sse.handleEvents).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — channel owns no tables (presence is in-memory; last-seen
// lives on the registry's nodes table).
func Schema() string { return "" }
