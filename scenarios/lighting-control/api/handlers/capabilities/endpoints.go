package capabilities

import "lighting-control/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:            "capabilities_describe",
		Path:          "/api/v1/capabilities/describe",
		Method:        "GET",
		Summary:       "Describe declared scenario dependencies and their current recovery actions.",
		Description:   "Read-only operational snapshot consumed by readiness and lifecycle tooling; no scenario business operation is performed.",
		RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Lifecycle/readiness tooling consumes this generated capability snapshot without a Connect client."},
		Category:      "capabilities",
	},
}
