package settings

import "flow-verifier/internal/module"

// Endpoints describes the settings HTTP surface for the codegen
// pipeline that emits .vrooli/endpoints.json. Adding or removing a
// route here without regenerating fails the CI drift check.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "settings.get",
		Path:        "/api/v1/settings",
		Method:      "GET",
		Summary:     "Get UI/CLI preferences",
		Description: "Returns the local principal's persisted UI/CLI preferences. Hard-coded defaults are returned when no row exists; never 404s.",
		Category:    "settings",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier settings get"},
	},
	{
		ID:          "settings.put",
		Path:        "/api/v1/settings",
		Method:      "PUT",
		Summary:     "Update UI/CLI preferences",
		Description: "Partial-update of the local principal's preferences. Body is merged with the current row; unknown enum values produce 400.",
		Category:    "settings",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier settings set"},
	},
}
