package budget

import (
	"treasury/internal/module"

	budgetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget/budget_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasury_budget_set_caps", Path: budgetconnect.BudgetAdminServiceSetBudgetCapsProcedure, Method: "POST", Summary: "Set spend caps and counterparty scope for a budget", Category: "budget", Request: &module.Schema{Type: "SetBudgetCapsRequest"}, Response: &module.Schema{Type: "SetBudgetCapsResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_set_gating", Path: budgetconnect.BudgetAdminServiceSetGatingProcedure, Method: "POST", Summary: "Set approval gating for a budget", Category: "budget", Request: &module.Schema{Type: "SetGatingRequest"}, Response: &module.Schema{Type: "SetGatingResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_freeze", Path: budgetconnect.BudgetAdminServiceFreezeBudgetProcedure, Method: "POST", Summary: "Freeze the budget before its next authorization", Category: "budget", Request: &module.Schema{Type: "FreezeBudgetRequest"}, Response: &module.Schema{Type: "FreezeBudgetResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_unfreeze", Path: budgetconnect.BudgetAdminServiceUnfreezeBudgetProcedure, Method: "POST", Summary: "Release a budget freeze", Category: "budget", Request: &module.Schema{Type: "UnfreezeBudgetRequest"}, Response: &module.Schema{Type: "UnfreezeBudgetResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_freeze_book", Path: budgetconnect.BudgetAdminServiceFreezeBookProcedure, Method: "POST", Summary: "Freeze every authorization and unsettled charge in a book", Category: "budget", Request: &module.Schema{Type: "FreezeBookRequest"}, Response: &module.Schema{Type: "FreezeBookResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_unfreeze_book", Path: budgetconnect.BudgetAdminServiceUnfreezeBookProcedure, Method: "POST", Summary: "Release a book freeze", Category: "budget", Request: &module.Schema{Type: "UnfreezeBookRequest"}, Response: &module.Schema{Type: "UnfreezeBookResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_freeze_all", Path: budgetconnect.BudgetAdminServiceFreezeAllProcedure, Method: "POST", Summary: "Engage the scenario-wide kill switch", Category: "budget", Request: &module.Schema{Type: "FreezeAllRequest"}, Response: &module.Schema{Type: "FreezeAllResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_freeze_status", Path: budgetconnect.BudgetAdminServiceGetFreezeStatusProcedure, Method: "POST", Summary: "Read the scenario-wide kill-switch state", Category: "budget", Request: &module.Schema{Type: "GetFreezeStatusRequest"}, Response: &module.Schema{Type: "GetFreezeStatusResponse"}, Errors: budgetErrors},
	{ID: "treasury_budget_unfreeze_all", Path: budgetconnect.BudgetAdminServiceUnfreezeAllProcedure, Method: "POST", Summary: "Release the scenario-wide kill switch", Category: "budget", Request: &module.Schema{Type: "UnfreezeAllRequest"}, Response: &module.Schema{Type: "UnfreezeAllResponse"}, Errors: budgetErrors},
}

var budgetErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or invalid operator credential"},
	{Status: 412, Code: "failed_precondition", Description: "Operator realm or budget service is unavailable"},
}
