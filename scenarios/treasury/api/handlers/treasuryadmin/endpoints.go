package treasuryadmin

import (
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/internal/module"
)

var adminErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or invalid operator credential"},
	{Status: 412, Code: "failed_precondition", Description: "Operator realm is not configured"},
	{Status: 501, Code: "unimplemented", Description: "Method behavior lands with its owning domain phase"},
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasuryadmin_create_mandate", Path: authorizationconnect.TreasuryAdminCreateMandateProcedure, Method: "POST", Summary: "Create a signed mandate", Category: "admin", Request: &module.Schema{Type: "CreateMandateRequest"}, Response: &module.Schema{Type: "CreateMandateResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_revoke_mandate", Path: authorizationconnect.TreasuryAdminRevokeMandateProcedure, Method: "POST", Summary: "Revoke a mandate", Category: "admin", Request: &module.Schema{Type: "RevokeMandateRequest"}, Response: &module.Schema{Type: "RevokeMandateResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_set_budget_caps", Path: authorizationconnect.TreasuryAdminSetBudgetCapsProcedure, Method: "POST", Summary: "Set budget caps", Category: "admin", Request: &module.Schema{Type: "SetBudgetCapsRequest"}, Response: &module.Schema{Type: "SetBudgetCapsResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_set_gating", Path: authorizationconnect.TreasuryAdminSetGatingProcedure, Method: "POST", Summary: "Set budget approval gating", Category: "admin", Request: &module.Schema{Type: "SetGatingRequest"}, Response: &module.Schema{Type: "SetGatingResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_resolve_approval", Path: authorizationconnect.TreasuryAdminResolveApprovalProcedure, Method: "POST", Summary: "Resolve a pending approval", Category: "admin", Request: &module.Schema{Type: "ResolveApprovalRequest"}, Response: &module.Schema{Type: "ResolveApprovalResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_freeze_budget", Path: authorizationconnect.TreasuryAdminFreezeBudgetProcedure, Method: "POST", Summary: "Freeze a budget", Category: "admin", Request: &module.Schema{Type: "FreezeBudgetRequest"}, Response: &module.Schema{Type: "FreezeBudgetResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_unfreeze_budget", Path: authorizationconnect.TreasuryAdminUnfreezeBudgetProcedure, Method: "POST", Summary: "Unfreeze a budget", Category: "admin", Request: &module.Schema{Type: "UnfreezeBudgetRequest"}, Response: &module.Schema{Type: "UnfreezeBudgetResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_register_instrument", Path: authorizationconnect.TreasuryAdminRegisterInstrumentProcedure, Method: "POST", Summary: "Register an instrument reference", Category: "admin", Request: &module.Schema{Type: "RegisterInstrumentRequest"}, Response: &module.Schema{Type: "RegisterInstrumentResponse"}, Errors: adminErrors},
}
