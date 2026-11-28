package capabilities

import "switchboard/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:       "capabilities_describe",
		Path:     "/api/v1/capabilities/describe",
		Method:   "GET",
		Summary:  "Describe declared scenario dependencies and their current recovery actions.",
		Category: "capabilities",
		RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Machine-readable capability status endpoint.", ProtoPayloads: &module.RESTProtoPayloads{
			Request: module.RESTPayload{Transport: "none", Conformance: "none"}, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
		}},
	},
}
