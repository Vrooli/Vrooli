package health

import "github.com/vrooli/api-core/module"

var Endpoints = []module.EndpointDescriptor{
	module.HealthEndpoint(module.HealthEndpointConfig{
		ResponseProtoFullName: "vrooli.architecture_cartographer.v1.health.Response",
		ErrorProtoFullName:    "vrooli.architecture_cartographer.v1.errors.ErrorEnvelope",
	}),
}
