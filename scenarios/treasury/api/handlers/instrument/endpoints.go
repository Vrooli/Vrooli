package instrument

import (
	domain "treasury/internal/instrument"
	"treasury/internal/module"

	instrumentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/instrument/instrument_v1connect"
)

func Schema() string { return domain.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasury_instrument_register", Path: instrumentconnect.InstrumentAdminServiceRegisterInstrumentProcedure, Method: "POST", Summary: "Register a mandate-scoped instrument reference", Category: "instrument", Request: &module.Schema{Type: "RegisterInstrumentRequest"}, Response: &module.Schema{Type: "RegisterInstrumentResponse"}, Errors: instrumentErrors},
}

var instrumentErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or invalid operator credential"},
	{Status: 412, Code: "failed_precondition", Description: "Operator realm or instrument service is unavailable"},
}
