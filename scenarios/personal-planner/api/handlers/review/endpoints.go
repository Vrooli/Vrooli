package review

import (
	gc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review/review_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "review_daily", Path: gc.ReviewServiceGetDailySummaryProcedure, Method: "POST", Summary: "Get daily review", Description: "Summarizes recorded focus and honest coverage for a local day.", Category: "review"},
	{ID: "review_weekly", Path: gc.ReviewServiceGetWeeklySummaryProcedure, Method: "POST", Summary: "Get weekly review", Description: "Summarizes planned time, recorded focus, and honest coverage across a local week.", Category: "review"},
	{ID: "review_reflection", Path: gc.ReviewServiceGetReflectionProcedure, Method: "POST", Summary: "Read review reflection", Description: "Reads the optional reflection for a local day.", Category: "review"},
	{ID: "review_save_reflection", Path: gc.ReviewServiceSaveReflectionProcedure, Method: "POST", Summary: "Save review reflection", Description: "Saves an optional reflection for a local day.", Category: "review"},
}
