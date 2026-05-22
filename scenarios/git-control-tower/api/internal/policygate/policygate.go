// Package policygate computes the agent-access gate decision for
// git-control-tower mutating commands.
//
// One shared implementation is consumed by:
//
//   - the CLI (cli/main.go) for primary friction at the point of intent;
//   - the API Connect interceptor (api/internal/handlers/interceptor.go)
//     for defense in depth on direct RPC calls.
//
// Decide() is a pure function — no I/O, no env reads — so it can be
// tested with a table-driven matrix. Callers wire it to env/header
// inputs at their layer.
package policygate

import (
	"strings"

	"git-control-tower/internal/config"

	"github.com/vrooli/cli-core/cliutil"
)

// Decision is the gate verdict for a single mutating command.
type Decision int

const (
	// DecisionAllow runs the command without friction.
	DecisionAllow Decision = iota
	// DecisionWarn runs the command but emits the agent-deny message.
	DecisionWarn
	// DecisionDeny refuses the command with the agent-deny message.
	DecisionDeny
)

// String returns a stable lowercase identifier for logs and audit
// events.
func (d Decision) String() string {
	switch d {
	case DecisionAllow:
		return "allow"
	case DecisionWarn:
		return "warn"
	case DecisionDeny:
		return "deny"
	default:
		return "unknown"
	}
}

// CommandSpec describes the mutating command being gated. Today this
// is informational — it flows into the audit log and the agent-deny
// message template. Future per-command policy granularity would expand
// this shape.
type CommandSpec struct {
	// Name is the human-readable command identifier
	// ("WorktreeService.CreateWorktree", "repo commit", etc.). Used in
	// audit logs and {command} template substitution.
	Name string
	// Effect is the manifest-declared effect ("write" or "destructive").
	// Read-only effects ("read") bypass the gate.
	Effect string
}

// CallerOverrideFlags carries the request-side authorization signals.
// On the CLI side this is set by the `--i-was-explicitly-authorized`
// flag; on the API side it's set by the `X-Vrooli-Authorized: true`
// header.
type CallerOverrideFlags struct {
	// AuthorizedByUser is true when the agent has been explicitly
	// authorized for this specific command by the user. Satisfies the
	// `confirm` policy.
	AuthorizedByUser bool
}

// Decide returns the gate decision for a single (caller, command,
// policy, override) tuple. Pure function: no I/O.
//
// Decision matrix:
//
//	caller                     │ allow  │ warn  │ confirm           │ deny
//	───────────────────────────┼────────┼───────┼───────────────────┼────────
//	human / unknown            │ Allow  │ Allow │ Allow             │ Allow
//	vrooli-agent / external    │ Allow  │ Warn  │ Allow if override │ Deny
//	override-agent             │ Allow  │ Warn  │ Allow if override │ Deny
//
// Non-mutating commands (Effect != "write" / "destructive") always
// Allow regardless of policy — they're read-only.
func Decide(kind cliutil.CallerKind, cmd CommandSpec, flags CallerOverrideFlags, policy config.PolicyConfig) Decision {
	if !isMutating(cmd.Effect) {
		return DecisionAllow
	}
	if !isAgentKind(kind) {
		return DecisionAllow
	}
	switch policy.AgentAccess {
	case config.AgentAccessAllow:
		return DecisionAllow
	case config.AgentAccessWarn:
		return DecisionWarn
	case config.AgentAccessDeny:
		return DecisionDeny
	case config.AgentAccessConfirm:
		if flags.AuthorizedByUser {
			return DecisionAllow
		}
		return DecisionDeny
	default:
		// Unknown policy → fail closed.
		return DecisionDeny
	}
}

// RenderDenyMessage produces the agent-deny message rendered with the
// template from policy. Empty template falls back to the hardcoded
// default. Supports {command} and {policy} placeholders.
func RenderDenyMessage(cmd CommandSpec, policy config.PolicyConfig) string {
	tmpl := policy.AgentDenyMessageTemplate
	if tmpl == "" {
		tmpl = defaultDenyTemplate
	}
	out := tmpl
	out = strings.ReplaceAll(out, "{command}", cmd.Name)
	out = strings.ReplaceAll(out, "{policy}", string(policy.AgentAccess))
	out = strings.ReplaceAll(out, "{override_flag}", policy.AgentOverrideFlag)
	return out
}

const defaultDenyTemplate = "git-control-tower command \"{command}\" was attempted by a detected agent caller, and the configured policy is \"{policy}\".\n\n" +
	"Git operations — including those routed through git-control-tower — are almost always a human's responsibility, not an agent's. Even when wrapped through GCT, commits, branch creation, resets, rebases, and merges should not be invoked by an agent unless the user has explicitly authorized this specific operation for this specific task. In the overwhelming majority of cases the correct action here is to STOP and leave the repository alone; let the human drive the git side.\n\n" +
	"If, and only if, you have explicit authorization from the user, re-run with the override flag ({override_flag}). Do NOT bypass this gate by aliasing, scripting around, or invoking git directly outside GCT — the workspace-sandbox guardrail will reject that path as well."

func isMutating(effect string) bool {
	switch strings.ToLower(strings.TrimSpace(effect)) {
	case "write", "destructive":
		return true
	default:
		return false
	}
}

func isAgentKind(kind cliutil.CallerKind) bool {
	switch kind {
	case cliutil.CallerKindVrooliAgent, cliutil.CallerKindExternalAgent, cliutil.CallerKindOverride:
		return true
	default:
		return false
	}
}
