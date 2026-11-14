package health

import "github.com/vrooli/api-core/module"

var Endpoints = []module.EndpointDescriptor{
	module.HealthEndpoint(module.HealthEndpointConfig{
		ResponseProtoFullName:        "vrooli.code_facts.v1.health.Response",
		ErrorProtoFullName:           "vrooli.code_facts.v1.shared.ErrorEnvelope",
		ResponseSchemaType:           "Response",
		AdditionalResponseProperties: map[string]string{"metrics": "object"},
	}),
}
