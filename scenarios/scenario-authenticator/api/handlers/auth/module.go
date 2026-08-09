package auth

import (
	"log"

	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

// Module returns the AccountsService Connect-RPC contribution. The accounts
// schema (+ realms seed) is registered separately in modules.AllSchemas; this
// module owns only the transport mount.
func Module(svc *accounts.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := accountsconnect.NewAccountsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "auth",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the accounts schema so the modules registry can collect it
// from this handler package uniformly with the other domains.
func Schema() string { return accounts.Schema() }

// Endpoints describes the AccountsService surface. Paths reference the generated
// *Procedure constants so renaming an RPC in accounts.proto breaks this at
// compile time; the global parity test asserts every RPC has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "auth_register",
		Path:        accountsconnect.AccountsServiceRegisterProcedure,
		Method:      "POST",
		Summary:     "Register an account",
		Description: "Creates an account in a realm and returns it auto-signed-in (access + refresh token). Duplicate email yields already_exists; weak input yields invalid_argument.",
		Category:    "auth",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"email": "string (required)", "password": "string (required, >=8 chars, upper+lower+digit)",
			"username": "string", "realm": "string (default realm if empty)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"account": "Account", "tokens": "TokenPair",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid email or weak password"},
			{Status: 409, Code: "already_exists", Description: "Email already registered in the realm"},
		},
		Examples: []module.Example{{Name: "Register", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.accounts.AccountsService/Register -H 'Content-Type: application/json' -d '{\"email\":\"a@b.co\",\"password\":\"Passw0rd\"}'"}},
	},
	{
		ID:          "auth_login",
		Path:        accountsconnect.AccountsServiceLoginProcedure,
		Method:      "POST",
		Summary:     "Sign in",
		Description: "Verifies credentials and issues a fresh access + refresh token. Unknown account and wrong password both yield unauthenticated (anti-enumeration); a locked account yields permission_denied.",
		Category:    "auth",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"email": "string (required)", "password": "string (required)", "realm": "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"account": "Account", "tokens": "TokenPair",
		}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Invalid email or password"},
			{Status: 403, Code: "permission_denied", Description: "Account temporarily locked"},
		},
		Examples: []module.Example{{Name: "Login", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.accounts.AccountsService/Login -H 'Content-Type: application/json' -d '{\"email\":\"a@b.co\",\"password\":\"Passw0rd\"}'"}},
	},
	{
		ID:          "auth_refresh",
		Path:        accountsconnect.AccountsServiceRefreshProcedure,
		Method:      "POST",
		Summary:     "Rotate a refresh token",
		Description: "Rotates the presented refresh token single-use and mints a new access token. Replaying an already-rotated token revokes the whole token family (reuse detection) and yields unauthenticated.",
		Category:    "auth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"refresh_token": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"tokens": "TokenPair"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid or replayed refresh token"}},
		Examples:    []module.Example{{Name: "Refresh", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.accounts.AccountsService/Refresh -H 'Content-Type: application/json' -d '{\"refresh_token\":\"...\"}'"}},
	},
	{
		ID:          "auth_change_password",
		Path:        accountsconnect.AccountsServiceChangePasswordProcedure,
		Method:      "POST",
		Summary:     "Change account password",
		Description: "Verifies the current password, stores a new Argon2id hash, revokes every live session, and records an audit event.",
		Category:    "auth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)", "current_password": "string (required)", "new_password": "string (required, >=8 chars, upper+lower+digit)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"revoked_sessions": "int64"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "New password does not meet policy"},
			{Status: 401, Code: "unauthenticated", Description: "Invalid access token or current password"},
		},
	},
	{
		ID:          "auth_logout",
		Path:        accountsconnect.AccountsServiceLogoutProcedure,
		Method:      "POST",
		Summary:     "Sign out",
		Description: "Blacklists the access token until its own expiry and revokes the caller's sessions. Idempotent.",
		Category:    "auth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{}},
		Examples:    []module.Example{{Name: "Logout", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.accounts.AccountsService/Logout -H 'Content-Type: application/json' -d '{\"access_token\":\"...\"}'"}},
	},
	{
		ID:          "auth_validate",
		Path:        accountsconnect.AccountsServiceValidateProcedure,
		Method:      "POST",
		Summary:     "Validate an access token",
		Description: "Verifies an access token server-side (signature, expiry, issuer, realm aud, not blacklisted) and returns its claims. Relying parties should prefer local JWKS verification; this is for callers that cannot verify locally.",
		Category:    "auth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"valid": "bool", "user_id": "string", "email": "string", "roles": "array<string>", "scopes": "array<string>", "realm": "string", "expires_at": "timestamp",
		}},
		Examples: []module.Example{{Name: "Validate", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.accounts.AccountsService/Validate -H 'Content-Type: application/json' -d '{\"access_token\":\"...\"}'"}},
	},
	{
		ID:          "auth_grant_scope",
		Path:        accountsconnect.AccountsServiceGrantScopeProcedure,
		Method:      "POST",
		Summary:     "Grant a principal scope",
		Description: "Assigns an opaque scope to the authenticated principal and records an audit event.",
		Category:    "authorization",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)", "principal_id": "string", "scope": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"principal_id": "string", "scopes": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scope must not be empty"}, {Status: 401, Code: "unauthenticated", Description: "Invalid access token"}},
	},
	{
		ID:          "auth_revoke_scope",
		Path:        accountsconnect.AccountsServiceRevokeScopeProcedure,
		Method:      "POST",
		Summary:     "Revoke a principal scope",
		Description: "Removes an opaque scope from the authenticated principal and records an audit event.",
		Category:    "authorization",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)", "principal_id": "string", "scope": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"principal_id": "string", "scopes": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scope must not be empty"}, {Status: 401, Code: "unauthenticated", Description: "Invalid access token"}},
	},
	{
		ID:          "auth_list_scopes",
		Path:        accountsconnect.AccountsServiceListScopesProcedure,
		Method:      "POST",
		Summary:     "List principal scopes",
		Description: "Lists opaque scopes assigned to the authenticated principal.",
		Category:    "authorization",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)", "principal_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"principal_id": "string", "scopes": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid access token"}},
	},
	{
		ID:          "auth_link_machine_account",
		Path:        accountsconnect.AccountsServiceLinkMachineAccountProcedure,
		Method:      "POST",
		Summary:     "Link a local machine principal",
		Description: "Binds the signed-in account to a machine and opaque local principal. A default binding is required for password-free exchange.",
		Category:    "auth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)", "machine_id": "string (required)", "local_principal": "string (required)", "realm": "string", "is_default": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"machine_id": "string", "local_principal": "string", "account_id": "string", "realm": "string", "is_default": "bool", "linked_at": "timestamp"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid machine binding"}, {Status: 401, Code: "unauthenticated", Description: "Invalid access token"}},
	},
	{
		ID:          "auth_exchange_machine_principal",
		Path:        accountsconnect.AccountsServiceExchangeMachinePrincipalProcedure,
		Method:      "POST",
		Summary:     "Exchange a local principal for a token",
		Description: "Local Unix-socket-only exchange of a kernel-verified principal for a normal short-lived access and refresh token.",
		Category:    "auth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"machine_id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"account": "Account", "tokens": "TokenPair"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Unbound or unverified local principal"}, {Status: 412, Code: "failed_precondition", Description: "Ambiguous default binding"}},
	},
	{
		ID:          "auth_issue_break_glass",
		Path:        accountsconnect.AccountsServiceIssueBreakGlassProcedure,
		Method:      "POST",
		Summary:     "Issue a short-lived break-glass credential",
		Description: "Issues an offline-verifiable credential from provisioned owner-only material, bounded by the account's current scopes.",
		Category:    "auth",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)", "scopes": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"credential": "string", "expires_at": "int64"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid access token or unavailable provision"}, {Status: 400, Code: "invalid_argument", Description: "Requested scope exceeds account ceiling"}},
	},
}
