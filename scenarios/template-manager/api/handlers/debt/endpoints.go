package debt

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	debtconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/debt/debt_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "debt_list", Path: debtconnect.DebtServiceListDebtProcedure, Method: "POST", Summary: "List template debt", Description: "Returns durable template debt entries.", Category: "debt"},
	{ID: "debt_get", Path: debtconnect.DebtServiceGetDebtProcedure, Method: "POST", Summary: "Get template debt", Description: "Returns one durable template debt entry.", Category: "debt"},
}
