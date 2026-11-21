package mandate

import (
	"treasury/internal/module"

	mandateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate/mandate_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasury_mandate_create", Path: mandateconnect.MandateAdminServiceCreateMandateProcedure, Method: "POST", Summary: "Create a signed mandate", Category: "mandate", Request: &module.Schema{Type: "CreateMandateRequest"}, Response: &module.Schema{Type: "CreateMandateResponse"}, Errors: mandateErrors},
	{ID: "treasury_mandate_revoke", Path: mandateconnect.MandateAdminServiceRevokeMandateProcedure, Method: "POST", Summary: "Revoke a mandate and its active session", Category: "mandate", Request: &module.Schema{Type: "RevokeMandateRequest"}, Response: &module.Schema{Type: "RevokeMandateResponse"}, Errors: mandateErrors},
	{ID: "treasury_mandate_revoke_session", Path: mandateconnect.MandateAdminServiceRevokeMandateSessionProcedure, Method: "POST", Summary: "Revoke an active run-owned mandate session key", Category: "mandate", Request: &module.Schema{Type: "RevokeMandateSessionRequest"}, Response: &module.Schema{Type: "RevokeMandateSessionResponse"}, Errors: mandateErrors},
	{ID: "treasury_mandate_cancel_standing", Path: mandateconnect.MandateAdminServiceCancelStandingMandateProcedure, Method: "POST", Summary: "Cancel a standing mandate before its next recurrence", Category: "mandate", Request: &module.Schema{Type: "CancelStandingMandateRequest"}, Response: &module.Schema{Type: "CancelStandingMandateResponse"}, Errors: mandateErrors},
	{ID: "treasury_mandate_list", Path: mandateconnect.MandateAdminServiceListMandatesProcedure, Method: "POST", Summary: "List operator mandates across books", Category: "mandate", Request: &module.Schema{Type: "ListMandatesRequest"}, Response: &module.Schema{Type: "ListMandatesResponse"}, Errors: mandateErrors},
}

var mandateErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or invalid operator credential"},
	{Status: 404, Code: "not_found", Description: "Mandate or active session does not exist"},
	{Status: 412, Code: "failed_precondition", Description: "Operator realm or mandate service is unavailable"},
}
