// Package discovery is the HTTP-handler home for the discovery domain.
// It exposes the generated Connect-RPC DiscoveryService
// (proto schema: packages/proto/schemas/web-console/v1/discovery).
//
// RPCs (mounted at /vrooli.web_console.v1.discovery.DiscoveryService/...):
//
//	GetAudioToolsEndpoint — returns the resolved audio-tools base URL
//	                        and WebSocket base URL so the browser never
//	                        composes its own scenario URLs.
package discovery

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/discovery/discovery_v1connect"

	"web-console/internal/module"
)

// AudioToolsResolver is the seam the Connect handler depends on.
// Implemented in package main by an adapter that wraps the existing
// audiotoolsint resolver + Client; tests pass an in-memory fake.
//
// Resolve returns the base URL (http or https), and an error when
// the URL cannot be determined. The handler maps errors to the
// proto-level "available=false, unavailable_reason=<token>" shape.
type AudioToolsResolver interface {
	Resolve(ctx context.Context) (string, error)
}

// Module wires the discovery domain into the API server.
func Module(resolver AudioToolsResolver, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := discoveryconnect.NewDiscoveryServiceHandler(NewConnectHandler(Deps{
		AudioTools: resolver,
		Logger:     logger,
	}))
	return module.Module{
		Name: "discovery",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
