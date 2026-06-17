// Package module defines the domain-module seam: each feature in the API
// returns a Module from its handlers package, and main.go passes the slice
// into server.New. The server iterates and calls Mount on each — there is no
// central routes.go, no per-domain field on server.Deps, and no manual
// editing of .vrooli/endpoints.json.
//
// The endpoint descriptor types are the fleet-wide canonical shapes defined
// once in github.com/vrooli/api-core/endpoints and re-exported here as type
// aliases, so every construction site (EndpointDescriptor{…}, RESTReasonOpsProbe,
// …) stays unchanged while the shape lives in exactly one place. Only the
// mux-coupled Module seam is local to the scenario.
package module

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/endpoints"
)

// Module is the contract every API feature exposes to the server.
type Module struct {
	Name      string
	Mount     func(r *mux.Router)
	Endpoints []EndpointDescriptor
}

// Canonical endpoint-descriptor types, re-exported from api-core/endpoints.
type (
	EndpointDescriptor = endpoints.EndpointDescriptor
	RESTReason         = endpoints.RESTReason
	RESTException      = endpoints.RESTException
	RESTProtoPayloads  = endpoints.RESTProtoPayloads
	RESTPayload        = endpoints.RESTPayload
	Schema             = endpoints.Schema
	ErrorDesc          = endpoints.ErrorDesc
	Example            = endpoints.Example
	CLIMapping         = endpoints.CLIMapping
)

// Canonical REST-exception reasons, re-exported from api-core/endpoints.
const (
	RESTReasonMultipartUpload = endpoints.RESTReasonMultipartUpload
	RESTReasonWebhookReceiver = endpoints.RESTReasonWebhookReceiver
	RESTReasonThirdPartyShape = endpoints.RESTReasonThirdPartyShape
	RESTReasonOpsProbe        = endpoints.RESTReasonOpsProbe
)
