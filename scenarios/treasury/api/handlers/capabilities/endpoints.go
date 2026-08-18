package capabilities

import "treasury/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:       "capabilities_describe",
		Path:     "/api/v1/capabilities/describe",
		Method:   "GET",
		Summary:  "Describe declared scenario dependencies and their current recovery actions.",
		Category: "capabilities",
	},
}
