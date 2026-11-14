package work

import (
	workconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work/work_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "work_list", Path: workconnect.WorkServiceListWorkItemsProcedure, Method: "POST", Summary: "List work items", Description: "Returns work items ordered newest-first.", Category: "work"},
	{ID: "work_create", Path: workconnect.WorkServiceCreateWorkItemProcedure, Method: "POST", Summary: "Create work item", Description: "Captures a work item with explicit remaining effort.", Category: "work"},
	{ID: "work_get", Path: workconnect.WorkServiceGetWorkItemProcedure, Method: "POST", Summary: "Get work item", Description: "Returns one work item by id.", Category: "work"},
	{ID: "work_today_plan", Path: workconnect.WorkServiceGetTodayPlanProcedure, Method: "POST", Summary: "Get today's plan", Description: "Projects captured work into a deterministic day plan and capacity summary.", Category: "work"},
}
