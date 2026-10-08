package planning

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning/planning_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "planning_generate", Path: connect.PlanningServiceGeneratePlanProcedure, Method: "POST", Summary: "Generate a deterministic dinner plan draft", Category: "planning"},
	{ID: "planning_get_plan", Path: connect.PlanningServiceGetPlanProcedure, Method: "POST", Summary: "Read persisted plan occurrences for a date range", Category: "planning"},
	{ID: "planning_apply", Path: connect.PlanningServiceApplyPlanProcedure, Method: "POST", Summary: "Apply a plan draft with optimistic concurrency", Category: "planning"},
	{ID: "planning_swap_preview", Path: connect.PlanningServicePreviewSwapProcedure, Method: "POST", Summary: "Preview a scoped meal swap", Category: "planning"},
	{ID: "planning_explore_recipes", Path: connect.PlanningServiceExploreRecipesProcedure, Method: "POST", Summary: "Discover saved recipes through shared eligibility", Category: "planning"},
	{ID: "planning_shopping_preview", Path: connect.PlanningServiceGetShoppingPreviewProcedure, Method: "POST", Summary: "Read an honest plan-derived shopping preview", Category: "shopping"},
	{ID: "planning_shopping_checked", Path: connect.PlanningServiceSetShoppingCheckedProcedure, Method: "POST", Summary: "Persist checklist state without changing inventory", Category: "shopping"},
	{ID: "planning_shopping_have_this", Path: connect.PlanningServiceSetShoppingHaveThisProcedure, Method: "POST", Summary: "Persist a qualitative pantry assertion without quantity", Category: "shopping"},
	{ID: "planning_shopping_confirm_purchases", Path: connect.PlanningServiceConfirmShoppingPurchasesProcedure, Method: "POST", Summary: "Confirm reviewed actual purchases atomically", Category: "shopping"},
	{ID: "planning_feedback_record", Path: connect.PlanningServiceRecordFeedbackProcedure, Method: "POST", Summary: "Record explicit meal feedback", Category: "feedback"},
	{ID: "planning_feedback_undo", Path: connect.PlanningServiceUndoFeedbackProcedure, Method: "POST", Summary: "Undo explicit meal feedback", Category: "feedback"},
}
