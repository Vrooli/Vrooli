// Package events is the HTTP-handler home for the events domain. It
// exposes the generated Connect-RPC EventsService (proto schema:
// packages/proto/schemas/web-console/v1/events).
//
// RPCs (mounted at /vrooli.web_console.v1.events.EventsService/...):
//
//	List — most recent events from the in-memory ring buffer, with a
//	       caller-supplied limit.
//
// Real-time streaming (formerly imagined as an SSE endpoint here) is
// deferred to the broader streams-migration phase alongside terminal
// and voice. When it lands, add a Stream RPC to the same service.
package events

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	eventsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events/events_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. Implemented in
// package main by eventsAdapter, which bridges to the EventLogger.
type Service interface {
	Recent(ctx context.Context, limit int) []Event
	Count(ctx context.Context) int
}

// Event is the transport-neutral view of a single lifecycle event.
// Mirrors the proto Event message; timestamp is RFC3339.
type Event struct {
	Type      string
	SessionID string
	Timestamp string
	Details   map[string]string
}

// Module wires the events domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := eventsconnect.NewEventsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "events",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
