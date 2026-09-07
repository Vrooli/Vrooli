package mfa

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	mfaconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/mfa/mfa_v1connect"

	"scenario-authenticator/internal/accounts"
	auditlog "scenario-authenticator/internal/audit"
	intmfa "scenario-authenticator/internal/mfa"
	"scenario-authenticator/internal/module"
)

func Module(accountsService *accounts.Service, store *intmfa.Store, audit auditlog.Logger, logger *log.Logger) module.Module {
	path, handler := mfaconnect.NewMFAServiceHandler(NewConnectHandler(Deps{Accounts: accountsService, Store: store, Audit: audit, Logger: logger}))
	return module.Module{
		Name:      "mfa",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

func Schema() string { return intmfa.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "mfa_begin_enrollment", Path: mfaconnect.MFAServiceBeginEnrollmentProcedure, Method: "POST", Summary: "Begin MFA enrollment", Description: "Creates a short-lived encrypted TOTP enrollment and returns its QR-compatible provisioning URI.", Category: "mfa", Request: &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"enrollment_id": "string", "provisioning_uri": "string", "expires_at": "timestamp"}}, Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid or expired token"}}},
	{ID: "mfa_confirm_enrollment", Path: mfaconnect.MFAServiceConfirmEnrollmentProcedure, Method: "POST", Summary: "Confirm MFA enrollment", Description: "Confirms the TOTP code and returns one-time recovery codes once.", Category: "mfa", Request: &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)", "enrollment_id": "string (required)", "totp_code": "string (required)"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"recovery_codes": "array<string>", "enrolled_at": "timestamp"}}, Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid token or TOTP code"}, {Status: 412, Code: "failed_precondition", Description: "Enrollment expired"}}},
	{ID: "mfa_remove_enrollment", Path: mfaconnect.MFAServiceRemoveEnrollmentProcedure, Method: "POST", Summary: "Remove MFA enrollment", Description: "Removes the account TOTP seed and invalidates all recovery codes.", Category: "mfa", Request: &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{}}, Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid or expired token"}}},
}
