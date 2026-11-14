// Package module defines the onboarding API domain composition seam.
package module

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/endpoints"
)

type Module struct {
	Name      string
	Mount     func(*mux.Router)
	Endpoints []EndpointDescriptor
}

// Connect builds the standard module wrapper for a generated Connect handler.
// Domain packages retain ownership of handler construction and endpoint
// descriptors while the router-mounting seam stays canonical.
func Connect(name, path string, handler http.Handler, endpoints []EndpointDescriptor) Module {
	return Module{
		Name: name,
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: endpoints,
	}
}

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

const (
	RESTReasonMultipartUpload = endpoints.RESTReasonMultipartUpload
	RESTReasonWebhookReceiver = endpoints.RESTReasonWebhookReceiver
	RESTReasonThirdPartyShape = endpoints.RESTReasonThirdPartyShape
	RESTReasonOpsProbe        = endpoints.RESTReasonOpsProbe
)
