package stt

import (
	"net/http"

	intvoice "audio-tools/internal/voice"
)

// StreamWSHandler upgrades to a WebSocket and delegates to the existing
// voice.Service WS handler. Documented in SEAMS.md as
// TransportReason: websocket_transport, not a REST exception.
func StreamWSHandler(voice *intvoice.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if voice == nil {
			http.Error(w, "voice service not configured", http.StatusServiceUnavailable)
			return
		}
		voice.HandleStreamWS(w, r)
	})
}
