package policy

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Role Policy",
		Commands: []cliapp.Command{
			support.Command("role-policy", "Inspect, validate, reload, and explain portable role policy", deps.Policy),
			support.Command("permission-policy", "Inspect, plan, and reconcile global portable permissions", deps.PermissionPolicy),
		},
	}
}
