// Package cliutil — agent caller detection.
//
// Two detectors are exported, with deliberately different precision:
//
//   - IsAgentControlledContext (strict). Matches only Vrooli-spawned agent
//     contexts (sandbox, agent-manager, swarm-manager, identity-token). This
//     is the high-confidence detector used where false positives would break
//     things (API base URL resolution, path canonicalization).
//
//   - IsLikelyAgentContext (broad). Matches Vrooli-spawned AND known
//     external agent runtimes (Claude Code, Codex, opencode, …). Use this
//     when you want a friction gate for "the caller is probably an agent,
//     not a human" — e.g. git-control-tower's policy gate. False positives
//     are recoverable here (VROOLI_CALLER=human override).
//
// The list of known external agent runtime signals lives below in
// knownAgentSignals. ADDING A NEW RUNTIME requires that the signal env var
// be set by the runtime ITSELF on the shell it spawns as a child — not by
// a Vrooli wrapper, not by a web-console PTY injector, not by an npm shim
// that wraps the binary. Verify by diff'ing /proc/$$/environ inside a
// tool-shell against /proc/$PPID/environ of the runtime; a signal valid
// for detection appears in the former and is absent from the latter.
// See `docs/reference/agent-detection-signals.md` for the source-of-truth
// catalog and how each signal was confirmed.
package cliutil

import (
	"os"
	"strconv"
	"strings"
)

// CallerKind classifies who initiated the current CLI/RPC call.
type CallerKind int

const (
	// CallerKindUnknown means no signal was observed and no override is
	// active. Callers should treat this as "human-like" for policy
	// decisions but not as positive proof of human caller (e.g. a CI
	// shell with no TTY also lands here).
	CallerKindUnknown CallerKind = iota
	// CallerKindHuman means VROOLI_CALLER=human is set. Explicit
	// human-override, overrides every other signal.
	CallerKindHuman
	// CallerKindVrooliAgent means a Vrooli-spawned agent context: sandbox,
	// agent-manager run, swarm-manager session, or identity-token bearer.
	CallerKindVrooliAgent
	// CallerKindExternalAgent means a non-Vrooli agent runtime (Claude
	// Code, Codex, opencode, …) is invoking us through its tool surface.
	CallerKindExternalAgent
	// CallerKindOverride means VROOLI_CALLER=agent is set. Forces agent
	// classification regardless of env signals (used for test harnesses
	// and intentional agent contexts that lack the usual env vars).
	CallerKindOverride
)

// String returns the lowercase canonical name for logs and headers.
func (k CallerKind) String() string {
	switch k {
	case CallerKindHuman:
		return "human"
	case CallerKindVrooliAgent:
		return "vrooli-agent"
	case CallerKindExternalAgent:
		return "external-agent"
	case CallerKindOverride:
		return "override-agent"
	default:
		return "unknown"
	}
}

// agentContextEnvKeys is preserved verbatim for IsAgentControlledContext —
// adding to this list changes the strict detector's semantics and could
// break callers that depend on its narrow definition.
var agentContextEnvKeys = []string{
	"VROOLI_SANDBOX_ID",
	"VROOLI_SANDBOX_MERGED",
	"VROOLI_AGENT_MANAGER_RUN_ID",
	"VROOLI_SWARM_MANAGER_SESSION_ID",
	"VROOLI_AGENT_IDENTITY_TOKEN",
}

// IsAgentControlledContext reports whether the current process is running
// inside a Vrooli-spawned agent context (strict). Unchanged since
// introduction — see package doc for the rationale.
func IsAgentControlledContext() bool {
	for _, key := range agentContextEnvKeys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

// agentSignal pairs a friendly name with a match predicate. Each predicate
// captures the rule for one runtime — most are simple env-var presence
// checks, but opencode requires a PID-match (see opencodePIDMatchesAncestor).
type agentSignal struct {
	Name  string
	Kind  CallerKind
	Match func() bool
}

// knownAgentSignals is the source of truth for the broad detector. Order
// matters: Vrooli-spawned signals are checked first so a sandbox running
// Claude Code classifies as Vrooli (the larger blast radius).
//
// Each external-agent row corresponds to a row in the matching catalog at
// `packages/cli-core/docs/reference/agent-detection-signals.md`.
var knownAgentSignals = []agentSignal{
	// Vrooli-spawned (mirrors agentContextEnvKeys / IsAgentControlledContext).
	{Name: "vrooli-sandbox", Kind: CallerKindVrooliAgent, Match: envAnyNonEmpty("VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED")},
	{Name: "vrooli-agent-manager", Kind: CallerKindVrooliAgent, Match: envAnyNonEmpty("VROOLI_AGENT_MANAGER_RUN_ID", "VROOLI_AGENT_IDENTITY_TOKEN")},
	{Name: "vrooli-swarm-manager", Kind: CallerKindVrooliAgent, Match: envAnyNonEmpty("VROOLI_SWARM_MANAGER_SESSION_ID")},

	// External agent runtimes — confirmed via self-vs-parent env diff.
	// See docs/reference/agent-detection-signals.md for evidence.
	{Name: "claude-code", Kind: CallerKindExternalAgent, Match: envEquals("CLAUDECODE", "1")},
	{Name: "codex", Kind: CallerKindExternalAgent, Match: envAllNonEmpty("CODEX_CI", "CODEX_THREAD_ID")},
	{Name: "opencode", Kind: CallerKindExternalAgent, Match: opencodePIDMatchesAncestor},
}

// DetectCallerKind returns the typed caller classification. Precedence:
//
//  1. VROOLI_CALLER=human  → CallerKindHuman
//  2. VROOLI_CALLER=agent  → CallerKindOverride
//  3. First matching row in knownAgentSignals
//  4. Otherwise → CallerKindUnknown
//
// TTY presence/absence is intentionally NOT a signal — a CI shell has no
// TTY and would misclassify as agent under that rule. Callers that want a
// human-vs-unknown distinction can layer their own TTY check on top, but
// only as a Human/Unknown tiebreaker, never as a positive agent signal.
func DetectCallerKind() CallerKind {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_CALLER"))) {
	case "human":
		return CallerKindHuman
	case "agent":
		return CallerKindOverride
	}
	for _, sig := range knownAgentSignals {
		if sig.Match() {
			return sig.Kind
		}
	}
	return CallerKindUnknown
}

// IsLikelyAgentContext reports whether the caller is likely an agent: any
// Vrooli-spawned context, any known external agent runtime, or the
// VROOLI_CALLER=agent override. Returns false for explicit human override
// and for unknown callers.
func IsLikelyAgentContext() bool {
	k := DetectCallerKind()
	return k == CallerKindVrooliAgent || k == CallerKindExternalAgent || k == CallerKindOverride
}

// --- helpers --------------------------------------------------------------

func envEquals(key, value string) func() bool {
	return func() bool {
		return strings.TrimSpace(os.Getenv(key)) == value
	}
}

func envAnyNonEmpty(keys ...string) func() bool {
	return func() bool {
		for _, k := range keys {
			if strings.TrimSpace(os.Getenv(k)) != "" {
				return true
			}
		}
		return false
	}
}

func envAllNonEmpty(keys ...string) func() bool {
	return func() bool {
		for _, k := range keys {
			if strings.TrimSpace(os.Getenv(k)) == "" {
				return false
			}
		}
		return true
	}
}

// opencodePIDMatchesAncestor implements the anti-spoofing rule documented
// in agent-detection-signals.md for opencode: OPENCODE_PID must be a value
// that appears in the calling process's ancestor PID chain. This rules out
// false positives from a leaked OPENCODE_PID env var (e.g. an opencode
// wrapper that exited but left the value in the inherited environment of
// a later command).
//
// On non-Linux platforms /proc is not available — Vrooli is Linux-only per
// CLAUDE.md, so this returns false. The contract is "opencode-signal not
// confirmable" → no classification, never a false positive.
func opencodePIDMatchesAncestor() bool {
	raw := strings.TrimSpace(os.Getenv("OPENCODE_PID"))
	if raw == "" {
		return false
	}
	target, err := strconv.Atoi(raw)
	if err != nil || target <= 0 {
		return false
	}
	return ancestorChainContains(os.Getpid(), target, 64)
}

// ancestorChainContains walks /proc/<pid>/status PPid links upward from
// `start` looking for `target`. Bounded by maxHops to avoid pathological
// /proc states. Returns false on any read error (fail closed → no
// classification).
func ancestorChainContains(start, target, maxHops int) bool {
	pid := start
	for i := 0; i < maxHops; i++ {
		if pid == target {
			return true
		}
		if pid <= 1 {
			return false
		}
		ppid, ok := readPPid(pid)
		if !ok {
			return false
		}
		pid = ppid
	}
	return false
}

// readPPid parses /proc/<pid>/status for the PPid: line. Returns (0,
// false) on any error (file missing, malformed, etc.).
func readPPid(pid int) (int, bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PPid:") {
			ppid, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "PPid:")))
			if err != nil {
				return 0, false
			}
			return ppid, true
		}
	}
	return 0, false
}
