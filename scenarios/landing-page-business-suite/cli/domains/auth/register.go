package auth

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Auth",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "auth-magic-link", Method: "POST", Path: "/auth/magic-link", Description: "Request a magic link"},
			{Name: "auth-verify", Method: "POST", Path: "/auth/verify", Description: "Verify a magic link token"},
			{Name: "auth-verify-code", Method: "POST", Path: "/auth/verify-code", Description: "Verify an emailed sign-in code"},
			{Name: "auth-magic-link-preview", Method: "POST", Path: "/auth/magic-link/preview", Description: "Preview a magic link without consuming it"},
			{Name: "auth-refresh", Method: "POST", Path: "/auth/refresh", Description: "Refresh auth token"},
			{Name: "auth-logout", Method: "POST", Path: "/auth/logout", Description: "Logout current user"},
			{Name: "auth-me", Method: "GET", Path: "/auth/me", Description: "Get authenticated user"},
		}),
	}
}
