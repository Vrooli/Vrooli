package sessions

import (
	"log"

	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions/sessions_v1connect"
)

// Module returns the SessionsService Connect-RPC contribution. Sessions are
// Redis-backed hot state owned by internal/sessions; this module owns only the
// transport mount (no SQL schema).
func Module(svc *accounts.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := sessionsconnect.NewSessionsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "sessions",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Endpoints describes the SessionsService surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "sessions_list",
		Path:        sessionsconnect.SessionsServiceListSessionsProcedure,
		Method:      "POST",
		Summary:     "List active sessions",
		Description: "Returns the active sessions for the access token's owner.",
		Category:    "sessions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"sessions": "array<Session>"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid or expired token"}},
		Examples:    []module.Example{{Name: "List sessions", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.sessions.SessionsService/ListSessions -H 'Content-Type: application/json' -d '{\"access_token\":\"...\"}'"}},
	},
	{
		ID:          "sessions_revoke",
		Path:        sessionsconnect.SessionsServiceRevokeSessionProcedure,
		Method:      "POST",
		Summary:     "Revoke a session",
		Description: "Drops a single session by id. Idempotent — revoking a session that is already gone succeeds (the device-sync-hub un-pair contract).",
		Category:    "sessions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"session_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{}},
		Examples:    []module.Example{{Name: "Revoke session", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.sessions.SessionsService/RevokeSession -H 'Content-Type: application/json' -d '{\"session_id\":\"...\"}'"}},
	},
	{
		ID:          "sessions_revoke_all",
		Path:        sessionsconnect.SessionsServiceRevokeAllSessionsProcedure,
		Method:      "POST",
		Summary:     "Revoke all sessions",
		Description: "Drops every session for the access token's owner (log out everywhere) and returns how many were revoked.",
		Category:    "sessions",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"access_token": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"revoked_count": "int64"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Invalid or expired token"}},
		Examples:    []module.Example{{Name: "Revoke all", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.sessions.SessionsService/RevokeAllSessions -H 'Content-Type: application/json' -d '{\"access_token\":\"...\"}'"}},
	},
}
