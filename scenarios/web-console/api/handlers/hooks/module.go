// Package hooks owns the /api/v1/hooks/* HTTP surface. These are
// inbound webhooks called by the Claude Code CLI on the Stop and
// UserPromptSubmit lifecycle events — a tool we do not control, so the
// request shape is dictated externally and the endpoints stay REST
// under RESTReasonWebhookReceiver (see endpoints.go).
//
// Phase A move: this package owns the route registration and the
// canonical endpoint descriptors. The handler bodies still live in
// api/tts_hook_handler.go and api/hook_prompt_submit_handler.go in
// package main, where they have access to the conversation store,
// session metadata store, and the shared error catalog. Phase B will
// pull the handler bodies into this package once those helpers move
// to a shared httpx package.
package hooks

import (
	"net/http"

	"github.com/gorilla/mux"

	"web-console/internal/module"
)

// Deps is the seam: caller supplies the two handler functions and the
// Module wires them onto the canonical paths.
type Deps struct {
	Stop          http.HandlerFunc
	PromptSubmit  http.HandlerFunc
}

// Module wires both hook handlers into the API router.
func Module(deps Deps) module.Module {
	return module.Module{
		Name: "hooks",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/hooks/stop", deps.Stop).Methods("POST")
			r.HandleFunc("/api/v1/hooks/prompt-submit", deps.PromptSubmit).Methods("POST")
		},
		Endpoints: Endpoints,
	}
}
