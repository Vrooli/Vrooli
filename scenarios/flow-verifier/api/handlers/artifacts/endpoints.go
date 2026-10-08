package artifacts

import (
	"flow-verifier/internal/module"

	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts/artifacts_v1connect"
)

// Endpoints describes the artifacts Connect-RPC surface. Scenario-level
// generate/clear live on ScenariosService (handlers/scenarios) so the
// streaming GenerateScenarioArtifacts RPC is colocated with the
// scenario index it operates on.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "flows.artifacts.status",
		Path:        artifactsconnect.ArtifactsServiceGetArtifactStatusProcedure,
		Method:      "POST",
		Summary:     "Inspect a flow's generated/ tree",
		Description: "Returns each expected artifact path with its existence/mtime, plus an overall fresh|missing status. Pure inspection — no mutation.",
		Category:    "artifacts",
	},
	{
		ID:          "flows.artifacts.generate",
		Path:        artifactsconnect.ArtifactsServiceGenerateArtifactsProcedure,
		Method:      "POST",
		Summary:     "Generate or regenerate a flow's artifacts",
		Description: "Runs the pipeline in generate mode for one flow. Returns the new artifacts status alongside the recorded run.",
		Category:    "artifacts",
	},
	{
		ID:          "flows.artifacts.clear",
		Path:        artifactsconnect.ArtifactsServiceClearArtifactsProcedure,
		Method:      "POST",
		Summary:     "Clear a flow's generated/ tree",
		Description: "Removes every file under <flow-dir>/generated/. Refuses crafted paths that would escape the scenario root.",
		Category:    "artifacts",
	},
}
