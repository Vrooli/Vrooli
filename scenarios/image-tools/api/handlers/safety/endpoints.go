package safety

import (
	"image-tools/internal/module"

	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety/safety_v1connect"
)

// Endpoints describes the safety module's surface. GetPolicy is a generated
// Connect procedure (pure policy discovery); enforcement lives on the AI submit
// edge, so the safety domain exposes no execution endpoint of its own.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "safety_get_policy",
		Path:        safetyconnect.SafetyServiceGetPolicyProcedure,
		Method:      "POST",
		Summary:     "Get the resolved Responsible-Use deployment policy",
		Description: "Returns the resolved Responsible-Use policy for the running deployment tier (local = unrestricted; public = consent-gated + NSFW auto-scan + provenance + rate-limit), plus the per-operation consent-weight table. Pure discovery; enforcement happens server-side on the AI submit edge.",
		Category:    "safety",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"tier":               "DeploymentTier",
				"require_consent":    "bool",
				"force_nsfw_scan":    "bool",
				"require_provenance": "bool",
				"rate_limit_per_min": "int32",
				"op_weights":         "array<OpWeight>",
				"summary":            "string",
			},
		},
		Examples: []module.Example{
			{Name: "Get the active policy", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.safety.SafetyService/GetPolicy -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
