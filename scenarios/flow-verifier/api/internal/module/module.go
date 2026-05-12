// Package module defines the domain-module seam: each feature in the API
// (notes, health, …) returns a Module from its handlers package, and main.go
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
//   - Name is for diagnostics ("notes", "health", "tasks"). Free-form;
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
// errors) and notes (full set) round-trip the same way.
type EndpointDescriptor struct {
	ID          string      `json:"id"`
	Path        string      `json:"path"`
	Method      string      `json:"method"`
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Request     *Schema     `json:"request,omitempty"`
	Response    *Schema     `json:"response,omitempty"`
	Errors      []ErrorDesc `json:"errors,omitempty"`
	Examples    []Example   `json:"examples,omitempty"`
	CLIMapping  *CLIMapping `json:"cli_mapping,omitempty"`
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
// it. Command uses flow-verifier as the binary name placeholder, e.g.
// "flow-verifier notes list". Args lists positional/flag tokens for
// commands that take parameters.
type CLIMapping struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}
