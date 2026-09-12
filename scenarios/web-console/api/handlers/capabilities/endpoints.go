package capabilities

import (
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities/capabilities_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the public surface of the capabilities module.
// Connect procedure paths reference generated *Procedure constants so
// renaming an RPC in capabilities.proto breaks this file at compile
// time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "capabilities_get",
		Path:        capabilitiesconnect.CapabilitiesServiceGetProcedure,
		Method:      "POST",
		Summary:     "Get full capabilities snapshot",
		Description: "Returns the full runtime-capability snapshot, including session backends and the active default backend. May trigger expensive verification checks when the server-side cache is stale.",
		Category:    "capabilities",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"capabilities":     "CapabilityState[]",
				"timestamp":        "string",
				"session_backends": "BackendOption[]",
				"default_backend":  "string",
			},
		},
	},
	{
		ID:          "capabilities_liveness",
		Path:        capabilitiesconnect.CapabilitiesServiceLivenessProcedure,
		Method:      "POST",
		Summary:     "Fast capability liveness probe",
		Description: "Returns cached full-check results when fresh, otherwise lightweight health checkers only. Suitable for hot-path callers like voice activation.",
		Category:    "capabilities",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"capabilities": "CapabilityState[]",
				"timestamp":    "string",
			},
		},
	},
	{
		ID:          "capabilities_run_action",
		Path:        capabilitiesconnect.CapabilitiesServiceRunActionProcedure,
		Method:      "POST",
		Summary:     "Run a delegated capability recovery action",
		Description: "Runs an explicit user-initiated lifecycle action for a declared scenario dependency, then returns a refreshed capability snapshot.",
		Category:    "capabilities",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"success":       "boolean",
				"status":        "string",
				"message":       "string",
				"capability_id": "string",
				"action_kind":   "string",
				"capabilities":  "CapabilityState[]",
				"timestamp":     "string",
			},
		},
	},
}
