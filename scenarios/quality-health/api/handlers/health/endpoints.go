package health

import "quality-health/internal/module"

// Endpoints is the machine-readable description of every route this
// module mounts. The codegen at api/cmd/gen-endpoints reads this slice
// (and every other domain's Endpoints slice via the modules registry)
// to emit the canonical .vrooli/endpoints.json. Adding or removing a
// route here without regenerating fails the CI drift check.
//
// The /api/v1/health alias mounted by Module is intentionally not a
// separate descriptor — both paths return the same envelope and serve
// the same purpose; documenting twice would just create a drift
// surface. The description records the alias.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "health",
		Path:        "/health",
		Method:      "GET",
		Summary:     "Service health check",
		Description: "Returns API readiness plus dependency status (also mounted at /api/v1/health for client callers).",
		Category:    "system",
		Response: &module.Schema{
			Type: "Response",
			Properties: map[string]string{
				"status":       "string",
				"readiness":    "boolean",
				"dependencies": "object",
			},
		},
		Examples: []module.Example{
			{Name: "Check health", Curl: "curl http://localhost:${API_PORT}/health"},
		},

		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Plain GET /health for lifecycle systems, load balancers, and curl probes that cannot use a generated Connect client.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{
					Transport:   "none",
					Conformance: "none",
				},
				Response: module.RESTPayload{
					ProtoFullName: "vrooli.quality_health.v1.health.Response",
					Transport:     "json",
					Conformance:   "protojson",
				},
				Error: module.RESTPayload{
					ProtoFullName: "vrooli.quality_health.v1.errors.ErrorEnvelope",
					Transport:     "json",
					Conformance:   "protojson",
				},
			},
		},
	},
}
