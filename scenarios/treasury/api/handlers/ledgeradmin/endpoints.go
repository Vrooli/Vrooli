package ledgeradmin

import (
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger/ledger_v1connect"
	"treasury/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasury_ledger_emissions_list", Path: ledgerconnect.LedgerServiceListEmissionsProcedure, Method: "POST", Summary: "Read durable Money Ledger delivery state", Category: "ledger", Request: &module.Schema{Type: "ListEmissionsRequest"}, Response: &module.Schema{Type: "ListEmissionsResponse"}, Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"}, {Status: 403, Code: "permission_denied", Description: "Caller is not in the operator realm"}, {Status: 400, Code: "invalid_argument", Description: "Unsupported status filter"}, {Status: 503, Code: "failed_precondition", Description: "Operator auth or readback is unavailable"}}},
}
