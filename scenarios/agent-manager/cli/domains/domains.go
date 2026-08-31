package domains

import (
	"agent-manager/cli/domains/measures"
	"agent-manager/cli/domains/runs"
	"agent-manager/cli/internal/support"

	"github.com/vrooli/api-core/spacecli"
	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		runs.Register(deps),
	}
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	spaceCommand := spacecli.CommandGroup(spacecli.Config{Owner: "agent-manager", Projection: spacedoc.ProjectionAgentThroughput}).Commands[0]
	spaceCommand.Name = "get"
	space := cliapp.SubcommandGroup{Name: "space", Description: "Inspect the coverage-space denominator", DefaultSubcommand: "get", Subcommands: []cliapp.Command{spaceCommand}}
	return []cliapp.SubcommandGroup{
		support.SubcommandGroup("profile", "Manage agent profiles", deps.Profile,
			[2]string{"list", "List all agent profiles"}, [2]string{"get", "Get profile details by id or key"}, [2]string{"create", "Create a new profile"}, [2]string{"update", "Update an existing profile"}, [2]string{"delete", "Delete a profile"}, [2]string{"ensure", "Resolve a profile key, creating defaults if needed"}, [2]string{"reconcile-scenario", "Reconcile profiles declared by a scenario"}),
		support.SubcommandGroup("role-policy", "Inspect portable role policy", deps.Policy,
			[2]string{"status", "Show activation state and latest diagnostic"}, [2]string{"catalog", "Inspect the active portable role catalog"}, [2]string{"validate", "Validate declared state without activation"}, [2]string{"reload", "Validate and atomically activate declared state"}, [2]string{"explain", "Explain profile or run role resolution"}),
		support.SubcommandGroup("settings", "Manage runtime settings", deps.Settings,
			[2]string{"investigation", "Get, update, or reset investigation settings"}, [2]string{"orchestration", "Get, update, or reset orchestration settings"}),
		support.SubcommandGroup("permission-policy", "Manage portable permission policy", deps.PermissionPolicy,
			[2]string{"status", "Show activation and reconcile state"}, [2]string{"catalog", "Inspect the active desired-permissions catalog"}, [2]string{"validate", "Validate declared state without activation"}, [2]string{"reload", "Validate and atomically activate declared state"}, [2]string{"plan", "Report projection drift without mutation"}, [2]string{"reconcile", "Apply declared permissions through every resource"}, [2]string{"doctor", "Summarize readiness and enforcement coverage"}),
		support.SubcommandGroup("runner", "Manage agent runners", deps.Runner,
			[2]string{"list", "List all runners and their status"}, [2]string{"probe", "Probe a runner"}, [2]string{"tools", "List canonical tool enforcement mappings"}),
		support.SubcommandGroup("declarations", "Manage unified scenario declarations", deps.Declarations,
			[2]string{"reconcile-scenario", "Reconcile a scenario's profiles and workflows"}, [2]string{"plan", "Validate declaration sources without writing"}),
		support.SubcommandGroup("workflow", "Validate and execute declared workflows", deps.Workflow,
			[2]string{"validate", "Validate and canonicalize a workflow file"}, [2]string{"plan", "Validate scenario workflow sources without writes"}, [2]string{"reconcile-scenario", "Reconcile scenario workflow sources"}, [2]string{"reload", "Reload scenario workflow sources"}, [2]string{"list", "List workflow revisions"}, [2]string{"get", "Get a workflow revision"}, [2]string{"explain", "Explain the active workflow revision"}, [2]string{"simulate", "Simulate a workflow execution plan"}, [2]string{"start", "Start a workflow execution"}, [2]string{"execution-list", "List workflow executions"}, [2]string{"execution-runs", "List execution node attempts and runs"}, [2]string{"execution-get", "Get a workflow execution"}, [2]string{"execution-result", "Get execution input and output"}, [2]string{"execution-advance", "Advance a workflow execution"}, [2]string{"execution-wait", "Wait for a terminal execution"}, [2]string{"trace", "Show an execution journal"}, [2]string{"signal", "Signal a waiting execution"}, [2]string{"cancel", "Cancel a workflow execution"}, [2]string{"retry", "Retry a workflow execution"}, [2]string{"resume", "Resume a workflow execution"}),
		support.SubcommandGroup("task", "Manage tasks", deps.Task,
			[2]string{"list", "List all tasks"}, [2]string{"get", "Get task details"}, [2]string{"create", "Create a task"}, [2]string{"update", "Update a task"}, [2]string{"delete", "Delete a cancelled task"}, [2]string{"cancel", "Cancel a queued or running task"}),
		support.SubcommandGroup("maintenance", "Run maintenance operations", deps.Maintenance, [2]string{"purge", "Delete matching profiles, tasks, or runs"}),
		support.SubcommandGroup("ops", "Inspect typed-event operational statistics", deps.Ops,
			[2]string{"summary", "Show every operational category"}, [2]string{"fallback", "Show runner and model fallback insights"}, [2]string{"health", "Show engine-derived health transitions"}, [2]string{"sandbox", "Show sandbox operation outcomes"}, [2]string{"heartbeat", "Show heartbeat-miss counters"}, [2]string{"checkpoint", "Show checkpoint-failure counters"}, [2]string{"retry", "Show retry-attempt counters"}),
		support.SubcommandGroup("health", "Inspect persisted health snapshots and audit", deps.Health,
			[2]string{"models", "Show current model health"}, [2]string{"runners", "Show current runner health"}, [2]string{"audit", "Show paginated health history"}),
		support.SubcommandGroup("events", "Query the typed operational event log", deps.Events, [2]string{"list", "List typed events with optional filters"}),
		support.SubcommandGroup("findings", "Inspect recurring investigation findings", deps.Findings, [2]string{"list", "List recurring findings"}),
		support.SubcommandGroup("subscription", "Manage subscription billing periods", deps.Subscription, [2]string{"periods", "Create, list, or remove billing periods"}),
		space,
		runs.SubcommandGroup(deps),
		measures.Register(),
	}
}
