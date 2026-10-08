// Package module contains the transport-neutral endpoint metadata seam.
package module

import "github.com/vrooli/api-core/endpoints"

type (
	EndpointDescriptor = endpoints.EndpointDescriptor
	RESTException      = endpoints.RESTException
	RESTPayload        = endpoints.RESTPayload
	RESTProtoPayloads  = endpoints.RESTProtoPayloads
	Schema             = endpoints.Schema
)

const (
	RESTReasonMultipartUpload = endpoints.RESTReasonMultipartUpload
	RESTReasonWebhookReceiver = endpoints.RESTReasonWebhookReceiver
	RESTReasonThirdPartyShape = endpoints.RESTReasonThirdPartyShape
	RESTReasonOpsProbe        = endpoints.RESTReasonOpsProbe
)
