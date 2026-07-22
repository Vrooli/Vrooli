package goals

import (
	"github.com/vrooli/cli-core/cliapp"
	"swarm-manager/cli/internal/support"
)

// Register exposes the goal/milestone CLI over the GoalService Connect API.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "goals", Description: "Goal and milestone management", Subcommands: []cliapp.Command{
		support.APICommand("list", "List goals [--json]", deps.GoalsList),
		support.APICommand("get", "Get a goal with scope (--name NAME) [--json]", deps.GoalsGet),
		support.APICommand("create", "Create goal (--name NAME --title TITLE [--targets kind/name,...]) [--json]", deps.GoalsCreate),
		support.APICommand("update", "Update goal (--name NAME [--title TITLE] [--description TEXT] [--priority N]) [--json]", deps.GoalsUpdate),
		support.APICommand("delete", "Delete goal (--name NAME) [--json]", deps.GoalsDelete),
		support.APICommand("archive", "Archive goal (--name NAME) [--json]", deps.GoalsArchive),
		support.APICommand("unarchive", "Unarchive goal (--name NAME) [--json]", deps.GoalsUnarchive),
		support.APICommand("context", "Get a goal graph snapshot (--name NAME) [--json]", deps.GoalsContext),
		support.APICommand("targets-add", "Add goal targets (--name NAME --targets kind/name,...) [--json]", deps.GoalsTargetsAdd),
		support.APICommand("targets-remove", "Remove goal targets (--name NAME --targets kind/name,...) [--json]", deps.GoalsTargetsRemove),
		support.APICommand("milestone-create", "Create milestone (--goal NAME --name NAME --title TITLE) [--json]", deps.GoalsMilestoneCreate),
		support.APICommand("milestone-assign", "Assign scoped items (--goal NAME --milestone NAME --items kind/name,...) [--json]", deps.GoalsMilestoneAssign),
		support.APICommand("milestone-unassign", "Unassign scoped items (--goal NAME --milestone NAME --items kind/name,...) [--json]", deps.GoalsMilestoneUnassign),
		support.APICommand("milestone-archive", "Archive milestone (--goal NAME --milestone NAME) [--json]", deps.GoalsMilestoneArchive),
	}}
}
