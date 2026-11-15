package funding

import (
	"treasury/internal/module"

	fundingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding/funding_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasury_funding_intent_create", Path: fundingconnect.FundingAdminServiceCreateFundingIntentProcedure, Method: "POST", Summary: "Create a durable funding intent for a budget", Category: "funding", Request: &module.Schema{Type: "CreateFundingIntentRequest"}, Response: &module.Schema{Type: "FundingIntentResponse"}, Errors: fundingErrors},
	{ID: "treasury_funding_intent_get", Path: fundingconnect.FundingAdminServiceGetFundingIntentProcedure, Method: "POST", Summary: "Read an operator-owned funding intent", Category: "funding", Request: &module.Schema{Type: "FundingIntentRequest"}, Response: &module.Schema{Type: "FundingIntentResponse"}, Errors: fundingErrors},
	{ID: "treasury_funding_transfer_confirm", Path: fundingconnect.FundingAdminServiceConfirmFundingTransferProcedure, Method: "POST", Summary: "Confirm a funding transfer against its verified receipt", Category: "funding", Request: &module.Schema{Type: "ConfirmFundingTransferRequest"}, Response: &module.Schema{Type: "FundingIntentResponse"}, Errors: fundingErrors},
	{ID: "treasury_funding_intent_list", Path: fundingconnect.FundingAdminServiceListFundingIntentsProcedure, Method: "POST", Summary: "List funding intents for a budget", Category: "funding", Request: &module.Schema{Type: "ListFundingIntentsRequest"}, Response: &module.Schema{Type: "ListFundingIntentsResponse"}, Errors: fundingErrors},
	{ID: "treasury_budget_wallet_provision", Path: fundingconnect.FundingAdminServiceProvisionBudgetWalletProcedure, Method: "POST", Summary: "Provision a derived wallet for an operator-owned budget and agent run", Category: "funding", Request: &module.Schema{Type: "ProvisionBudgetWalletRequest"}, Response: &module.Schema{Type: "BudgetWalletResponse"}, Errors: fundingErrors},
	{ID: "treasury_budget_wallet_get", Path: fundingconnect.FundingAdminServiceGetBudgetWalletProcedure, Method: "POST", Summary: "Read an operator-owned budget wallet address", Category: "funding", Request: &module.Schema{Type: "GetBudgetWalletRequest"}, Response: &module.Schema{Type: "BudgetWalletResponse"}, Errors: fundingErrors},
	{ID: "treasury_withdrawal_destination_set", Path: fundingconnect.FundingAdminServiceSetWithdrawalDestinationProcedure, Method: "POST", Summary: "Allowlist an operator-controlled withdrawal destination", Category: "funding", Request: &module.Schema{Type: "SetWithdrawalDestinationRequest"}, Response: &module.Schema{Type: "WithdrawalDestinationResponse"}, Errors: fundingErrors},
	{ID: "treasury_withdrawal_preview", Path: fundingconnect.FundingAdminServicePreviewWithdrawalProcedure, Method: "POST", Summary: "Preview an allowlisted withdrawal with fee and time estimates", Category: "funding", Request: &module.Schema{Type: "PreviewWithdrawalRequest"}, Response: &module.Schema{Type: "WithdrawalQuoteResponse"}, Errors: fundingErrors},
	{ID: "treasury_withdrawal_execute", Path: fundingconnect.FundingAdminServiceExecuteWithdrawalProcedure, Method: "POST", Summary: "Execute an approved local-fork withdrawal once", Category: "funding", Request: &module.Schema{Type: "ExecuteWithdrawalRequest"}, Response: &module.Schema{Type: "WithdrawalResponse"}, Errors: fundingErrors},
	{ID: "treasury_withdrawal_get", Path: fundingconnect.FundingAdminServiceGetWithdrawalProcedure, Method: "POST", Summary: "Read a persisted withdrawal outcome", Category: "funding", Request: &module.Schema{Type: "GetWithdrawalRequest"}, Response: &module.Schema{Type: "WithdrawalResponse"}, Errors: fundingErrors},
	{ID: "treasury_receipt_funding_submit", Path: fundingconnect.FundingAdminServiceSubmitReceiptFundingProcedure, Method: "POST", Summary: "Submit retained token evidence for exact ledger acceptance and budget funding", Category: "funding", Request: &module.Schema{Type: "SubmitReceiptFundingRequest"}, Response: &module.Schema{Type: "ReceiptFundingResponse"}, Errors: fundingErrors},
	{ID: "treasury_receipt_funding_get", Path: fundingconnect.FundingAdminServiceGetReceiptFundingProcedure, Method: "POST", Summary: "Read receipt acceptance and budget funding delivery", Category: "funding", Request: &module.Schema{Type: "GetReceiptFundingRequest"}, Response: &module.Schema{Type: "ReceiptFundingResponse"}, Errors: fundingErrors},
}

var AgentEndpoints = []module.EndpointDescriptor{
	{ID: "funding_agent_budget_wallet_get", Path: fundingconnect.FundingAgentServiceGetBudgetWalletProcedure, Method: "POST", Summary: "Bind the agent-run public session key and read its mandate-bound derived budget wallet", Category: "funding", Request: &module.Schema{Type: "GetBudgetWalletRequest"}, Response: &module.Schema{Type: "BudgetWalletResponse"}, Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Verified agent identity is required"}, {Status: 403, Code: "permission_denied", Description: "Wallet is bound to another agent run or budget grant"}, {Status: 404, Code: "not_found", Description: "Budget wallet does not exist"}}},
}

var fundingErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or invalid operator credential"},
	{Status: 404, Code: "not_found", Description: "Funding intent or budget does not exist"},
	{Status: 409, Code: "already_exists", Description: "Funding intent identity conflicts with an existing request"},
	{Status: 412, Code: "failed_precondition", Description: "Funding intent service is unavailable"},
}
