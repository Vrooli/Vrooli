package realtime

import (
	"log"

	"device-sync-hub/internal/module"
	internalrealtime "device-sync-hub/internal/realtime"

	"github.com/gorilla/mux"
)

// Module returns the realtime domain's contribution to the API: the single SSE
// event stream. The hub is constructed once in main.go and shared with the
// transfer domain (which emits item events through it) and the devices pairing
// hook (which emits pairing-request events). realtime owns no tables and no
// Connect service, so it contributes Endpoints only — no Schema, no proto file.
func Module(hub *internalrealtime.Hub, logger *log.Logger) module.Module {
	h := newSSEHandler(SSEDeps{Hub: hub, Logger: logger})
	return module.Module{
		Name: "realtime",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/realtime/events", h.handleEvents).Methods("GET")
		},
		Endpoints: Endpoints,
	}
}
