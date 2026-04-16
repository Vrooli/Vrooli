package waitlist

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Engagement - Waitlist",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "waitlist-create", Method: "POST", Path: "/waitlist", Description: "Create waitlist entry"},
			{Name: "admin-waitlist-list", Method: "GET", Path: "/admin/waitlist", Description: "List waitlist entries (admin)"},
			{Name: "admin-waitlist-delete", Method: "DELETE", Path: "/admin/waitlist/{id}", Description: "Delete waitlist entry (admin)"},
			{Name: "admin-waitlist-export", Method: "GET", Path: "/admin/waitlist/export", Description: "Export waitlist entries (admin)"},
		}),
	}
}
