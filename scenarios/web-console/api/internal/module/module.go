// Package module defines the domain-module seam: each feature in the API
// (health, sessions, workspace, …) returns a Module from its handlers
// package, and main.go passes the slice into the server wiring. The
// server iterates and calls Mount on each — there is no central
// routes.go, no per-domain field on a shared Deps struct, and no manual
// editing of .vrooli/endpoints.json.
//
// Web Console is mid-migration: most routes still live as
// HandleFunc calls in main.go. The Module shape is being introduced
// domain-by-domain. Health is the first migrated domain; subsequent
// domains follow the same pattern (handlers/<domain>/{module,endpoints}.go,
// registered in internal/modules/registry.go).
//
// The Module type is data, not behaviour. A domain can have whatever
// internal shape it wants; the server only consumes Name, Mount, and
// Endpoints. See docs/concepts/ARCHITECTURE.md for the migration plan.
package module

import "github.com/gorilla/mux"

// Module is the contract every migrated API feature exposes to the
// server.
//
//   - Name is for diagnostics ("health", "sessions", …). Free-form;
//     callers do not interpret it.
//   - Mount registers the module's routes (and any subrouter middleware)
//     on the production router. Called once during server setup.
//   - Endpoints describes each route in machine-readable form for the
//     codegen pipeline that emits .vrooli/endpoints.json. The slice
//     is also the source the doc/SDK/Postman generators read.
type Module struct {
	Name      string
	Mount     func(r *mux.Router)
	Endpoints []EndpointDescriptor
}

// EndpointDescriptor mirrors one entry under endpoints[] in
// .vrooli/endpoints.json. JSON tags match the hand-authored shape so a
// slice of these marshals byte-comparable to the legacy file as routes
// migrate over.
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
	// HTML wrappers served to iframes). Streaming endpoints that
	// terminate at a WebSocket upgrade (xterm.js bridge, voice stream)
	// also live here until a proto server-streaming RPC replaces them.
	RESTReasonOpsProbe RESTReason = "ops_probe"
)

// RESTException tags an EndpointDescriptor whose Path is a hand-authored
// REST path rather than a generated Connect procedure constant. The Note
// field surfaces in .vrooli/endpoints.json so consumers can see the
// human-readable justification.
type RESTException struct {
	Reason RESTReason `json:"reason"`
	Note   string     `json:"note,omitempty"`
}

// Schema is the permissive shape used by .vrooli/endpoints.json's
// request/response fields. Properties are name → human-readable type
// (e.g. "string", "array<Session>"); this is documentation, not JSON
// Schema. Real wire validation lives in the proto schemas.
type Schema struct {
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ErrorDesc documents one error envelope a handler can emit. Code values
// match the wire strings of the canonical error code set.
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
// it. Command uses web-console as the binary name prefix, e.g.
// "web-console status". Args lists positional/flag tokens for commands
// that take parameters.
type CLIMapping struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}
