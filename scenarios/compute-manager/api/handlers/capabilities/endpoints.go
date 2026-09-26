package capabilities

import "compute-manager/internal/module"

// Endpoints describes the routes this module mounts. The codegen at
// api/cmd/gen-endpoints reads this slice to emit .vrooli/endpoints.json.
//
// This route is a plain GET rather than a Connect procedure because it is
// an operational probe: lifecycle tooling and the setup path read it before
// any generated client is available, and it returns declared-dependency
// facts rather than product state. The RESTException below records that
// deliberately, so the transport check treats it as an intentional
// exception rather than a missing Connect contract.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "capabilities_describe",
		Path:        "/api/v1/capabilities/describe",
		Method:      "GET",
		Summary:     "Describe declared scenario dependencies and their current recovery actions.",
		Description: "Returns each dependency this scenario declares in .vrooli/service.json together with a liveness verdict. An available verdict means the dependency scenario is running, never that this scenario can successfully call it.",
		Category:    "capabilities",
		Examples: []module.Example{
			{Name: "Describe capabilities", Curl: "curl http://localhost:${API_PORT}/api/v1/capabilities/describe"},
		},

		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Read by lifecycle and setup tooling before a generated Connect client exists; reports declared-dependency liveness, not product state.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.compute_manager.v1.shared.Response", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.compute_manager.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}
