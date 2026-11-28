// Package module defines the shared API feature contract used by Vrooli
// scenario servers. A feature mounts its routes and declares the endpoint
// descriptors that tooling publishes for it.
package module

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/endpoints"
)

// Module is the contract every API feature exposes to its server.
type Module struct {
	Name      string
	Mount     func(r *mux.Router)
	Endpoints []EndpointDescriptor
}

// Health constructs the shared public probe module. The same handler serves
// the platform probe path and the versioned API path; descriptors remain
// supplied by the owning scenario so payload contracts stay owner-controlled.
func Health(handler http.Handler, descriptors []EndpointDescriptor) Module {
	return Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			r.Handle("/health", handler).Methods(http.MethodGet)
			r.Handle("/api/v1/health", handler).Methods(http.MethodGet)
		},
		Endpoints: descriptors,
	}
}

// These aliases preserve the descriptor construction vocabulary while the
// wire shape remains owned by api-core/endpoints.
type (
	EndpointDescriptor = endpoints.EndpointDescriptor
	RESTReason         = endpoints.RESTReason
	RESTException      = endpoints.RESTException
	RESTProtoPayloads  = endpoints.RESTProtoPayloads
	RESTPayload        = endpoints.RESTPayload
	Schema             = endpoints.Schema
	ErrorDesc          = endpoints.ErrorDesc
	Example            = endpoints.Example
)

// REST-exception reasons are re-exported alongside the descriptor types.
const (
	RESTReasonMultipartUpload = endpoints.RESTReasonMultipartUpload
	RESTReasonWebhookReceiver = endpoints.RESTReasonWebhookReceiver
	RESTReasonThirdPartyShape = endpoints.RESTReasonThirdPartyShape
	RESTReasonOpsProbe        = endpoints.RESTReasonOpsProbe
)
