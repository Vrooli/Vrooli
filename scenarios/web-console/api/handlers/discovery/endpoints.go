package discovery

import (
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/discovery/discovery_v1connect"

	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "discovery_get_audio_tools_endpoint",
		Path:        discoveryconnect.DiscoveryServiceGetAudioToolsEndpointProcedure,
		Method:      "POST",
		Summary:     "Resolve audio-tools base URL for the browser",
		Description: "Returns the audio-tools HTTP + WebSocket base URLs as resolved by the server-side api-core/discovery. Browsers consume this at boot to populate window.__AUDIO_TOOLS_URL__ instead of composing the URL themselves.",
		Category:    "discovery",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"available":          "bool",
				"base_url":           "string",
				"ws_base_url":        "string",
				"unavailable_reason": "string",
			},
		},
		CLIMapping: &module.CLIMapping{Command: "web-console discovery audio-tools"},
	},
}
