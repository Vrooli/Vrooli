package x402resource

import "treasury/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "x402_resource_payment_verify", Path: pathVerify, Method: "POST",
		Summary: "Verify and durably bind an x402 payment to an LPBS job without settling it", Category: "x402",
		Request: &module.Schema{Type: "TreasuryX402ResourceAuthorization", Properties: map[string]string{
			"price_id": "string", "job_id": "string", "resource_id": "string", "order_id": "string", "payment_signature": "string",
		}},
		Response: &module.Schema{Type: "TreasuryX402ResourceAdmission", Properties: map[string]string{
			"admission_id": "string", "price_id": "string", "status": "authorized|settling|recovering|unknown|settled|failed|cancelled", "payer": "string", "network": "string", "mode": "production|local-fork", "job_id": "string", "resource_id": "string", "order_id": "string",
		}},
		Errors:        []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "A verified service principal is required"}, {Status: 403, Code: "permission_denied", Description: "The treasury:x402-authorize capability is required"}, {Status: 409, Code: "failed_precondition", Description: "The payment is already bound to a different job/resource/order"}, {Status: 503, Code: "unavailable", Description: "Treasury or the facilitator is unavailable"}},
		RESTException: restException("Private service-to-service verification. Successful verification creates a durable resource-scoped authorization but does not settle payment."),
	},
	{
		ID: "x402_resource_payment_settle", Path: pathSettle, Method: "POST",
		Summary: "Settle one previously verified LPBS x402 authorization idempotently", Category: "x402",
		Request: &module.Schema{Type: "TreasuryX402ResourceSettlement", Properties: map[string]string{
			"job_id": "string", "resource_id": "string", "order_id": "string",
		}},
		Response: &module.Schema{Type: "TreasuryX402ResourceAdmission", Properties: map[string]string{
			"admission_id": "string", "status": "authorized|settling|recovering|unknown|settled|failed|cancelled", "payer": "string", "transaction_id": "string", "network": "string", "mode": "production|local-fork", "job_id": "string", "resource_id": "string", "order_id": "string",
		}},
		Errors:        []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "A verified service principal is required"}, {Status: 403, Code: "permission_denied", Description: "The treasury:x402-settle capability is required"}, {Status: 404, Code: "not_found", Description: "The resource admission was not found"}, {Status: 409, Code: "failed_precondition", Description: "The admission is bound to a different job/resource/order"}, {Status: 202, Code: "unavailable", Description: "Settlement is still in progress or needs same-admission recovery"}},
		RESTException: restException("Private service-to-service settlement. Replays use the same durable admission and protected payment nonce; settled receipts are emitted by Treasury only once."),
	},
	{
		ID: "x402_resource_payment_cancel", Path: pathCancel, Method: "POST",
		Summary: "Cancel an unsettled LPBS x402 authorization when its delivery verifier declines", Category: "x402",
		Request: &module.Schema{Type: "TreasuryX402ResourceCancellation", Properties: map[string]string{
			"job_id": "string", "resource_id": "string", "order_id": "string",
		}},
		Response: &module.Schema{Type: "TreasuryX402ResourceAdmission", Properties: map[string]string{
			"admission_id": "string", "status": "cancelled", "payer": "string", "network": "string", "mode": "production|local-fork", "job_id": "string", "resource_id": "string", "order_id": "string",
		}},
		Errors:        []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "A verified service principal is required"}, {Status: 403, Code: "permission_denied", Description: "The treasury:x402-cancel capability is required"}, {Status: 404, Code: "not_found", Description: "The resource admission was not found"}, {Status: 409, Code: "failed_precondition", Description: "The admission does not match or settlement has already started"}, {Status: 503, Code: "unavailable", Description: "Protected payload cleanup is temporarily unavailable; retry the same cancellation"}},
		RESTException: restException("Private service-to-service release. Cancellation is allowed only for a matching verified admission before settlement begins; it never calls the facilitator."),
	},
}

func restException(note string) *module.RESTException {
	jsonPayload := module.RESTPayload{Transport: "json", Conformance: "external_shape"}
	return &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: note, ProtoPayloads: &module.RESTProtoPayloads{Request: jsonPayload, Response: jsonPayload, Error: jsonPayload}}
}
