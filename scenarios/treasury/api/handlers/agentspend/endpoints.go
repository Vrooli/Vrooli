package agentspend

import (
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "agentspend_propose_charge", Path: authorizationconnect.AgentSpendProposeChargeProcedure, Method: "POST", Summary: "Evaluate and reserve a proposed charge", Category: "authorization", Request: &module.Schema{Type: "ProposeChargeRequest"}, Response: &module.Schema{Type: "ProposeChargeResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Malformed proposal"}, {Status: 500, Code: "internal", Description: "Fail-closed dependency or persistence failure"}}},
	{ID: "agentspend_get_authorization", Path: authorizationconnect.AgentSpendGetAuthorizationProcedure, Method: "POST", Summary: "Read an authorization decision", Category: "authorization", Request: &module.Schema{Type: "GetAuthorizationRequest"}, Response: &module.Schema{Type: "GetAuthorizationResponse"}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Authorization does not exist"}}},
	{ID: "agentspend_get_budget_headroom", Path: authorizationconnect.AgentSpendGetBudgetHeadroomProcedure, Method: "POST", Summary: "Compute spend headroom from Treasury authorization records", Category: "budget", Request: &module.Schema{Type: "GetBudgetHeadroomRequest"}, Response: &module.Schema{Type: "GetBudgetHeadroomResponse"}, Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Verified agent identity is required"}, {Status: 404, Code: "not_found", Description: "Budget does not exist"}}},
	{ID: "agentspend_list_mandates", Path: authorizationconnect.AgentSpendListMandatesProcedure, Method: "POST", Summary: "List mandates visible to the verified agent", Category: "mandate", Request: &module.Schema{Type: "ListMandatesRequest"}, Response: &module.Schema{Type: "ListMandatesResponse"}, Errors: []module.ErrorDesc{{Status: 501, Code: "unimplemented", Description: "Introduced with the governed agent CLI surface"}}},
	{ID: "agentspend_report_outcome", Path: authorizationconnect.AgentSpendReportOutcomeProcedure, Method: "POST", Summary: "Execute an approved charge once and return its durable rail outcome", Category: "settlement", Request: &module.Schema{Type: "ReportOutcomeRequest"}, Response: &module.Schema{Type: "ReportOutcomeResponse"}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid, unauthorized, expired, or idempotency-aliased settlement"}, {Status: 404, Code: "not_found", Description: "Settlement dependency not found"}, {Status: 500, Code: "internal", Description: "Settlement execution or durable outcome recording failed"}}},
}
