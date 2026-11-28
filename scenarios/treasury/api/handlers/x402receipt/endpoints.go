package x402receipt

import "treasury/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:       "x402_accepted_receipt_readback",
		Path:     path,
		Method:   "GET",
		Summary:  "Read one accepted Treasury x402 seller receipt for LPBS reconciliation",
		Category: "x402",
		Request: &module.Schema{Type: "AcceptedX402ReceiptQuery", Properties: map[string]string{
			"external_id": "string",
		}},
		Response: &module.Schema{Type: "AcceptedX402Receipt", Properties: map[string]string{
			"external_id": "string", "posting_id": "string", "receipt_id": "string", "accepted_at": "string",
			"amount_minor": "integer", "currency": "string", "basis": "string",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "A Treasury x402 transaction external ID is required"},
			{Status: 401, Code: "unauthenticated", Description: "A verified service principal is required"},
			{Status: 403, Code: "permission_denied", Description: "The treasury:x402-receipt-read capability is required"},
			{Status: 404, Code: "not_found", Description: "No accepted receipt exists for this transaction"},
			{Status: 409, Code: "failed_precondition", Description: "More than one accepted receipt has this transaction identity"},
			{Status: 503, Code: "unavailable", Description: "Receipt readback is temporarily unavailable"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Private service-to-service reconciliation endpoint. A narrowly scoped verified service principal receives only the accepted receipt matching the exact x402 transaction identity.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "query-json", Conformance: "external_shape"},
				Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
	},
}
