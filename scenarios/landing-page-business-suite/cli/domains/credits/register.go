package credits

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Credits",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "admin-api-keys-list", Method: "GET", Path: "/admin/api-keys", Description: "List API keys (admin)"},
			{Name: "admin-api-keys-create", Method: "POST", Path: "/admin/api-keys", Description: "Create API key (admin)"},
			{Name: "admin-api-keys-delete", Method: "DELETE", Path: "/admin/api-keys", Description: "Delete API key (admin)"},
			{Name: "admin-api-keys-test", Method: "POST", Path: "/admin/api-keys/test", Description: "Test API key (admin)"},
			{Name: "admin-api-keys-toggle", Method: "POST", Path: "/admin/api-keys/toggle", Description: "Toggle API key (admin)"},
			{Name: "admin-tiers-limits", Method: "GET", Path: "/admin/tiers/limits", Description: "List tier limits (admin)"},
			{Name: "admin-tier-limits", Method: "GET", Path: "/admin/tiers/{tier}/limits", Description: "Get tier limits (admin)"},
			{Name: "admin-tier-limits-update", Method: "PUT", Path: "/admin/tiers/{tier}/limits", Description: "Update tier limits (admin)"},
			{Name: "admin-limits-create", Method: "POST", Path: "/admin/limits", Description: "Create tier limit (admin)"},
			{Name: "admin-limits-delete", Method: "DELETE", Path: "/admin/limits", Description: "Delete tier limit (admin)"},
			{Name: "admin-app-limits", Method: "GET", Path: "/admin/apps/{app}/limits", Description: "Get app limits (admin)"},
			{Name: "usage-report", Method: "POST", Path: "/usage/report", Description: "Report usage (service auth)"},
			{Name: "usage-summary", Method: "GET", Path: "/usage/summary", Description: "Get usage summary"},
			{Name: "usage-check", Method: "GET", Path: "/usage/check", Description: "Check usage limits"},
			{Name: "usage-health", Method: "GET", Path: "/usage/health", Description: "Usage health"},
			{Name: "admin-usage-summary", Method: "GET", Path: "/admin/usage", Description: "Admin usage summary"},
		}),
	}
}
