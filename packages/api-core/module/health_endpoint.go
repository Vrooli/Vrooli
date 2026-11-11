package module

import "net/http"

// HealthEndpointConfig describes the owner-specific proto envelope behind the
// shared operational health route. AdditionalResponseProperties extends the
// common health response shape for owners that publish extra fields.
type HealthEndpointConfig struct {
	ResponseProtoFullName        string
	ErrorProtoFullName           string
	ResponseSchemaType           string
	AdditionalResponseProperties map[string]string
}

// HealthEndpoint returns the standard descriptor for a plain GET /health
// probe. The route, response contract and exception policy stay consistent
// while each service supplies its generated proto type names.
func HealthEndpoint(config HealthEndpointConfig) EndpointDescriptor {
	schemaType := config.ResponseSchemaType
	if schemaType == "" {
		schemaType = "object"
	}
	properties := map[string]string{
		"status":       "string",
		"readiness":    "boolean",
		"dependencies": "object",
	}
	for name, kind := range config.AdditionalResponseProperties {
		properties[name] = kind
	}
	return EndpointDescriptor{
		ID:          "health",
		Path:        "/health",
		Method:      http.MethodGet,
		Summary:     "Service health check",
		Description: "Returns API readiness plus dependency status (also mounted at /api/v1/health for client callers).",
		Category:    "system",
		Response:    &Schema{Type: schemaType, Properties: properties},
		Examples: []Example{{
			Name: "Check health",
			Curl: "curl http://localhost:${API_PORT}/health",
		}},
		RESTException: &RESTException{
			Reason: RESTReasonOpsProbe,
			Note:   "Plain GET /health for lifecycle systems, load balancers, and curl probes that cannot use a generated Connect client.",
			ProtoPayloads: &RESTProtoPayloads{
				Request: RESTPayload{Transport: "none", Conformance: "none"},
				Response: RESTPayload{
					ProtoFullName: config.ResponseProtoFullName,
					Transport:     "json",
					Conformance:   "protojson",
				},
				Error: RESTPayload{
					ProtoFullName: config.ErrorProtoFullName,
					Transport:     "json",
					Conformance:   "protojson",
				},
			},
		},
	}
}
