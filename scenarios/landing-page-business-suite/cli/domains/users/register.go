package users

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Admin Users",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "admin-users-list", Method: "GET", Path: "/admin/users", Description: "List users"},
			{Name: "admin-users-get", Method: "GET", Path: "/admin/users/{id}", Description: "Get user"},
			{Name: "admin-users-sessions", Method: "GET", Path: "/admin/users/{id}/sessions", Description: "List user sessions"},
			{Name: "admin-users-session-revoke", Method: "DELETE", Path: "/admin/users/{id}/sessions/{sid}", Description: "Revoke user session"},
			{Name: "admin-users-sessions-revoke-all", Method: "POST", Path: "/admin/users/{id}/sessions/revoke-all", Description: "Revoke all user sessions"},
		}),
	}
}
