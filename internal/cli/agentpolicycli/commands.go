// Package agentpolicycli defines the top-level `vrooli agent-policy`
// command surface. It fans permission verbs out to every installed
// coding-agent resource CLI so one invocation can deny/allow/ask a bash
// pattern across the platform catalog.
package agentpolicycli

import (
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/codingagents"
)

// CommandID is the verb identifier inside the agent-policy subtree.
type CommandID string

const (
	CommandList   CommandID = "list"
	CommandDeny   CommandID = "deny"
	CommandAllow  CommandID = "allow"
	CommandAsk    CommandID = "ask"
	CommandRemove CommandID = "remove"
	CommandReset  CommandID = "reset"
	CommandDoctor CommandID = "doctor"
)

// OverrideFlag mirrors agentpolicy.OverrideFlag — the shared
// `--i-was-explicitly-authorized` spelling across the platform.
const OverrideFlag = "--i-was-explicitly-authorized"

// SupportedAgents lists the resource CLI binaries the fan-out targets. Each
// entry must:
//
//   - be installed on $PATH (the handler reports "not-installed" for
//     missing entries rather than failing the whole call), and
//   - expose `permissions <verb>` with the canonical CLI surface
//     produced by Phases 1–3 of the agent-permissions plan.
//
// Adding a new coding-agent resource to the platform = add it to
// internal/codingagents.Catalog.
var SupportedAgents = codingagents.ResourceCLIs()

func CommandSpecs() []commandtree.Spec[CommandID] {
	mutatingArgs := commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{
			{Name: "pattern", Description: "Bash pattern using each agent's native syntax (passed through verbatim)", Required: true},
		},
		Options: []commandtree.OptionArg{
			{Name: OverrideFlag, Description: "Override the agent-vs-human gate (only with explicit human authorization)"},
		},
	}
	resetArgs := commandtree.ArgSchema{
		Options: []commandtree.OptionArg{
			{Name: OverrideFlag, Description: "Override the agent-vs-human gate (only with explicit human authorization)"},
		},
	}
	readArgs := commandtree.ArgSchema{
		Options: []commandtree.OptionArg{commandtree.JSONOption()},
	}
	return []commandtree.Spec[CommandID]{
		{Name: string(CommandList), Group: "Agent Policy", Summary: "List managed permission patterns on every installed coding-agent resource", Handler: CommandList, Args: readArgs},
		{Name: string(CommandDeny), Group: "Agent Policy", Summary: "Add a bash deny pattern on every installed coding-agent resource", Handler: CommandDeny, Args: mutatingArgs},
		{Name: string(CommandAllow), Group: "Agent Policy", Summary: "Add a bash allow pattern on every installed coding-agent resource", Handler: CommandAllow, Args: mutatingArgs},
		{Name: string(CommandAsk), Group: "Agent Policy", Summary: "Add a bash ask pattern on every installed coding-agent resource", Handler: CommandAsk, Args: mutatingArgs},
		{Name: string(CommandRemove), Group: "Agent Policy", Summary: "Remove a bash pattern on every installed coding-agent resource", Handler: CommandRemove, Args: mutatingArgs},
		{Name: string(CommandReset), Group: "Agent Policy", Summary: "Clear Vrooli-managed entries on every installed coding-agent resource", Handler: CommandReset, Args: resetArgs},
		{Name: string(CommandDoctor), Group: "Agent Policy", Summary: "Run `permissions doctor` on every installed coding-agent resource", Handler: CommandDoctor, Args: readArgs},
	}
}

func RenderCommandHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandtree.RenderHelpText(commandtree.Help{
		Title:        "vrooli agent-policy - Manage permissions on every installed coding-agent resource",
		Usage:        "vrooli agent-policy <verb> [options]",
		Description:  "Fans permission verbs out to " + strings.Join(SupportedAgents, ", ") + ". Resources not installed are reported as not-installed and skipped — they do not fail the call. Mutating verbs refuse agent callers unless --i-was-explicitly-authorized is passed.",
		DefaultGroup: "Agent Policy",
		Notes:        []string{"Pattern syntax is each agent's native form (e.g. 'Bash(git stash*)' for Claude, 'git stash *' for OpenCode); the fan-out does not translate."},
	}, CommandSpecs()))
}
