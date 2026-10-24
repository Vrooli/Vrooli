package planning

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "plan"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"PlanningService.CreatePlannedScenario":      h.create,
		"PlanningService.ListPlannedScenarios":       h.list,
		"PlanningService.GetPlannedScenario":         h.get,
		"PlanningService.PutPlannedProtoFile":        h.put,
		"PlanningService.DeletePlannedProtoFile":     h.remove,
		"PlanningService.ValidatePlannedScenario":    h.validate,
		"PlanningService.MaterializePlannedScenario": h.materialize,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("planning: load from manifest: %w", err)
	}
	return group, nil
}
