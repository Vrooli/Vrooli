package cooking

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cooking/cooking_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "cooking_start_session", Path: connect.CookingServiceStartSessionProcedure, Method: "POST", Summary: "Start a cooking session pinned to a recipe revision and method", Category: "cooking"},
	{ID: "cooking_get_session", Path: connect.CookingServiceGetSessionProcedure, Method: "POST", Summary: "Resume a saved cooking session", Category: "cooking"},
	{ID: "cooking_list_sessions", Path: connect.CookingServiceListSessionsProcedure, Method: "POST", Summary: "List cooking sessions in a workspace", Category: "cooking"},
	{ID: "cooking_save_session", Path: connect.CookingServiceSaveSessionProcedure, Method: "POST", Summary: "Save explicit cooking progress, timers, or completion", Category: "cooking"},
}
