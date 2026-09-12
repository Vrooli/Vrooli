package settings

import (
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings/settings_v1connect"

	"web-console/internal/module"
)

// Endpoints is the machine-readable description of the settings module's
// public surface. Connect-RPC method paths reference generated *Procedure
// constants so adding/renaming an RPC in settings.proto breaks this file
// at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "settings_session_defaults_get",
		Path:        settingsconnect.SettingsServiceGetSessionDefaultsProcedure,
		Method:      "POST",
		Summary:     "Get session defaults",
		Description: "Returns the backend and expiration policy applied to newly created sessions when the client doesn't override them.",
		Category:    "settings",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"defaults": "SessionDefaults"},
		},
	},
	{
		ID:          "settings_session_defaults_update",
		Path:        settingsconnect.SettingsServiceUpdateSessionDefaultsProcedure,
		Method:      "POST",
		Summary:     "Update session defaults",
		Description: "Sparse update: only the fields the caller sets are applied. Returns the merged defaults.",
		Category:    "settings",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"default_backend": "string (optional)",
				"default_policy":  "ExpirationPolicy (optional)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"defaults": "SessionDefaults"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown backend or malformed expiration policy"},
		},
	},
}
