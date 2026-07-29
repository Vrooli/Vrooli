package feedback

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Engagement - Feedback",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "feedback-create", Method: "POST", Path: "/feedback", Description: "Submit feedback"},
			{Name: "admin-feedback-list", Method: "GET", Path: "/admin/feedback", Description: "List feedback (admin)"},
			{Name: "admin-feedback-bulk-delete", Method: "POST", Path: "/admin/feedback/bulk-delete", Description: "Bulk delete feedback (admin)"},
			{Name: "admin-feedback-get", Method: "GET", Path: "/admin/feedback/{id}", Description: "Get feedback (admin)"},
			{Name: "admin-feedback-delete", Method: "DELETE", Path: "/admin/feedback/{id}", Description: "Delete feedback (admin)"},
			{Name: "admin-feedback-status-update", Method: "PATCH", Path: "/admin/feedback/{id}/status", Description: "Update feedback status (admin)"},
		}),
	}
}
