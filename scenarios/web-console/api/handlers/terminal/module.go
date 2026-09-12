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
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	terminalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal/terminal_v1connect"

	"web-console/internal/module"
)

// LegacyDeps groups the REST-only handler functions (image upload and
// xterm.js WebSocket bridge). They stay REST — multipart can't ride
// Connect, and xterm.js needs a raw WS upgrade.
type LegacyDeps struct {
	Upload http.HandlerFunc
	WS     http.HandlerFunc
}

// Module wires the terminal domain (Connect-RPC TerminalService +
// REST exceptions for upload/WS) into the API router.
func Module(svc Service, legacy LegacyDeps, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := terminalconnect.NewTerminalServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "terminal",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/api/v1/sessions/{id}/upload", legacy.Upload).Methods("POST")
			r.HandleFunc("/api/v1/sessions/{id}/ws", legacy.WS).Methods("GET")
		},
		Endpoints: Endpoints,
	}
}
