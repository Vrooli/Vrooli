package health

import "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "health", Path: "/health", Method: "GET", Summary: "Service health check",
		Description: "Returns API readiness for infrastructure probes.", Category: "system",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Lifecycle probes, load balancers, and curl must reach this health endpoint without a generated client.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.vrooli_onboarding.v1.shared.HealthResponse", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.vrooli_onboarding.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
	{
		ID: "health-v1", Path: "/api/v1/health", Method: "GET", Summary: "Service health check for clients",
		Description: "The client-facing alias of the operational health probe.", Category: "system",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Lifecycle probes, load balancers, and curl must reach this health endpoint without a generated client.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.vrooli_onboarding.v1.shared.HealthResponse", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.vrooli_onboarding.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}
