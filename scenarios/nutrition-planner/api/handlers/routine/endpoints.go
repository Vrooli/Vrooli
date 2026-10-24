package routine

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/routine/routine_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "routine_list_templates", Path: connect.RoutineServiceListTemplatesProcedure, Method: "POST", Summary: "List recurring meal templates", Category: "routine"},
	{ID: "routine_create_template", Path: connect.RoutineServiceCreateTemplateProcedure, Method: "POST", Summary: "Create a recurring meal template", Category: "routine"},
	{ID: "routine_update_template", Path: connect.RoutineServiceUpdateTemplateProcedure, Method: "POST", Summary: "Create a new recurring template revision", Category: "routine"},
	{ID: "routine_generate_occurrences", Path: connect.RoutineServiceGenerateOccurrencesProcedure, Method: "POST", Summary: "Generate dated routine occurrences", Category: "routine"},
}
