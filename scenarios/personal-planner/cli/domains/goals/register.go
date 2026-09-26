package goals

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "goals"

func Register(c *cliapp.ScenarioApp, m []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(c)
	b := map[string]cliapp.PrimitiveHandler{
		"GoalsService.ListGoals":             cliapp.ProtoList(h.listCall, h.listReport),
		"GoalsService.CreateGoal":            cliapp.ProtoMutation(h.createCall, h.createReport),
		"GoalsService.UpdateGoalProgress":    cliapp.ProtoMutation(h.progressCall, h.progressReport),
		"GoalsService.ListMilestones":        cliapp.ProtoList(h.milestonesCall, h.milestonesReport),
		"GoalsService.CreateMilestone":       cliapp.ProtoMutation(h.createMilestoneCall, h.createMilestoneReport),
		"GoalsService.UpdateMilestoneStatus": cliapp.ProtoMutation(h.completeMilestoneCall, h.completeMilestoneReport),
	}
	g, e := cliapp.LoadFromManifestPrimitives(m, GroupName, b)
	if e != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("goals: load from manifest: %w", e)
	}
	return g, nil
}
