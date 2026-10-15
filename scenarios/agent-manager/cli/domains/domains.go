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
	subscription := support.SubcommandGroup("subscription", "Manage subscription billing periods and quota observations", deps.Subscription,
		[2]string{"periods", "Create, list, or remove billing periods"}, [2]string{"quota", "List provider quota observations"})
	for i := range subscription.Subcommands {
		if subscription.Subcommands[i].Name == "quota" {
			subscription.Subcommands[i].Usage = "agent-manager subscription quota list [--provider --pool --window --limit --json]"
			subscription.Subcommands[i].HelpText = "Reads provider-reported quota standing and percentage/window metadata with provenance; amounts remain unknown unless reported."
		}
	}
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
			[2]string{"list", "List all runners and their status"}, [2]string{"probe", "Probe a runner"}, [2]string{"tools", "List canonical tool enforcement mappings"}, [2]string{"execution-options", "List truthful runner, objective, model, and effort options"}),
		support.SubcommandGroup("declarations", "Manage unified scenario declarations", deps.Declarations,
			[2]string{"reconcile-scenario", "Reconcile a scenario's profiles and workflows"}, [2]string{"plan", "Validate declaration sources without writing"}),
		support.SubcommandGroup("workflow", "Validate and execute declared workflows", deps.Workflow,
			[2]string{"validate", "Validate and canonicalize a workflow file"}, [2]string{"plan", "Validate scenario workflow sources without writes"}, [2]string{"reconcile-scenario", "Reconcile scenario workflow sources"}, [2]string{"reload", "Reload scenario workflow sources"}, [2]string{"list", "List workflow revisions"}, [2]string{"get", "Get a workflow revision"}, [2]string{"explain", "Explain the active workflow revision"}, [2]string{"simulate", "Simulate a workflow execution plan"}, [2]string{"start", "Start a workflow execution"}, [2]string{"execution-list", "List workflow executions"}, [2]string{"execution-runs", "List execution node attempts and runs"}, [2]string{"execution-get", "Get a workflow execution"}, [2]string{"execution-result", "Get execution input and output"}, [2]string{"execution-advance", "Advance a workflow execution"}, [2]string{"execution-wait", "Wait for a terminal execution"}, [2]string{"trace", "Show an execution journal"}, [2]string{"signal", "Signal a waiting execution"}, [2]string{"cancel", "Cancel a workflow execution"}, [2]string{"retry", "Retry a workflow execution"}, [2]string{"resume", "Resume a workflow execution"}),
		effortGroup(deps),
		support.SubcommandGroup("watch", "Supervise a durable family execution cohort", deps.Watch,
			[2]string{"create", "Create a durable cohort watch"}, [2]string{"get", "Get a cohort watch"}, [2]string{"list", "List cohort watches"}, [2]string{"wait", "Wait for a watch revision"}, [2]string{"cancel", "Cancel an active watch"}, [2]string{"inspect", "Inspect bounded pending events"}, [2]string{"action", "Request a typed intervention"}, [2]string{"actions", "List durable intervention history"}, [2]string{"policy-get", "Get the durable supervision policy"}, [2]string{"policy-outcomes", "List bounded durable policy outcomes"},
			[2]string{"policy-candidate", "Create an immutable candidate policy"},
			[2]string{"policy-evaluate", "Replay and evaluate candidate evidence"},
			[2]string{"policy-assess", "Assess a durable supervision outcome"},
			[2]string{"policy-promote", "Promote a policy after measured gates"},
			[2]string{"policy-reject", "Reject a candidate policy"},
			[2]string{"policy-rollback", "Roll back supervision policy"},
			[2]string{"policy-disable", "Disable supervision policy"}),
		support.SubcommandGroup("investigation", "Run bounded caller-neutral investigations", deps.Investigation,
			[2]string{"start", "Start or reattach a typed investigation"}, [2]string{"get", "Get a typed investigation"}, [2]string{"list", "List typed investigations"}, [2]string{"wait", "Wait for a typed investigation"}, [2]string{"cancel", "Cancel a typed investigation"}),
		support.SubcommandGroup("task", "Manage tasks", deps.Task,
			[2]string{"list", "List all tasks"}, [2]string{"get", "Get task details"}, [2]string{"create", "Create a task"}, [2]string{"update", "Update a task"}, [2]string{"delete", "Delete a cancelled task"}, [2]string{"cancel", "Cancel a queued or running task"}),
		maintenanceGroup(deps),
		storageGroup(deps),
		support.SubcommandGroup("standing-grants", "Create, list, or revoke operator-approved standing grants", deps.StandingGrants,
			[2]string{"create", "Create a bounded standing grant for a service subject (human-only; use --local-owner)"},
			[2]string{"list", "List recorded standing grants (human-only; use --local-owner)"},
			[2]string{"revoke", "Revoke a standing grant by id (human-only; use --local-owner)"}),
		support.SubcommandGroup("ops", "Inspect typed-event operational statistics", deps.Ops,
			[2]string{"summary", "Show every operational category"}, [2]string{"fallback", "Show runner and model fallback insights"}, [2]string{"health", "Show engine-derived health transitions"}, [2]string{"sandbox", "Show sandbox operation outcomes"}, [2]string{"heartbeat", "Show heartbeat-miss counters"}, [2]string{"checkpoint", "Show checkpoint-failure counters"}, [2]string{"retry", "Show retry-attempt counters"}),
		support.SubcommandGroup("health", "Inspect persisted health snapshots and audit", deps.Health,
			[2]string{"models", "Show current model health"}, [2]string{"runners", "Show current runner health"}, [2]string{"audit", "Show paginated health history"}),
		support.SubcommandGroup("events", "Query the typed operational event log", deps.Events, [2]string{"list", "List typed events with optional filters"}),
		support.SubcommandGroup("findings", "Inspect recurring investigation findings", deps.Findings, [2]string{"list", "List recurring findings"}),
		subscription,
		conversationGroup(deps),
		space,
		runs.SubcommandGroup(deps),
		measures.Register(),
	}
}

func maintenanceGroup(deps support.Dependencies) cliapp.SubcommandGroup {
	group := support.SubcommandGroup("maintenance", "Run maintenance operations", deps.Maintenance,
		[2]string{"status", "Inspect durable admission and admitted-work inventory"},
		[2]string{"begin", "Close admission without cancelling admitted work"},
		[2]string{"drain", "Wait for admitted work; timeout leaves the fence closed"},
		[2]string{"resume", "Reopen admission using the exact maintenance revision"},
		[2]string{"purge", "Delete matching profiles, tasks, or runs"})
	for i := range group.Subcommands {
		command := &group.Subcommands[i]
		if command.Name == "purge" {
			continue
		}
		command.Usage = "agent-manager maintenance " + command.Name + " [--json]"
		command.HelpText = "Read the bounded owner projection. HTTP errors retain observed fence and inventory evidence; unknown is not empty proof."
		command.Args.Flags = []cliapp.Flag{{Name: "json", Bool: true, LocalOnly: true, Description: "Print the owner projection"}}
		if command.Name != "status" {
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "local-owner", Bool: true, LocalOnly: true, Description: "Explicit local owner exchange; unavailable inside identified agent runs"})
			command.HelpText = "Use canonical configured owner authentication. Optional --local-owner explicitly exchanges local human authority and is unavailable inside identified agent runs. Timeout or unknown evidence preserves the fence and admitted work."
		}
		switch command.Name {
		case "begin":
			command.Usage = "agent-manager maintenance begin --reason <text> [--local-owner] [--json]"
			command.HelpText += " Required --reason: nonblank, 1-512 bytes."
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "reason", Required: true, Description: "Required nonblank reason, 1-512 bytes"})
		case "drain":
			command.Usage = "agent-manager maintenance drain [--timeout 1m] [--local-owner] [--json]"
			command.HelpText += " --timeout accepts whole seconds from 1s to 120s (default 1m). It waits; it never cancels work."
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "timeout", Default: "1m", Description: "Whole seconds from 1s to 120s"})
		case "resume":
			command.Usage = "agent-manager maintenance resume --revision <closed-revision> [--local-owner] [--json]"
			command.HelpText += " Required --revision: the exact closed revision returned by begin or status. Lifecycle lock contention refuses resume."
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "revision", Required: true, Description: "Exact maintenance revision"})
		}
	}
	return group
}

func effortGroup(deps support.Dependencies) cliapp.SubcommandGroup {
	group := support.SubcommandGroup("effort", "Observe and supervise arbitrary efforts", deps.Effort,
		[2]string{"board", "Read the shared effort board"}, [2]string{"compact", "Read a compact joined owner observation"}, [2]string{"list", "List durable effort enrollments"},
		[2]string{"discover", "Reconcile bounded discovery (operator authentication)"},
		[2]string{"enroll", "Enroll or amend with operator authority"}, [2]string{"reconcile-metadata", "Reconcile owner-facing metadata with scoped authority"}, [2]string{"withdraw", "Withdraw with revision fencing"},
		[2]string{"epoch-check", "Report an epoch file's step-back triggers and gates (exit 3 on STEP_BACK, 4 when --acceptance is refused)"},
		[2]string{"lint", "Check a goal home's orchestrator rules (exit 4 on a failing rule)"},
		[2]string{"handoff", "Replace the single ## Handoff of a goal home's QUEUE.md"},
		[2]string{"park", "Park the orchestrator for as long as goal state allows (exit 4 when refused)"},
		[2]string{"signals", "Read an effort's signal log after a cursor (check-in dispatch input)"},
		[2]string{"signal", "Record a feedback or step_back signal for an effort"},
		[2]string{"decision", "Raise, answer, list or sync operator decisions of a goal home"},
		[2]string{"exact-setup", "Stage, inspect, activate or revoke protected native exact authority"},
		[2]string{"exact-reconcile", "Submit one exact metadata mutation with existing native identity"},
		[2]string{"exact-receipt", "Read an immutable receipt under its original standing grant"})
	group.DefaultSubcommand = "board"
	types := map[string]string{"enroll": "EnrollEffortRequest", "reconcile-metadata": "ReconcileEffortMetadataRequest", "withdraw": "WithdrawEffortRequest"}
	offline := false
	for i := range group.Subcommands {
		command := &group.Subcommands[i]
		if command.Name == "exact-setup" || command.Name == "exact-reconcile" || command.Name == "exact-receipt" {
			name := command.Name
			command.Run = nil
			command.RunCtx = func(ctx cliapp.RunContext) error {
				args := []string{name}
				if name == "exact-setup" {
					args = append(args, ctx.Positional("action"))
				}
				for _, flag := range []string{"request-file", "grant"} {
					if value := ctx.Flag(flag); value != "" {
						args = append(args, "--"+flag, value)
					}
				}
				if name == "exact-setup" && ctx.BoolFlag("local-owner") {
					args = append(args, "--local-owner")
				}
				return deps.Effort(args)
			}
		}
		if command.Name == "exact-setup" {
			command.Usage = "agent-manager effort exact-setup <stage|activate|revoke|status> [--request-file public.json | --grant id] [--local-owner]"
			command.Args.Positionals = []cliapp.Positional{{Name: "action", Required: true, Description: "stage, activate, revoke or status"}}
			command.Args.Flags = []cliapp.Flag{{Name: "request-file", LocalOnly: true, Description: "Typed public request for stage, activate or revoke"}, {Name: "grant", LocalOnly: true, Description: "Original grant ID for status"}, {Name: "local-owner", Bool: true, LocalOnly: true, Description: "Explicit local human exchange; unavailable inside identified agent runs"}}
			command.HelpText = "Requires independently verified operator authority. Stage is inactive; activate additionally requires the Site key-possession proof. Status recovers the original approval without changing identity. Optional --local-owner exchanges the existing local binding for this request only, after input validation; refusal is terminal. Setup requests refuse redirects, which leave the original mutation outcome unconfirmed."
			continue
		}
		if command.Name == "exact-reconcile" || command.Name == "exact-receipt" {
			command.Usage = "agent-manager effort " + command.Name + " --request-file request.json --grant id"
			command.Args.Flags = []cliapp.Flag{{Name: "request-file", Required: true, LocalOnly: true, Description: "Bounded typed request; receipt reads retain original executor and payload"}, {Name: "grant", Required: true, LocalOnly: true, Description: "Original owner-installed standing grant ID"}}
			command.HelpText = "Keeps this process alive between public challenge output and bounded signed proof on stdin. Existing native identity is pinned and rechecked; a broker refusal or credential change prevents submission."
			continue
		}
		command.Args.Flags = []cliapp.Flag{{Name: "json", Bool: true, LocalOnly: true, Description: "Print the typed RPC response as JSON"}}
		command.Usage = "agent-manager effort " + command.Name + " [--json]"
		if command.Name == "compact" {
			command.HelpText = "Text output renders the compact joined owner observation; --json returns the full typed EffortBoard response for machine consumers."
		}
		if goalHomeCommand(command) {
			if command.Name == "lint" || command.Name == "handoff" {
				command.NeedsAPIOverride = &offline
			}
			continue
		}
		if types[command.Name] != "" || command.Name == "discover" {
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "local-owner", Bool: true, LocalOnly: true, Description: "Explicit local operator exchange; unavailable in identified agent runs"})
		}
		if message := types[command.Name]; message != "" {
			command.Usage = "agent-manager effort " + command.Name + " --request-file request.json [--json] [--local-owner]"
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "request-file", Required: true, LocalOnly: true, Description: "Proto JSON agent_manager.v1." + message})
			command.HelpText = "Request schema: packages/proto/schemas/agent-manager/v1/domain/effort.proto (" + message + "). Examples and authority: scenarios/agent-manager/docs/reference/effort-supervision.md. Authentication uses the configured owner token or an explicitly scoped orchestrator run; request text cannot grant permission."
		} else if command.Name != "discover" {
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "page-size", Default: "50", Description: "Maximum rows, 1-100"}, cliapp.Flag{Name: "page-token", Description: "Next page token from prior response"})
			if command.Name != "list" {
				command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "effort-ref", Description: "Exact stable effort reference"})
			}
			command.Usage += " [--effort-ref ref] [--page-size 50] [--page-token token]"
		}
	}
	return group
}

// goalHomeCommand documents the effort commands that read or write a goal home
// (large-effort-orchestration §3) and reports whether command is one. Their
// flags are parsed by the local handler; these declarations drive help.
func goalHomeCommand(command *cliapp.Command) bool {
	const reference = " Grammar and examples: scenarios/agent-manager/docs/reference/effort-supervision.md#goal-home-commands."
	jsonFlag := cliapp.Flag{Name: "json", Bool: true, LocalOnly: true, Description: "Print the report as JSON"}
	switch command.Name {
	case "epoch-check":
		command.Usage = "agent-manager effort epoch-check <epoch-file> [--acceptance] [--wake-key <run-id>] [--effort <ref>] [--runs ids] [--spend-threshold n] [--json]"
		command.HelpText = "Prints step-back triggers, typed exit gates (G<n> test|journey|inventory|review|custom) as met or unmet, the measured yield and goal-level findings. Exits 3 on STEP_BACK and wakes --wake-key. An epoch with an ACCEPTED line never exits 3 or wakes. --acceptance is the orchestrator's check: exit 4 on an unmet gate or an unanswered diminishing-returns finding; it never exits 3 or wakes." + reference
		command.Args.Flags = []cliapp.Flag{jsonFlag, {Name: "acceptance", Bool: true, LocalOnly: true, Description: "Orchestrator acceptance check (exit 4 when refused)"}, {Name: "wake-key", LocalOnly: true, Description: "Orchestrator run ID to wake on STEP_BACK"}, {Name: "effort", LocalOnly: true, Description: "Effort reference whose signal log records a step_back on STEP_BACK"}, {Name: "runs", LocalOnly: true, Description: "Worker run IDs for the spend trigger"}, {Name: "spend-threshold", LocalOnly: true, Description: "Weighted non-cache tokens above which spend fires"}}
	case "lint":
		command.Usage = "agent-manager effort lint <goal home> [--json]"
		command.HelpText = "Fails (exit 4), naming each rule, when the resume set (GOAL.md, QUEUE.md, open FEEDBACK, open WORKAROUNDS) exceeds 30,000 bytes, QUEUE.md does not have exactly one handoff or it exceeds 4 KB, no ## Next item is marked [ready] and ## Needs operator is empty, open operator or supervisor feedback is not named in the handoff, or diminishing returns has no [re-aim] item under ## Needs operator. Reads files only." + reference
		command.Args.Flags = []cliapp.Flag{jsonFlag}
	case "handoff":
		command.Usage = "agent-manager effort handoff set <goal home> (--file <markdown> | --stdin) [--json]"
		command.HelpText = "Atomically replaces the single ## Handoff section of QUEUE.md (creating it after ## Needs operator when missing). Writes nothing when the text is unchanged. Refuses (exit 4) text over 4 KB, a queue with more than one handoff, and a QUEUE.md that changed during the write. Reads and writes files only." + reference
		command.Args.Flags = []cliapp.Flag{jsonFlag, {Name: "file", LocalOnly: true, Description: "Markdown file holding the handoff text"}, {Name: "stdin", Bool: true, LocalOnly: true, Description: "Read the handoff text from standard input"}}
	case "park":
		command.Usage = "agent-manager effort park <goal home> --run <run-id> [--timeout D] [--json]"
		command.HelpText = "Parks the orchestrator (producer children, keyed by its run) for up to 1h while a direct child run is live, or up to 72h while ## Needs operator holds an item; otherwise refuses (exit 4) with the admissible-slice rule. --timeout may only shorten the maximum." + reference
		command.Args.Flags = []cliapp.Flag{jsonFlag, {Name: "run", LocalOnly: true, Description: "The calling orchestrator's run ID"}, {Name: "timeout", LocalOnly: true, Description: "Park length, at most the rule's maximum"}, {Name: "identity-token", LocalOnly: true, Description: "Owning run's identity token (defaults to the run environment)"}}
	case "signals":
		command.Usage = "agent-manager effort signals <effort-ref> [--after N] [--limit 100] [--json]"
		command.HelpText = "Lists the effort's durable signals with seq greater than --after, in seq order, and the latest seq. Kinds: child_run_ended (a linked non-orchestrator run ended), step_back (epoch-check --effort), feedback (recorded by `effort signal --kind feedback` for a new operator or supervisor FEEDBACK entry), operator_answer (a decision was answered or defaulted), cap_stop (a run stopped at its weighted-token cap). One fact is one signal." + reference
		command.Args.Flags = []cliapp.Flag{{Name: "json", Bool: true, LocalOnly: true, Description: "Print the typed ListEffortSignalsResponse as JSON"}, {Name: "after", LocalOnly: true, Description: "Cursor: only signals with a greater seq"}, {Name: "limit", LocalOnly: true, Description: "Maximum signals, 1-500"}}
	case "signal":
		command.Usage = "agent-manager effort signal <effort-ref> --kind feedback|step_back [--detail text] [--run run-id] [--dedupe-key key] [--json]"
		command.HelpText = "Records one signal in the effort's log; the next orchestrator check-in starts on it. Whoever writes an operator or supervisor FEEDBACK.md entry records --kind feedback naming the entry. The same --dedupe-key (default kind:run:detail) records nothing new. Agent Manager derives child_run_ended, operator_answer and cap_stop itself." + reference
		command.Args.Flags = []cliapp.Flag{jsonFlag, {Name: "kind", LocalOnly: true, Description: "feedback or step_back"}, {Name: "detail", LocalOnly: true, Description: "What happened"}, {Name: "run", LocalOnly: true, Description: "Run of the effort the signal names"}, {Name: "dedupe-key", LocalOnly: true, Description: "Same key, same signal"}}
	case "decision":
		command.Usage = "agent-manager effort decision raise|answer|list|sync <goal home> [options]"
		command.HelpText = "raise <home> --id ID --question Q --option key=label (2-8) --recommend key [--why text] [--default key --reversible] [--deadline 24h] [--urgency high] [--dry-run]: stores one record per (goal home, id), publishes one decision_requested event (notification-hub asks the operator once) and adds the item under ## Needs operator. An identical re-raise changes nothing; a changed one is refused. --dry-run prints the record, event and item and writes nothing. answer <home> <id> <key> [--note text] [--local-owner]: the operator's desktop answer; agent runs are refused. list [<home>] [--effort-ref ref] [--unrecorded]. sync <home>: writes each answered or defaulted decision verbatim into FEEDBACK.md (\"applied by default\" for a default), removes its ## Needs operator item and marks it recorded; idempotent. --home-ref names the repository-relative goal home when the directory is a copy outside the repository." + reference
		command.Args.Flags = []cliapp.Flag{jsonFlag, {Name: "id", LocalOnly: true, Description: "Decision ID (raise)"}, {Name: "question", LocalOnly: true, Description: "One-sentence question (raise)"}, {Name: "option", LocalOnly: true, Description: "key=label, repeatable (raise)"}, {Name: "recommend", LocalOnly: true, Description: "Recommended option key (raise)"}, {Name: "why", LocalOnly: true, Description: "Why the recommendation (raise)"}, {Name: "default", LocalOnly: true, Description: "Option applied at the deadline (raise; needs --reversible)"}, {Name: "reversible", Bool: true, LocalOnly: true, Description: "The decision can be undone (raise)"}, {Name: "deadline", LocalOnly: true, Description: "Duration of at least 12h or RFC 3339 time (raise)"}, {Name: "urgency", LocalOnly: true, Description: "low, normal, high or critical (raise)"}, {Name: "note", LocalOnly: true, Description: "Context (raise) or the operator's note (answer)"}, {Name: "context-url", LocalOnly: true, Description: "Link for more context (raise)"}, {Name: "home-ref", LocalOnly: true, Description: "Repository-relative goal home"}, {Name: "effort-ref", LocalOnly: true, Description: "Effort reference (raise, list)"}, {Name: "unrecorded", Bool: true, LocalOnly: true, Description: "Only decisions not yet in FEEDBACK.md (list)"}, {Name: "dry-run", Bool: true, LocalOnly: true, Description: "Print what raise would do; write nothing (raise)"}, {Name: "local-owner", Bool: true, LocalOnly: true, Description: "Explicit local operator exchange (answer)"}}
	default:
		return false
	}
	return true
}

func conversationGroup(deps support.Dependencies) cliapp.SubcommandGroup {
	group := support.SubcommandGroup("conversation", "Search and inspect attributable conversation history", deps.Conversation,
		[2]string{"search", "Search conversations across runs"}, [2]string{"context", "Fetch bounded context for a stable search hit"}, [2]string{"index", "Inspect or rebuild the conversation search index"})
	group.Subcommands[0].Usage = "agent-manager conversation search [query] [--mode hybrid|text|regex|semantic] [--sort relevance|newest|oldest] [filters] [--page-size n] [--page-token token] [--json]"
	group.Subcommands[0].HelpText = "Filters: --roles, --harnesses, --provider-origins, --project-scopes, --cwd-scopes, --runners, --models, --profiles, --run-statuses, --tags, --workloads, --occurred-after, --occurred-before, --content-classes, --include-tool-events\n\nExamples:\n  agent-manager conversation search \"phase resource usage system monitor\" --mode hybrid --json\n  agent-manager conversation search '(?i)test.genie.*system.monitor' --mode regex --occurred-after 2026-08-01T00:00:00Z"
	group.Subcommands[1].Usage = "agent-manager conversation context <stable-hit-id> [--before n] [--after n] [--json]"
	group.Subcommands[1].HelpText = "Fetches only bounded neighboring events; it never returns an unbounded raw transcript."
	group.Subcommands[2].Usage = "agent-manager conversation index <status|reindex|cancel> [options]"
	group.Subcommands[2].HelpText = "Commands:\n  status [--json]\n  reindex [--dry-run] [--full=true] [--max-documents n] [--idempotency-key key] [--control-token token] [--json]\n  cancel <operation-id> [--control-token token] [--json]"
	group.Subcommands[2].DryRun = cliapp.DryRunCommandLocal
	group.Subcommands[2].DryRunAlternative = "agent-manager conversation index reindex --dry-run"
	return group
}
