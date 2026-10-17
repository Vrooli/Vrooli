package planning

import (
	"tech-tree-designer/internal/module"

	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning/planning_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "planning-create",
		Path:        planningconnect.PlanningServiceCreatePlannedScenarioProcedure,
		Method:      "POST",
		Summary:     "Create a planned scenario",
		Description: "Creates or updates a planned scenario metadata record for contract-first design.",
		Category:    "planning",
		Request:     &module.Schema{Type: "CreatePlannedScenarioRequest"},
		Response:    &module.Schema{Type: "PlannedScenario"},
	},
	{
		ID:          "planning-list",
		Path:        planningconnect.PlanningServiceListPlannedScenariosProcedure,
		Method:      "POST",
		Summary:     "List planned scenarios",
		Description: "Lists planned scenario records and their proto file trees.",
		Category:    "planning",
		Request:     &module.Schema{Type: "ListPlannedScenariosRequest"},
		Response:    &module.Schema{Type: "ListPlannedScenariosResponse"},
	},
	{
		ID:          "planning-get",
		Path:        planningconnect.PlanningServiceGetPlannedScenarioProcedure,
		Method:      "POST",
		Summary:     "Get a planned scenario",
		Description: "Returns planned scenario metadata and stored proto files.",
		Category:    "planning",
		Request:     &module.Schema{Type: "GetPlannedScenarioRequest"},
		Response:    &module.Schema{Type: "PlannedScenario"},
	},
	{
		ID:          "planning-file-put",
		Path:        planningconnect.PlanningServicePutPlannedProtoFileProcedure,
		Method:      "POST",
		Summary:     "Add or edit a planned proto file",
		Description: "Stores or replaces real .proto text for a planned scenario file path.",
		Category:    "planning",
		Request:     &module.Schema{Type: "PutPlannedProtoFileRequest"},
		Response:    &module.Schema{Type: "PlannedProtoFile"},
	},
	{
		ID:          "planning-file-delete",
		Path:        planningconnect.PlanningServiceDeletePlannedProtoFileProcedure,
		Method:      "POST",
		Summary:     "Remove a planned proto file",
		Description: "Deletes one stored proto file from a planned scenario.",
		Category:    "planning",
		Request:     &module.Schema{Type: "DeletePlannedProtoFileRequest"},
		Response:    &module.Schema{Type: "DeletePlannedProtoFileResponse"},
	},
	{
		ID:          "planning-validate",
		Path:        planningconnect.PlanningServiceValidatePlannedScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate planned proto files",
		Description: "Compiles planned proto text with live schemas and returns structure, naming, import, and stability findings.",
		Category:    "planning",
		Request:     &module.Schema{Type: "ValidatePlannedScenarioRequest"},
		Response:    &module.Schema{Type: "ValidatePlannedScenarioResponse"},
	},
	{
		ID:          "planning-materialize",
		Path:        planningconnect.PlanningServiceMaterializePlannedScenarioProcedure,
		Method:      "POST",
		Summary:     "Materialize planned proto files",
		Description: "Writes validated planned proto text into packages/proto/schemas/<slug>/ and regenerates proto artifacts.",
		Category:    "planning",
		Request:     &module.Schema{Type: "MaterializePlannedScenarioRequest"},
		Response:    &module.Schema{Type: "MaterializePlannedScenarioResponse"},
	},
}
