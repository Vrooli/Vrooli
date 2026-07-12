// Package agentpolicy is the shared agent-vs-human policy gate for
// Vrooli resource and scenario CLIs.
//
// Extracted from the duplicate-then-extract progression:
//
//  1. Phase 1: resources/claude-code/cli/internal/policygate
//  2. Phase 2: resources/opencode/cli/internal/policygate
//  3. Phase 3: resources/codex/cli/internal/policygate
//  4. Phase 4 (this package): three implementations had converged on a
//     single shape, so the canonical version moved here.
//
// Decide() is the only authoritative gate function. Callers (resource
// CLIs and scenario CLIs compose
// it with cliutil.DetectCallerKind to gate mutating verbs against
// agent callers.
//
// The decision matrix is operator-tunable via the AgentAccess knob on
// Policy. Default is AgentAccessDeny — stricter than the GCT default
// of `confirm` because permission verbs let agents disarm their own
// gate, which is uniquely dangerous.
//
// RenderDenyMessage is intentionally parameterised by a resource label
// and a config-file hint so the same shared package can produce
// recognisable, resource-specific deny messages without forcing every
// caller through a wrapper.
package agentpolicy

import (
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// AgentAccess is the policy applied to mutating verbs when the caller
// looks like an agent.
type AgentAccess string

const (
	AgentAccessAllow   AgentAccess = "allow"
	AgentAccessWarn    AgentAccess = "warn"
	AgentAccessConfirm AgentAccess = "confirm"
	AgentAccessDeny    AgentAccess = "deny"
)

// DefaultAgentAccess is the locked default for permissions verbs.
const DefaultAgentAccess = AgentAccessDeny

// OverrideFlag is the shared flag name across the Vrooli platform.
// Resource and scenario CLIs that consume this gate MUST use this exact
// spelling so operators learn one flag for the whole surface.
const OverrideFlag = "--i-was-explicitly-authorized"

// Decision is the gate verdict.
type Decision int

const (
	DecisionAllow Decision = iota
	DecisionWarn
	DecisionDeny
)

// String returns the lowercase canonical name for logs.
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

// CommandSpec describes the verb being gated.
type CommandSpec struct {
	// Name is the verb identifier ("permissions deny",
	// "permissions show", ...).
	Name string
	// Mutating is true for verbs that change on-disk state. Read
	// verbs (list/show/drift-check/doctor) set this false and bypass
	// the gate.
	Mutating bool
}

// CallerOverrideFlags carries authorization signals from the request.
type CallerOverrideFlags struct {
	// AuthorizedByUser is true when --i-was-explicitly-authorized was
	// passed.
	AuthorizedByUser bool
}

// Policy is the operator-tunable shape. Today the only knob is
// AgentAccess; the shape is reserved for future granularity.
type Policy struct {
	AgentAccess AgentAccess
}

// DefaultPolicy returns the locked defaults.
func DefaultPolicy() Policy {
	return Policy{AgentAccess: DefaultAgentAccess}
}

// Decide returns the gate verdict. Pure function — no I/O.
//
// Decision matrix:
//
//	caller                     │ allow  │ warn  │ confirm           │ deny
//	───────────────────────────┼────────┼───────┼───────────────────┼────────
//	human / unknown            │ Allow  │ Allow │ Allow             │ Allow
//	vrooli/external/override   │ Allow  │ Warn  │ Allow if override │ Deny
//
// Non-mutating verbs always Allow regardless of policy.
func Decide(kind cliutil.CallerKind, cmd CommandSpec, flags CallerOverrideFlags, policy Policy) Decision {
	if !cmd.Mutating {
		return DecisionAllow
	}
	if !isAgentKind(kind) {
		return DecisionAllow
	}
	switch policy.AgentAccess {
	case AgentAccessAllow:
		return DecisionAllow
	case AgentAccessWarn:
		return DecisionWarn
	case AgentAccessConfirm:
		if flags.AuthorizedByUser {
			return DecisionAllow
		}
		return DecisionDeny
	case AgentAccessDeny:
		if flags.AuthorizedByUser {
			return DecisionAllow
		}
		return DecisionDeny
	default:
		// Unknown policy → fail closed.
		return DecisionDeny
	}
}

// DenyContext is the resource-specific labelling that
// RenderDenyMessage interpolates into the otherwise-shared deny text.
type DenyContext struct {
	// ResourceLabel identifies the CLI being gated, e.g.
	// "resource-claude-code" or "scenario-foo-bar".
	ResourceLabel string
	// ConfigPath is a human-readable hint at the file under
	// management, e.g. "~/.claude/settings.json" or
	// "~/.config/opencode/opencode.json permission.bash". Used only
	// in the explanation text; the gate itself does not read it.
	ConfigPath string
}

// RenderDenyMessage produces the agent-deny explanation. Mirrors GCT's
// wording but is parametric over resource label + config path so the
// same shared text engine can serve all three coding-agent resources
// and any future scenario gate.
func RenderDenyMessage(dctx DenyContext, cmd CommandSpec, policy Policy) string {
	var b strings.Builder
	b.WriteString(dctx.ResourceLabel)
	b.WriteString(" command \"")
	b.WriteString(cmd.Name)
	b.WriteString("\" was attempted by a detected agent caller, and the configured policy is \"")
	b.WriteString(string(policy.AgentAccess))
	b.WriteString("\".\n\n")
	b.WriteString("Coding-agent permission files (")
	b.WriteString(dctx.ConfigPath)
	b.WriteString(") gate what bash commands the agent is allowed to run. An agent rewriting that gate is effectively disarming itself, which is almost never the right action.\n\n")
	b.WriteString("If, and only if, you have explicit authorization from the user for this specific change, re-run with ")
	b.WriteString(OverrideFlag)
	b.WriteString(". Do NOT edit ")
	b.WriteString(dctx.ConfigPath)
	b.WriteString(" directly to work around this gate.")
	return b.String()
}

func isAgentKind(kind cliutil.CallerKind) bool {
	switch kind {
	case cliutil.CallerKindVrooliAgent, cliutil.CallerKindExternalAgent, cliutil.CallerKindOverride:
		return true
	default:
		return false
	}
}
