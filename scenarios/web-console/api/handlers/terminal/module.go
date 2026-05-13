// Package terminal owns the terminal-session HTTP surface: the
// per-session multipart upload (image injection) and the terminal
// WebSocket bridge.
//
// Both endpoints stay REST: upload because multipart cannot be carried
// over Connect-RPC's wire format; WebSocket because xterm.js requires a
// raw upgrade. Phase A move: this package owns the route registration
// and canonical endpoint descriptors. The handler bodies still live in
// package main (api/upload_handler.go and api/terminal_ws.go) where
// they have access to the SessionManager, EventLogger, Metrics, and
// shared error catalog. Phase B will pull handler bodies in once those
// helpers move to a shared httpx package.
package terminal

import (
	"net/http"

	"github.com/gorilla/mux"

	"web-console/internal/module"
)

// Deps is the seam: caller supplies the two handler functions and the
// Module wires them onto the canonical paths.
type Deps struct {
	Upload http.HandlerFunc
	WS     http.HandlerFunc
}

// Module wires both terminal endpoints into the API router.
func Module(deps Deps) module.Module {
	return module.Module{
		Name: "terminal",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/sessions/{id}/upload", deps.Upload).Methods("POST")
			r.HandleFunc("/api/v1/sessions/{id}/ws", deps.WS).Methods("GET")
		},
		Endpoints: Endpoints,
	}
}
