package capabilities

import "decentralized-deploy/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:       "capabilities_describe",
		Path:     "/api/v1/capabilities/describe",
		Method:   "GET",
		Summary:  "Describe declared scenario dependencies and their current recovery actions.",
		Category: "capabilities",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Operational capability description probe returns a dynamic dependency and recovery report.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.decentralized_deploy.v1.deployment.CapabilitiesDescription", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.decentralized_deploy.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}
