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
	{ID: "treasuryadmin_list_approvals", Path: authorizationconnect.TreasuryAdminListApprovalsProcedure, Method: "POST", Summary: "List approval requests", Category: "admin", Request: &module.Schema{Type: "ListApprovalsRequest"}, Response: &module.Schema{Type: "ListApprovalsResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_resolve_approval", Path: authorizationconnect.TreasuryAdminResolveApprovalProcedure, Method: "POST", Summary: "Resolve a pending approval", Category: "admin", Request: &module.Schema{Type: "ResolveApprovalRequest"}, Response: &module.Schema{Type: "ResolveApprovalResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_report_manual_outcome", Path: authorizationconnect.TreasuryAdminReportManualOutcomeProcedure, Method: "POST", Summary: "Record an operator-attested manual settlement", Category: "admin", Request: &module.Schema{Type: "ReportManualOutcomeRequest"}, Response: &module.Schema{Type: "ReportManualOutcomeResponse"}, Errors: adminErrors},
	{ID: "treasuryadmin_resolve_unknown_settlement", Path: authorizationconnect.TreasuryAdminResolveUnknownSettlementProcedure, Method: "POST", Summary: "Reconcile an unknown or interrupted rail outcome through its registered adapter", Category: "admin", Request: &module.Schema{Type: "ResolveUnknownSettlementRequest"}, Response: &module.Schema{Type: "ResolveUnknownSettlementResponse"}, Errors: adminErrors},
}
