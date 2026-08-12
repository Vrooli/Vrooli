package channel

import (
	"vrooli-bridge/internal/module"

	presenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
)

// Endpoints describes the channel module's public surface: the Connect
// PresenceService heartbeat RPC and the SSE dial-out stream (a documented REST
// ops-probe exception — a long-lived text/event-stream consumed by the node's
// EventSource directly, no Connect client).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "presence_report_heartbeat",
		Path:        presenceconnect.PresenceServiceReportHeartbeatProcedure,
		Method:      "POST",
		Summary:     "Report a node heartbeat",
		Description: "Node-facing: records a node's liveness + self-reported readiness in the presence hub and persists last-seen. Returns the protocol compatibility verdict. Called by the node-agent, not an operator.",
		Category:    "channel",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"heartbeat": "Heartbeat (node_id, sequence, health, sent_at)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"compatibility": "CompatibilityStatus"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing node id"},
		},
		Examples: []module.Example{
			{Name: "Heartbeat", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.presence.PresenceService/ReportHeartbeat -H 'Content-Type: application/json' -d '{\"heartbeat\":{\"node_id\":\"abc123\",\"health\":{\"toolchain_present\":true}}}'"},
		},
	},
	{
		ID:          "channel_events",
		Path:        "/api/v1/channel/events",
		Method:      "GET",
		Summary:     "Open the dial-out channel",
		Description: "Node-facing: the node opens this server-sent-events stream and holds it open; it is online for as long as the stream is held (NAT/firewall-proof — the node always initiates, no inbound port). Phase 1 sends keepalive pings; later phases push JobPush and ProvisionCommand frames down this stream. Authenticated in Phase 1 by the ?node= stub credential (X-Bridge-Node header for non-browser clients); Phase 2 replaces it with per-node Ed25519 mutual auth.",
		Category:    "channel",
		Response: &module.Schema{Type: "text/event-stream", Properties: map[string]string{
			"data": "ServerFrame (vrooli.vrooli_bridge.v1.channel.ServerFrame, protojson per SSE message in later phases)",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Missing node id"},
		},
		Examples: []module.Example{
			{Name: "Dial out", Curl: "curl -N 'http://localhost:${API_PORT}/api/v1/channel/events?node=abc123'"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Long-lived SSE stream (text/event-stream) held open by the node-agent's EventSource directly (no Connect client). Server→node push frames are the proto-typed channel.ServerFrame marshaled with protojson.",
			ProtoPayloads: &module.RESTProtoPayloads{
				// GET dial-out: no request body.
				Request: module.RESTPayload{Transport: "none", Conformance: "none"},
				// Each SSE message is a protojson-encoded ServerFrame envelope.
				Response: module.RESTPayload{ProtoFullName: "vrooli.vrooli_bridge.v1.channel.ServerFrame", Transport: "json", Conformance: "protojson"},
				// A mid-stream failure closes the stream rather than delivering a
				// structured error body; the open/handshake error is the HTTP status.
				Error: module.RESTPayload{Transport: "none", Conformance: "none"},
			},
		},
	},
	{
		ID: "channel_session", Path: "/api/v1/channel/session", Method: "GET",
		Summary:     "Open an authenticated interactive session",
		Description: "Opens a binary WebSocket session with explicit owner re-authentication, the vrooli-bridge:session node scope, bounded flow-control window, idle timeout and hard lifetime. SSE job delivery remains available separately.",
		Category:    "channel", Response: &module.Schema{Type: "application/octet-stream", Properties: map[string]string{"frame": "vrooli.vrooli_bridge.v1.session.Frame"}},
		Errors:        []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Owner token and re-authentication required"}, {Status: 403, Code: "permission_denied", Description: "Node lacks vrooli-bridge:session scope"}},
		RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Bidirectional binary WebSocket; each message carries the proto-encoded v1/session Frame contract.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "binary", Conformance: "transport_only"}, Response: module.RESTPayload{Transport: "binary", Conformance: "transport_only"}, Error: module.RESTPayload{Transport: "none", Conformance: "none"}}},
	},
	{
		ID: "channel_session_kill", Path: "/api/v1/channel/session/{id}", Method: "DELETE",
		Summary: "Terminate an interactive session", Description: "Owner-gated kill switch. Closes the live transport and records the terminal reason.", Category: "channel",
		Errors:        []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Owner authentication required"}, {Status: 404, Code: "not_found", Description: "Session not found"}},
		RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Owner-gated operational control that closes the live binary WebSocket session.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "none", Conformance: "none"}, Response: module.RESTPayload{Transport: "none", Conformance: "none"}, Error: module.RESTPayload{Transport: "none", Conformance: "none"}}},
	},
	{
		ID:          "presence_report_delivery_ack",
		Path:        presenceconnect.PresenceServiceReportDeliveryAckProcedure,
		Method:      "POST",
		Summary:     "Report a delivered server frame",
		Description: "Node-facing: confirms a signed server frame reached the node before the represented operation began.",
		Category:    "channel",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"ack": "DeliveryAck (frame_id, run_id or op_id, received_at)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"accepted": "boolean"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing frame id"},
			{Status: 401, Code: "unauthenticated", Description: "Missing or invalid node proof"},
			{Status: 403, Code: "permission_denied", Description: "Frame was not sent to this node"},
		},
	},
	{
		ID: "channel_session_frame", Path: presenceconnect.PresenceServiceReportSessionFrameProcedure, Method: "POST",
		Summary: "Receive an authenticated node session frame", Description: "Node-facing Connect-RPC for PTY output and session close events. Requires the node proof and binds the frame to its Bridge session.", Category: "channel",
	},
}
