package realtime

import "device-sync-hub/internal/module"

// Endpoints describes the realtime module's single public surface: the SSE event
// stream. It is a documented REST exception (ops_probe — a direct browser GET, not
// a Connect procedure) — a long-lived server->client push channel the browser
// EventSource API consumes, which cannot ride Connect framing. Each event's payload
// is the proto-typed realtime.Event marshaled with protojson, so the wire shape
// stays contract-bound even though the streaming envelope is REST.
//
// No CLIMapping: a live event stream has no meaningful CLI verb (the CLI is
// request/response). It carries no proto Connect service, so the global parity
// test does not cover it; it is documented here purely for endpoints.json.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "realtime_events",
		Path:        "/api/v1/realtime/events",
		Method:      "GET",
		Summary:     "Subscribe to realtime events",
		Description: "Opens a server-sent-events stream of presence, item-arrived/deleted, and pairing-request events scoped to the calling trusted device. Authenticated by the device token via the X-Device-Token header or a ?token= query parameter (for the browser EventSource API).",
		Category:    "realtime",
		Response: &module.Schema{Type: "text/event-stream", Properties: map[string]string{
			"data": "Event (vrooli.device_sync_hub.v1.realtime.Event, protojson per SSE message)",
		}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or untrusted device token"},
		},
		Examples: []module.Example{
			{Name: "Subscribe", Curl: "curl -N http://localhost:${API_PORT}/api/v1/realtime/events -H 'X-Device-Token: <token>'"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Long-lived SSE stream (text/event-stream) consumed by the browser EventSource API directly (no Connect client); each event payload is the proto-typed realtime.Event marshaled with protojson.",
			ProtoPayloads: module.ProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.device_sync_hub.v1.realtime.Event", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{Transport: "none", Conformance: "none"},
			},
		},
	},
}
