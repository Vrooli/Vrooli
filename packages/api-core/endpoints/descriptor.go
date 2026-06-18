// Package endpoints defines the canonical, scenario-agnostic types that
// describe one entry under endpoints[] in a scenario's
// .vrooli/endpoints.json. Every proto-first scenario's
// api/internal/module package re-exports these as type aliases, so the
// shape is defined exactly once across the fleet instead of being copied
// (and drifting) per scenario.
//
// These are data types only — no behaviour, no transport. The mux-coupled
// Module seam stays in each scenario's internal/module package (it wires
// handlers to the production router); only the machine-readable descriptor
// shape lives here. The generator that emits endpoints.json lives in the
// sibling endpoints/gen package.
package endpoints

// EndpointDescriptor mirrors one entry under endpoints[] in
// .vrooli/endpoints.json. JSON tags are deliberately matched so a slice
// of these marshals byte-comparable to the hand-authored shape. Optional
// fields use omitempty so health (no request, no errors) and a fully
// described RPC round-trip the same way.
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
	RESTException *RESTException `json:"rest_exception,omitempty"`
}

// RESTReason enumerates the only mechanically-allowed reasons an endpoint
// may use a hand-authored REST path instead of a generated Connect-RPC
// procedure constant. The gen package's transport validation rejects any
// EndpointDescriptor whose Path is not a Connect procedure path unless it
// carries a RESTException with one of these reasons.
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
	RESTReasonMultipartUpload RESTReason = "multipart_upload"

	// RESTReasonWebhookReceiver covers endpoints whose request shape is
	// dictated by a third-party system (Stripe, GitHub, etc.) we do not
	// own and cannot ask to switch to proto.
	RESTReasonWebhookReceiver RESTReason = "webhook_receiver"

	// RESTReasonThirdPartyShape covers endpoints whose request or
	// response shape is dictated by a third-party API contract (OpenAPI
	// passthrough, OAuth callbacks, etc.) we do not own.
	RESTReasonThirdPartyShape RESTReason = "third_party_shape"

	// RESTReasonOpsProbe covers operational endpoints that lifecycle
	// systems, load balancers, and curl probes must reach without a
	// generated client (e.g. plain GET /health, static browser-facing
	// HTML wrappers served to iframes).
	RESTReasonOpsProbe RESTReason = "ops_probe"
)

// RESTException tags an EndpointDescriptor whose Path is a hand-authored
// REST path rather than a generated Connect procedure constant. The Note
// field surfaces in .vrooli/endpoints.json so consumers can see the
// human-readable justification.
//
// ProtoPayloads declares the request/response/error payload intent for the
// REST edge so proto stays the documented source of truth even where the
// transport can't be a Connect call. It is canonically a pointer with
// omitempty: a scenario that does not declare payload intent omits the key
// entirely, and one that does emits the full object.
type RESTException struct {
	Reason        RESTReason         `json:"reason"`
	Note          string             `json:"note,omitempty"`
	ProtoPayloads *RESTProtoPayloads `json:"proto_payloads,omitempty"`
}

// RESTProtoPayloads declares the proto intent for each role of a REST-exception
// endpoint. All three roles are required by the endpoints.json schema; a role
// with no body uses transport/conformance "none".
type RESTProtoPayloads struct {
	Request  RESTPayload `json:"request"`
	Response RESTPayload `json:"response"`
	Error    RESTPayload `json:"error"`
}

// RESTPayload describes one payload role of a REST-exception endpoint.
//
//   - Transport: how the bytes travel — "connect", "json",
//     "multipart/form-data", or "none" (no body for this role).
//   - Conformance: how the body relates to the proto — "protojson" (a
//     proto message encoded as JSON), "transport_only" (proto-typed but the
//     transport is non-proto, e.g. multipart), "external_shape" (a
//     third-party-dictated shape), or "none".
//   - ProtoFullName: the fully-qualified proto message when the role is
//     proto-typed; omitted for "none".
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
// must match the wire strings of httpx.Code* constants (e.g.
// "invalid_request", "not_found", "internal").
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
