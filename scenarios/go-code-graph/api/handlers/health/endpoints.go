package health

import "github.com/vrooli/api-core/module"

var Endpoints = []module.EndpointDescriptor{
	module.HealthEndpoint(module.HealthEndpointConfig{
		ResponseProtoFullName: "vrooli.go_code_graph.v1.health.Response",
		ErrorProtoFullName:    "vrooli.go_code_graph.v1.errors.ErrorEnvelope",
	}),
}
