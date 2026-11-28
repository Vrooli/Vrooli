package nutrition

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition/nutrition_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "nutrition_list_targets", Path: connect.NutritionServiceListTargetsProcedure, Method: "POST", Summary: "List nutrition targets", Category: "nutrition"},
	{ID: "nutrition_create_target", Path: connect.NutritionServiceCreateTargetProcedure, Method: "POST", Summary: "Create a nutrition target", Category: "nutrition"},
	{ID: "nutrition_get_target", Path: connect.NutritionServiceGetTargetProcedure, Method: "POST", Summary: "Get a nutrition target revision", Category: "nutrition"},
	{ID: "nutrition_evaluate_scope", Path: connect.NutritionServiceEvaluateScopeProcedure, Method: "POST", Summary: "Evaluate an explicit intake scope", Category: "nutrition"},
	{ID: "nutrition_list_intakes", Path: connect.NutritionServiceListIntakesProcedure, Method: "POST", Summary: "List recorded intake events", Category: "nutrition"},
	{ID: "nutrition_record_intake", Path: connect.NutritionServiceRecordIntakeProcedure, Method: "POST", Summary: "Record or correct an intake event", Category: "nutrition"},
}
