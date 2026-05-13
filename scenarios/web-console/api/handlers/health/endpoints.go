// Package health owns the /health endpoint descriptor and the mounting
// glue. The actual handler is built from api-core/health in main.go;
// extracting it cleanly from the Server struct is part of a later phase.
// This package's job in Phase 0 of the Connect-RPC migration is to
// expose the canonical EndpointDescriptor so gen-endpoints can validate
// it against the RESTException rule (health is RESTReasonOpsProbe).
package health

import "web-console/internal/module"

// Endpoints is the machine-readable description of every route this
// module owns. The codegen at api/cmd/gen-endpoints reads this slice
// (and every other domain's Endpoints slice via the modules registry)
// to emit the canonical .vrooli/endpoints.json. Adding or removing a
// route here without regenerating fails the CI drift check.
//
// The /api/v1/health alias is intentionally not a separate descriptor —
// both paths return the same envelope; documenting twice would create
// a drift surface. The description records the alias.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "health",
		Path:        "/health",
		Method:      "GET",
		Summary:     "Service health check",
		Description: "Returns API readiness plus dependency status (also mounted at /api/v1/health for client callers).",
		Category:    "system",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"status":       "string",
				"readiness":    "boolean",
				"dependencies": "object",
			},
		},
		Examples: []module.Example{
			{Name: "Check health", Curl: "curl http://localhost:${API_PORT}/health"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "web-console status",
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Plain GET /health for lifecycle systems, load balancers, and curl probes that cannot use a generated Connect client.",
		},
	},
}
