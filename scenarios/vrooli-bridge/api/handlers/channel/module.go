package channel

import (
	"log"
	"net/http"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/session"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	presenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
)

// Module returns the channel domain's contribution to the API: the dial-out SSE
// stream (a REST ops-probe edge the node's EventSource holds open) plus the
// PresenceService Connect handler the node heartbeats to. Both feed the shared
// presence hub; the heartbeat handler also persists last-seen via the registry
// seam. channel owns no tables of its own.
func Module(hub *presence.Hub, lastSeen LastSeenRecorder, verifier *nodeauth.Verifier, logger *log.Logger, opts ...HeartbeatOption) module.Module {
	sse := newSSEHandler(sseDeps{Hub: hub, Verifier: verifier, Logger: logger})
	heartbeatDeps := HeartbeatDeps{
		Hub:      hub,
		LastSeen: lastSeen,
		Verifier: verifier,
		Logger:   logger,
	}
	for _, opt := range opts {
		opt(&heartbeatDeps)
	}
	path, handler := presenceconnect.NewPresenceServiceHandler(NewHeartbeatHandler(HeartbeatDeps{
		Hub: heartbeatDeps.Hub, LastSeen: heartbeatDeps.LastSeen,
		Verifier: heartbeatDeps.Verifier, Logger: heartbeatDeps.Logger,
		DeliveryAckRecorder: heartbeatDeps.DeliveryAckRecorder, Audit: heartbeatDeps.Audit,
		SessionManager: heartbeatDeps.SessionManager, SessionPush: heartbeatDeps.SessionPush,
		RelayResponses: heartbeatDeps.RelayResponses,
	}))
	return module.Module{
		Name: "channel",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
			r.HandleFunc("/api/v1/channel/events", sse.handleEvents).Methods(http.MethodGet)
			if heartbeatDeps.SessionManager != nil {
				r.HandleFunc("/api/v1/channel/session", (&sessionWSHandler{manager: heartbeatDeps.SessionManager, auth: heartbeatDeps.SessionAuth, registry: heartbeatDeps.SessionRegistry, audit: heartbeatDeps.Audit, push: heartbeatDeps.SessionPush}).handle).Methods(http.MethodGet)
				r.HandleFunc("/api/v1/channel/session/{id}", (&sessionWSHandler{manager: heartbeatDeps.SessionManager, auth: heartbeatDeps.SessionAuth, registry: heartbeatDeps.SessionRegistry, audit: heartbeatDeps.Audit, push: heartbeatDeps.SessionPush}).kill).Methods(http.MethodDelete)
			}
		},
		Endpoints: Endpoints,
	}
}

// WithSessionManager enables the bidirectional WebSocket session edge. The
// manager is optional so the SSE-only channel remains easy to exercise in
// focused tests and during staged rollout.
func WithSessionManager(manager *session.Manager, validator auth.Validator, nodes registry.Service) HeartbeatOption {
	return func(d *HeartbeatDeps) { d.SessionManager, d.SessionAuth, d.SessionRegistry = manager, validator, nodes }
}

// Schema returns "" — channel owns no tables (presence is in-memory; last-seen
// lives on the registry's nodes table).
func Schema() string { return "" }
