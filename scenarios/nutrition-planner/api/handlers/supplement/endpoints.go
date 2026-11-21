package supplement

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/supplement/supplement_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "supplement_list_schedules", Path: connect.SupplementServiceListSchedulesProcedure, Method: "POST", Summary: "List supplement schedules", Category: "supplement"},
	{ID: "supplement_create_schedule", Path: connect.SupplementServiceCreateScheduleProcedure, Method: "POST", Summary: "Create a confirmed supplement schedule", Category: "supplement"},
	{ID: "supplement_update_schedule", Path: connect.SupplementServiceUpdateScheduleProcedure, Method: "POST", Summary: "Create a new supplement schedule revision", Category: "supplement"},
	{ID: "supplement_get_schedule", Path: connect.SupplementServiceGetScheduleProcedure, Method: "POST", Summary: "Get a supplement schedule revision", Category: "supplement"},
}
