package scenarios

import (
	"flow-verifier/internal/module"

	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

// Endpoints describes the scenarios Connect-RPC surface for the codegen
// pipeline that emits .vrooli/endpoints.json.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "scenarios.list",
		Path:        scenariosconnect.ScenariosServiceListScenariosProcedure,
		Method:      "POST",
		Summary:     "List discovered scenarios",
		Description: "Walks <vrooli-root>/scenarios/* for .vrooli/service.json descriptors and returns one row per scenario.",
		Category:    "scenarios",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier scenarios list"},
	},
	{
		ID:          "scenarios.get",
		Path:        scenariosconnect.ScenariosServiceGetScenarioProcedure,
		Method:      "POST",
		Summary:     "Scenario detail",
		Description: "Returns the scenario descriptor plus the flow summaries discovered inside it.",
		Category:    "scenarios",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier scenarios show"},
	},
	{
		ID:          "scenarios.generate_artifacts",
		Path:        scenariosconnect.ScenariosServiceGenerateScenarioArtifactsProcedure,
		Method:      "POST",
		Summary:     "Generate every flow's artifacts in a scenario (server-stream)",
		Description: "Server-streams one ScenarioArtifactsProgress per flow as artifacts are regenerated.",
		Category:    "scenarios",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier artifacts generate --scenario <id>"},
	},
	{
		ID:          "scenarios.clear_artifacts",
		Path:        scenariosconnect.ScenariosServiceClearScenarioArtifactsProcedure,
		Method:      "POST",
		Summary:     "Clear every flow's generated/ tree in a scenario",
		Description: "Removes every generated/ tree under the scenario root. Refuses traversal outside root.",
		Category:    "scenarios",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier artifacts clear --scenario <id> --yes"},
	},
}
