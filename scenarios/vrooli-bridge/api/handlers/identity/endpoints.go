package identity

import (
	"vrooli-bridge/internal/module"

	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity/identity_v1connect"
)

// Endpoints describes the identity module's public surface. Paths reference the
// generated *Procedure constants, so renaming an RPC in identity.proto breaks
// this file at compile time. The global parity test (registry_test.go) asserts
// every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "identity_login",
		Path:        identityconnect.IdentityServiceLoginProcedure,
		Method:      "POST",
		Summary:     "Owner sign-in (same-origin)",
		Description: "Forwards email + password to scenario-authenticator (resolved by name via api-core/discovery) and returns the issued owner JWT. The control plane owns no credential logic — it relays. Called same-origin by the bridge UI so the browser never makes a cross-origin call.",
		Category:    "identity",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"email": "string", "password": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"token": "string (owner JWT)", "refresh_token": "string", "email": "string", "user_id": "string"}}, //nolint:gosec // schema field labels in an API descriptor, not hardcoded credentials
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Invalid email or password"},
			{Status: 503, Code: "unavailable", Description: "scenario-authenticator could not be reached"},
		},
		Examples: []module.Example{
			{Name: "Sign in", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.identity.IdentityService/Login -H 'Content-Type: application/json' -d '{\"email\":\"you@example.com\",\"password\":\"…\"}'"},
		},
	},
	{
		ID:          "identity_register",
		Path:        identityconnect.IdentityServiceRegisterProcedure,
		Method:      "POST",
		Summary:     "Create owner account (same-origin)",
		Description: "Creates an owner account in scenario-authenticator and returns the issued owner JWT (the account is signed in immediately, so a fresh owner can register their first node in one flow). Relayed via api-core/discovery; same-origin only.",
		Category:    "identity",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"email": "string", "password": "string", "username": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"token": "string (owner JWT)", "refresh_token": "string", "email": "string", "user_id": "string"}}, //nolint:gosec // schema field labels in an API descriptor, not hardcoded credentials
		Errors: []module.ErrorDesc{
			{Status: 409, Code: "already_exists", Description: "Email already registered"},
			{Status: 400, Code: "invalid_argument", Description: "Weak password or malformed email (authenticator message relayed)"},
			{Status: 503, Code: "unavailable", Description: "scenario-authenticator could not be reached"},
		},
		Examples: []module.Example{
			{Name: "Create account", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.identity.IdentityService/Register -H 'Content-Type: application/json' -d '{\"email\":\"you@example.com\",\"password\":\"…\"}'"},
		},
	},
}
