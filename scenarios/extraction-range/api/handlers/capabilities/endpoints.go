package capabilities

import "extraction-range/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:            "capabilities_describe",
		Path:          "/api/v1/capabilities/describe",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Plain JSON from the shared capability registry, read by lifecycle and operator tooling.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
		Method:        "GET",
		Summary:       "Describe declared scenario dependencies and their current recovery actions.",
		Category:      "capabilities",
	},
}
