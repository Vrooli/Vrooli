// Package module contains the transport-neutral endpoint metadata seam.
package module

import "github.com/vrooli/api-core/endpoints"

type EndpointDescriptor = endpoints.EndpointDescriptor
type RESTException = endpoints.RESTException
type RESTPayload = endpoints.RESTPayload
type RESTProtoPayloads = endpoints.RESTProtoPayloads
type Schema = endpoints.Schema

const (
	RESTReasonMultipartUpload = endpoints.RESTReasonMultipartUpload
	RESTReasonWebhookReceiver = endpoints.RESTReasonWebhookReceiver
	RESTReasonThirdPartyShape = endpoints.RESTReasonThirdPartyShape
	RESTReasonOpsProbe        = endpoints.RESTReasonOpsProbe
)
