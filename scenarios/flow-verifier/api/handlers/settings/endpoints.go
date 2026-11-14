package settings

import (
	"flow-verifier/internal/module"

	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings/settings_v1connect"
)

// Endpoints describes the settings Connect-RPC surface for the codegen
// pipeline that emits .vrooli/endpoints.json. The Path fields reference
// the generated *Procedure constants so renaming an RPC in settings.proto
// breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "settings.get",
		Path:        settingsconnect.SettingsServiceGetSettingsProcedure,
		Method:      "POST",
		Summary:     "Get UI/CLI preferences",
		Description: "Returns the local principal's persisted UI/CLI preferences. Hard-coded defaults are returned when no row exists; never NotFound.",
		Category:    "settings",
	},
	{
		ID:          "settings.update",
		Path:        settingsconnect.SettingsServiceUpdateSettingsProcedure,
		Method:      "POST",
		Summary:     "Update UI/CLI preferences",
		Description: "Partial-update of the local principal's preferences. The FieldMask names which fields of the supplied Settings message are merged into the stored row.",
		Category:    "settings",
	},
}
