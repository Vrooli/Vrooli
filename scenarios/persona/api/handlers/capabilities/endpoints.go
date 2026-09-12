package capabilities

import "persona/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:       "capabilities_describe",
		Path:     "/api/v1/capabilities/describe",
		Method:   "GET",
		Summary:  "Describe declared scenario dependencies and their current recovery actions.",
		Category: "capabilities",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Machine-readable capability status endpoint.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{
					ProtoFullName: "vrooli.persona.v1.capabilities.DescribeResponse",
					Transport:     "json",
					Conformance:   "external_shape",
				},
				Error: module.RESTPayload{
					ProtoFullName: "vrooli.persona.v1.shared.ErrorEnvelope",
					Transport:     "json",
					Conformance:   "external_shape",
				},
			},
		},
	},
}
