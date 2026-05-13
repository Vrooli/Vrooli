// Voice streaming WebSocket — the voice domain's one REST-exception
// endpoint. xterm.js-style raw upgrade for full-duplex audio frames
// can't ride Connect-RPC, so /api/v1/voice/stream stays REST until a
// Connect server-streaming RPC replaces it.
//
// Phase A move: this file owns the route registration and canonical
// endpoint descriptor; the handler body still lives in
// api/voice_stream_ws.go (package main) where it has access to the
// capability registry, voice config, speaker-verification helpers, and
// the Whisper transcoder pipeline.
package voice

import (
	"net/http"

	"github.com/gorilla/mux"

	"web-console/internal/module"
)

// StreamEndpoint is the canonical descriptor for the voice WS route.
// Appended to Endpoints at init time so the modules registry picks it
// up from a single source.
var StreamEndpoint = module.EndpointDescriptor{
	ID:          "voice_stream",
	Path:        "/api/v1/voice/stream",
	Method:      "GET",
	Summary:     "Voice streaming WebSocket",
	Description: "Bi-directional WebSocket carrying microphone audio frames upstream and partial/final transcripts plus wake-word and speaker-verification events downstream.",
	Category:    "voice",
	RESTException: &module.RESTException{
		Reason: module.RESTReasonOpsProbe,
		Note:   "Browser microphones stream raw audio over a WebSocket upgrade. Migration to Connect server-streaming is deferred to the final streams phase.",
	},
}

func init() {
	Endpoints = append(Endpoints, StreamEndpoint)
}

// StreamModule wires the voice WS route into the API router. The handler
// is supplied by package main (api/voice_stream_ws.go).
func StreamModule(handler http.HandlerFunc) module.Module {
	return module.Module{
		Name: "voice_stream",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/voice/stream", handler).Methods("GET")
		},
		// StreamEndpoint is already registered in the package-level
		// Endpoints slice via init() so the modules registry picks it
		// up exactly once.
	}
}
