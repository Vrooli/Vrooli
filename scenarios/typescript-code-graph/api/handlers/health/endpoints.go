package health

import "github.com/vrooli/api-core/module"

var Endpoints = []module.EndpointDescriptor{
	module.HealthEndpoint(module.HealthEndpointConfig{
		ResponseProtoFullName: "vrooli.typescript_code_graph.v1.health.Response",
		ErrorProtoFullName:    "vrooli.typescript_code_graph.v1.errors.ErrorEnvelope",
	}),
}
