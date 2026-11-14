package capabilities

import "chain-forensics/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:       "capabilities_describe",
		Path:     "/api/v1/capabilities/describe",
		Method:   "GET",
		Summary:  "Describe declared scenario dependencies and their current recovery actions.",
		Category: "capabilities",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Read by lifecycle and setup tooling before a generated Connect client exists; reports declared-dependency liveness.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.chain_forensics.v1.shared.Response", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.chain_forensics.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}
