// Package events owns the descriptor for the server-sent events stream
// endpoint. The handler continues to live in api/events.go; this package
// only exposes the canonical metadata so gen-endpoints can validate the
// route under the RESTException rule. The SSE stream stays REST during
// Phase 2 — migration to Connect server-streaming is deferred to the
// final streams phase alongside terminal/voice WS so the streaming
// pattern is decided once.
package events

import "web-console/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "events_stream",
		Path:        "/api/v1/events",
		Method:      "GET",
		Summary:     "Server-sent events stream",
		Description: "Server-Sent Events stream for live session/conversation updates consumed by the UI via EventSource.",
		Category:    "events",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonThirdPartyShape,
			Note:   "EventSource is a browser primitive with a fixed wire shape. Migration to Connect server-streaming is deferred to the final streams phase.",
		},
	},
}
