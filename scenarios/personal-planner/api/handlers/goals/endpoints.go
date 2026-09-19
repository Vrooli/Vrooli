package goals

import (
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals/goals_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "goals_list", Path: gc.GoalsServiceListGoalsProcedure, Method: "POST", Summary: "List goals", Description: "Lists active and historical goals.", Category: "goals"},
	{ID: "goals_create", Path: gc.GoalsServiceCreateGoalProcedure, Method: "POST", Summary: "Create goal", Description: "Creates an outcome-oriented goal.", Category: "goals"},
	{ID: "goals_update_progress", Path: gc.GoalsServiceUpdateGoalProgressProcedure, Method: "POST", Summary: "Update goal progress", Description: "Records explicit goal progress.", Category: "goals"},
	{ID: "goals_list_milestones", Path: gc.GoalsServiceListMilestonesProcedure, Method: "POST", Summary: "List goal milestones", Description: "Lists milestones for a goal.", Category: "goals"},
	{ID: "goals_create_milestone", Path: gc.GoalsServiceCreateMilestoneProcedure, Method: "POST", Summary: "Create goal milestone", Description: "Creates a measurable goal milestone.", Category: "goals"},
	{ID: "goals_update_milestone_status", Path: gc.GoalsServiceUpdateMilestoneStatusProcedure, Method: "POST", Summary: "Update milestone status", Description: "Marks a goal milestone open or complete.", Category: "goals"},
}
