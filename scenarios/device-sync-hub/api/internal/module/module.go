// Package module defines the domain-module seam: each feature in the API
// (devices, transfer, health, …) returns a Module from its handlers package, and main.go
// passes the slice into server.New. The server iterates and calls Mount on
// each — there is no central routes.go, no per-domain field on server.Deps,
// and no manual editing of .vrooli/endpoints.json.
//
// The Module type is data, not behaviour. A domain can have whatever
// internal shape it wants; the server only consumes Name, Mount, and
// Endpoints. See docs/concepts/ARCHITECTURE.md "Domain modules".
package module

import "github.com/gorilla/mux"

// Module is the contract every API feature exposes to the server.
//
//   - Name is for diagnostics ("devices", "transfer", "health"). Free-form;
//     server does not interpret it.
//   - Mount registers the module's routes (and any subrouter middleware)
//     on the production router. Called once during server.New.
//   - Endpoints describes each route in machine-readable form for the
//     codegen pipeline that emits .vrooli/endpoints.json. The slice
//     is also the source the doc/SDK/Postman generators read.
type Module struct {
	Name      string
	Mount     func(r *mux.Router)
	Endpoints []EndpointDescriptor
}

// EndpointDescriptor mirrors one entry under endpoints[] in
// .vrooli/endpoints.json. JSON tags are deliberately matched so a slice
// of these marshals byte-comparable to the hand-authored shape Pass-2
// established. Optional fields use omitempty so health (no request, no
// errors) and transfer (full set) round-trip the same way.
type EndpointDescriptor struct {
	ID            string         `json:"id"`
	Path          string         `json:"path"`
	Method        string         `json:"method"`
	Summary       string         `json:"summary"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	Request       *Schema        `json:"request,omitempty"`
	Response      *Schema        `json:"response,omitempty"`
	Errors        []ErrorDesc    `json:"errors,omitempty"`
	Examples      []Example      `json:"examples,omitempty"`
	CLIMapping    *CLIMapping    `json:"cli_mapping,omitempty"`
	RESTException *RESTException `json:"rest_exception,omitempty"`
}

// RESTReason enumerates the only mechanically-allowed reasons an endpoint
// may use a hand-authored REST path instead of a generated Connect-RPC
// procedure constant. The gen-endpoints validation pass rejects any
// EndpointDescriptor whose Path is not a Connect procedure path unless
// it carries a RESTException with one of these reasons.
//
// Adding a new reason here is a deliberate architectural decision — drift
// between proto-typed payloads and JSON-typed payloads is exactly what
// this enforcement exists to prevent. If you find yourself wanting a new
// reason, first check whether the actual payload can be expressed as a
// proto message; if it can, use Connect.
type RESTReason string

const (
	// RESTReasonMultipartUpload covers endpoints that accept opaque file
	// bytes via multipart/form-data. The proto-typed response payload is
	// still the source of truth for the metadata shape; only the request
	// transport is REST because proto cannot express multipart uploads.
	// The transfer upload endpoint is the canonical worked example.
	RESTReasonMultipartUpload RESTReason = "multipart_upload"

	// RESTReasonWebhookReceiver covers endpoints whose request shape is
	// dictated by a third-party system (Stripe, GitHub, etc.) we do not
	// own and cannot ask to switch to proto.
	RESTReasonWebhookReceiver RESTReason = "webhook_receiver"

	// RESTReasonThirdPartyShape covers endpoints whose request or
	// response shape is dictated by a third-party API contract (OpenAPI
	// passthrough, OAuth callbacks, etc.) we do not own.
	RESTReasonThirdPartyShape RESTReason = "third_party_shape"

	// RESTReasonOpsProbe covers operational and direct-GET endpoints that
	// clients reach without a generated Connect client: lifecycle/health
	// probes, opaque binary downloads streamed with their original filename,
	// and long-lived server-sent-event streams consumed by the browser
	// EventSource API. In every case the proto-typed metadata (transfer.Item,
	// realtime.Event, the health Response) remains the source of truth and is
	// served proto-typed; only the byte/stream/probe transport is REST because
	// no proto message can carry it. This is the fleet's catch-all reason for a
	// hand-authored HTTP route that is not a Connect procedure.
	RESTReasonOpsProbe RESTReason = "ops_probe"
)

// RESTException tags an EndpointDescriptor whose Path is a hand-authored
// REST path rather than a generated Connect procedure constant. The Note
// field surfaces in .vrooli/endpoints.json so consumers can see the
// human-readable justification. ProtoPayloads declares, per payload role,
// which proto message (if any) is the typed source of truth even when the
// transport is REST — proto-health and the endpoints schema both require it.
type RESTException struct {
	Reason        RESTReason    `json:"reason"`
	Note          string        `json:"note,omitempty"`
	ProtoPayloads ProtoPayloads `json:"proto_payloads"`
}

// ProtoPayloads documents the proto-typed shape behind a REST-exception
// endpoint's request, response, and error roles. Each role is always present
// (the schema requires all three); a role with no proto type uses transport
// "none" / conformance "none".
type ProtoPayloads struct {
	Request  RESTPayload `json:"request"`
	Response RESTPayload `json:"response"`
	Error    RESTPayload `json:"error"`
}

// RESTPayload is one payload role of a RESTException. ProtoFullName is the
// fully-qualified proto message name when the role is proto-typed. Transport
// is one of "connect" | "json" | "multipart/form-data" | "none"; Conformance
// is one of "protojson" | "transport_only" | "external_shape" | "none".
type RESTPayload struct {
	ProtoFullName string `json:"proto_full_name,omitempty"`
	Transport     string `json:"transport"`
	Conformance   string `json:"conformance"`
}

// Schema is the permissive shape used by .vrooli/endpoints.json's
// request/response fields. Properties are name → human-readable type
// (e.g. "string", "array<Note>"); this is documentation, not JSON Schema.
// Real wire validation lives in the proto schemas.
type Schema struct {
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ErrorDesc documents one error envelope a handler can emit. Code values
// must match the wire strings of httpx.Code* constants (e.g. "invalid_request",
// "not_found", "internal").
type ErrorDesc struct {
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// Example is a copy-pastable invocation of the endpoint. Curl is
// preferred; the ${API_PORT} placeholder resolves at scenario start.
type Example struct {
	Name string `json:"name"`
	Curl string `json:"curl"`
}

// CLIMapping links the endpoint to the scenario CLI command that mirrors
// it. Command uses device-sync-hub as the binary name placeholder, e.g.
// "device-sync-hub devices list". Args lists positional/flag tokens for
// commands that take parameters.
type CLIMapping struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}
