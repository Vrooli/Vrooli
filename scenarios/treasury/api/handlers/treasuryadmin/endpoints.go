package treasuryadmin

import (
	"treasury/internal/module"

	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
)

var adminErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or invalid operator credential"},
	{Status: 412, Code: "failed_precondition", Description: "Operator realm is not configured"},
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasuryadmin_create_book", Path: authorizationconnect.TreasuryAdminCreateBookProcedure, Method: "POST", Summary: "Create an operator custody book", Category: "admin", Request: &module.Schema{Type: "CreateBookRequest"}, Response: &module.Schema{Type: "CreateBookResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_get_book", Path: authorizationconnect.TreasuryAdminGetBookProcedure, Method: "POST", Summary: "Read an operator custody book", Category: "admin", Request: &module.Schema{Type: "GetBookRequest"}, Response: &module.Schema{Type: "GetBookResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_create_mandate", Path: authorizationconnect.TreasuryAdminCreateMandateProcedure, Method: "POST", Summary: "Create a signed mandate", Category: "admin", Request: &module.Schema{Type: "CreateMandateRequest"}, Response: &module.Schema{Type: "CreateMandateResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_revoke_mandate", Path: authorizationconnect.TreasuryAdminRevokeMandateProcedure, Method: "POST", Summary: "Revoke a mandate", Category: "admin", Request: &module.Schema{Type: "RevokeMandateRequest"}, Response: &module.Schema{Type: "RevokeMandateResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_cancel_standing_mandate", Path: authorizationconnect.TreasuryAdminCancelStandingMandateProcedure, Method: "POST", Summary: "Cancel a standing mandate before its next recurrence", Category: "admin", Request: &module.Schema{Type: "CancelStandingMandateRequest"}, Response: &module.Schema{Type: "CancelStandingMandateResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_list_mandates", Path: authorizationconnect.TreasuryAdminListMandatesProcedure, Method: "POST", Summary: "List operator mandates across books", Category: "admin", Request: &module.Schema{Type: "ListMandatesRequest"}, Response: &module.Schema{Type: "ListMandatesResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_set_budget_caps", Path: authorizationconnect.TreasuryAdminSetBudgetCapsProcedure, Method: "POST", Summary: "Set budget caps", Category: "admin", Request: &module.Schema{Type: "SetBudgetCapsRequest"}, Response: &module.Schema{Type: "SetBudgetCapsResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_set_gating", Path: authorizationconnect.TreasuryAdminSetGatingProcedure, Method: "POST", Summary: "Set budget approval gating", Category: "admin", Request: &module.Schema{Type: "SetGatingRequest"}, Response: &module.Schema{Type: "SetGatingResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_list_approvals", Path: authorizationconnect.TreasuryAdminListApprovalsProcedure, Method: "POST", Summary: "List approval requests", Category: "admin", Request: &module.Schema{Type: "ListApprovalsRequest"}, Response: &module.Schema{Type: "ListApprovalsResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_resolve_approval", Path: authorizationconnect.TreasuryAdminResolveApprovalProcedure, Method: "POST", Summary: "Resolve a pending approval", Category: "admin", Request: &module.Schema{Type: "ResolveApprovalRequest"}, Response: &module.Schema{Type: "ResolveApprovalResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_freeze_budget", Path: authorizationconnect.TreasuryAdminFreezeBudgetProcedure, Method: "POST", Summary: "Freeze a budget", Category: "admin", Request: &module.Schema{Type: "FreezeBudgetRequest"}, Response: &module.Schema{Type: "FreezeBudgetResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_unfreeze_budget", Path: authorizationconnect.TreasuryAdminUnfreezeBudgetProcedure, Method: "POST", Summary: "Unfreeze a budget", Category: "admin", Request: &module.Schema{Type: "UnfreezeBudgetRequest"}, Response: &module.Schema{Type: "UnfreezeBudgetResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_freeze_book", Path: authorizationconnect.TreasuryAdminFreezeBookProcedure, Method: "POST", Summary: "Freeze every authorization and unsettled charge in a book", Category: "admin", Request: &module.Schema{Type: "FreezeBookRequest"}, Response: &module.Schema{Type: "FreezeBookResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_unfreeze_book", Path: authorizationconnect.TreasuryAdminUnfreezeBookProcedure, Method: "POST", Summary: "Release a book freeze", Category: "admin", Request: &module.Schema{Type: "UnfreezeBookRequest"}, Response: &module.Schema{Type: "UnfreezeBookResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_freeze_all", Path: authorizationconnect.TreasuryAdminFreezeAllProcedure, Method: "POST", Summary: "Engage the scenario-wide kill switch", Category: "admin", Request: &module.Schema{Type: "FreezeAllRequest"}, Response: &module.Schema{Type: "FreezeAllResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_get_freeze_status", Path: authorizationconnect.TreasuryAdminGetFreezeStatusProcedure, Method: "POST", Summary: "Read the scenario-wide kill-switch state", Category: "admin", Request: &module.Schema{Type: "GetFreezeStatusRequest"}, Response: &module.Schema{Type: "GetFreezeStatusResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_unfreeze_all", Path: authorizationconnect.TreasuryAdminUnfreezeAllProcedure, Method: "POST", Summary: "Release the scenario-wide kill switch", Category: "admin", Request: &module.Schema{Type: "UnfreezeAllRequest"}, Response: &module.Schema{Type: "UnfreezeAllResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_register_instrument", Path: authorizationconnect.TreasuryAdminRegisterInstrumentProcedure, Method: "POST", Summary: "Register an instrument reference", Category: "admin", Request: &module.Schema{Type: "RegisterInstrumentRequest"}, Response: &module.Schema{Type: "RegisterInstrumentResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_report_manual_outcome", Path: authorizationconnect.TreasuryAdminReportManualOutcomeProcedure, Method: "POST", Summary: "Record an operator-attested manual settlement", Category: "admin", Request: &module.Schema{Type: "ReportManualOutcomeRequest"}, Response: &module.Schema{Type: "ReportManualOutcomeResponse"}, Errors: adminErrors},
}
