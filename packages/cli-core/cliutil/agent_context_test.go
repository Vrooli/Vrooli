package cliutil

import (
	"os"
	"strconv"
	"testing"
)

// clearAllSignalEnv wipes every env var that knownAgentSignals or
// DetectCallerKind inspects so tests start from a clean baseline. Uses
// t.Setenv so the runtime restores the original values after the test.
func clearAllSignalEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"VROOLI_CALLER",
		"VROOLI_SANDBOX_ID",
		"VROOLI_SANDBOX_MERGED",
		"VROOLI_AGENT_MANAGER_RUN" + "_ID",
		"VROOLI_AGENT_IDENTITY_TOKEN",
		"VROOLI_SWARM_MANAGER_SESSION_ID",
		"CLAUDECODE",
		"CODEX_CI",
		"CODEX_THREAD_ID",
		"OPENCODE_PID",
		"GROK_AGENT",
		"ANTIGRAVITY_AGENT",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestIsAgentControlledContext(t *testing.T) {
	clearAllSignalEnv(t)
	if IsAgentControlledContext() {
		t.Fatal("expected false without any agent env")
	}
	t.Setenv("VROOLI_SANDBOX_ID", "sbx-1")
	if !IsAgentControlledContext() {
		t.Fatal("expected true with sandbox env")
	}
}

func TestIsAgentControlledContext_IgnoresExternalAgentSignals(t *testing.T) {
	// Strict detector must NOT classify external agents — that's the
	// whole point of the strict/broad split.
	clearAllSignalEnv(t)
	t.Setenv("CLAUDECODE", "1")
	if IsAgentControlledContext() {
		t.Fatal("strict detector should ignore CLAUDECODE")
	}
}

func TestDetectCallerKind_NoSignals(t *testing.T) {
	clearAllSignalEnv(t)
	if got := DetectCallerKind(); got != CallerKindUnknown {
		t.Fatalf("no signals: got %v, want CallerKindUnknown", got)
	}
}

func TestDetectCallerKind_HumanOverrideWins(t *testing.T) {
	clearAllSignalEnv(t)
	t.Setenv("VROOLI_SANDBOX_ID", "sbx-1") // would otherwise classify Vrooli
	t.Setenv("CLAUDECODE", "1")            // would otherwise classify external
	t.Setenv("VROOLI_CALLER", "human")
	if got := DetectCallerKind(); got != CallerKindHuman {
		t.Fatalf("VROOLI_CALLER=human did not override; got %v", got)
	}
	if IsLikelyAgentContext() {
		t.Fatal("IsLikelyAgentContext should be false under VROOLI_CALLER=human")
	}
}

func TestDetectCallerKind_AgentOverride(t *testing.T) {
	clearAllSignalEnv(t)
	t.Setenv("VROOLI_CALLER", "agent")
	if got := DetectCallerKind(); got != CallerKindOverride {
		t.Fatalf("VROOLI_CALLER=agent: got %v, want CallerKindOverride", got)
	}
	if !IsLikelyAgentContext() {
		t.Fatal("override should make IsLikelyAgentContext true")
	}
}

func TestDetectCallerKind_VrooliSignals(t *testing.T) {
	cases := []struct {
		name string
		env  string
	}{
		{"sandbox-id", "VROOLI_SANDBOX_ID"},
		{"sandbox-merged", "VROOLI_SANDBOX_MERGED"},
		{"identity-token", "VROOLI_AGENT_IDENTITY_TOKEN"},
		{"identity-token", "VROOLI_AGENT_IDENTITY_TOKEN"},
		{"swarm-manager-session", "VROOLI_SWARM_MANAGER_SESSION_ID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearAllSignalEnv(t)
			t.Setenv(tc.env, "x")
			if got := DetectCallerKind(); got != CallerKindVrooliAgent {
				t.Fatalf("%s: got %v, want CallerKindVrooliAgent", tc.env, got)
			}
			if !IsLikelyAgentContext() {
				t.Fatalf("%s: IsLikelyAgentContext should be true", tc.env)
			}
		})
	}
}

func TestDetectCallerKind_ClaudeCode(t *testing.T) {
	clearAllSignalEnv(t)
	t.Setenv("CLAUDECODE", "1")
	if got := DetectCallerKind(); got != CallerKindExternalAgent {
		t.Fatalf("CLAUDECODE=1: got %v, want CallerKindExternalAgent", got)
	}
}

func TestDetectCallerKind_ClaudeCode_OnlyOne(t *testing.T) {
	// CLAUDECODE=other-value must NOT match. The signal is presence of
	// the exact value "1" (which is what the runtime sets).
	clearAllSignalEnv(t)
	t.Setenv("CLAUDECODE", "true")
	if got := DetectCallerKind(); got == CallerKindExternalAgent {
		t.Fatal("CLAUDECODE=true should not classify as external agent")
	}
}

func TestDetectCallerKind_CodexRequiresBothSignals(t *testing.T) {
	// Codex requires CODEX_CI=1 AND CODEX_THREAD_ID set — either alone is
	// insufficient (CODEX_CI alone could be set by CI; CODEX_THREAD_ID
	// alone could be leaked from a stale env).
	clearAllSignalEnv(t)
	t.Setenv("CODEX_CI", "1")
	if got := DetectCallerKind(); got == CallerKindExternalAgent {
		t.Fatal("CODEX_CI alone should not classify as codex")
	}
	t.Setenv("CODEX_THREAD_ID", "abc-123")
	if got := DetectCallerKind(); got != CallerKindExternalAgent {
		t.Fatalf("CODEX_CI=1 + CODEX_THREAD_ID: got %v, want CallerKindExternalAgent", got)
	}
}

func TestDetectCallerKind_Grok(t *testing.T) {
	// grok injects GROK_AGENT=1 into the tool subprocesses it spawns
	// (verified via /proc env-diff 2026-06-28: present in grok tool shells,
	// absent from grok's own runtime env and every ancestor).
	clearAllSignalEnv(t)
	t.Setenv("GROK_AGENT", "1")
	if got := DetectCallerKind(); got != CallerKindExternalAgent {
		t.Fatalf("GROK_AGENT=1: got %v, want CallerKindExternalAgent", got)
	}
}

func TestDetectCallerKind_GrokCustomAgentValueDoesNotMatch(t *testing.T) {
	// GROK_AGENT doubles as a user-facing config var ("custom agent
	// definition path or name"). grok overwrites it to the literal "1" in
	// tool shells, so a non-"1" value means a human exported a custom-agent
	// name in their shell rc — that must NOT classify as an agent caller.
	clearAllSignalEnv(t)
	t.Setenv("GROK_AGENT", "my-custom-agent")
	if got := DetectCallerKind(); got == CallerKindExternalAgent {
		t.Fatal("GROK_AGENT=<custom-name> should not classify as grok agent")
	}
}

func TestDetectCallerKind_Antigravity(t *testing.T) {
	// Antigravity (Google's `agy` CLI) injects ANTIGRAVITY_AGENT=1 into the
	// shells it spawns for tool commands — the analog of grok's GROK_AGENT=1.
	// Binary-confirmed in agy 1.0.13's command-exec path (live /proc
	// confirmation pending). We match the exact sentinel value "1".
	clearAllSignalEnv(t)
	t.Setenv("ANTIGRAVITY_AGENT", "1")
	if got := DetectCallerKind(); got != CallerKindExternalAgent {
		t.Fatalf("ANTIGRAVITY_AGENT=1: got %v, want CallerKindExternalAgent", got)
	}
}

func TestDetectCallerKind_AntigravityNonSentinelValueDoesNotMatch(t *testing.T) {
	// Only the fixed sentinel "1" identifies an agy tool shell. A human who
	// exported ANTIGRAVITY_AGENT to any other value in their rc must NOT be
	// classified as an agent caller (mirrors the GROK_AGENT value-match rule).
	clearAllSignalEnv(t)
	t.Setenv("ANTIGRAVITY_AGENT", "custom")
	if got := DetectCallerKind(); got == CallerKindExternalAgent {
		t.Fatal("ANTIGRAVITY_AGENT=<non-1> should not classify as antigravity agent")
	}
}

func TestDetectCallerKind_OpencodeRequiresPIDMatch(t *testing.T) {
	// OPENCODE_PID alone with a non-ancestor PID must NOT classify —
	// that's the PID-match anti-spoofing rule.
	clearAllSignalEnv(t)
	// Pick a definitely-not-an-ancestor PID. Using a high number avoids
	// system PIDs that happen to be ancestors in some test runners.
	t.Setenv("OPENCODE_PID", "999999")
	if got := DetectCallerKind(); got == CallerKindExternalAgent {
		t.Fatal("OPENCODE_PID with non-ancestor value should not classify as opencode")
	}
}

func TestDetectCallerKind_OpencodeAncestorMatchClassifies(t *testing.T) {
	// Setting OPENCODE_PID to our own PID makes the ancestor chain
	// (starting at our own PID) contain it on the first iteration. That
	// satisfies the PID-match rule.
	clearAllSignalEnv(t)
	t.Setenv("OPENCODE_PID", strconv.Itoa(os.Getpid()))
	if got := DetectCallerKind(); got != CallerKindExternalAgent {
		t.Fatalf("OPENCODE_PID=$$ should classify as external; got %v", got)
	}
}

func TestDetectCallerKind_VrooliBeforeExternal(t *testing.T) {
	// A Vrooli-spawned sandbox running Claude Code must classify as
	// Vrooli, not external. The Vrooli signal has the larger blast
	// radius and policy consumers want to treat the Vrooli context as
	// authoritative.
	clearAllSignalEnv(t)
	t.Setenv("VROOLI_SANDBOX_ID", "sbx-1")
	t.Setenv("CLAUDECODE", "1")
	if got := DetectCallerKind(); got != CallerKindVrooliAgent {
		t.Fatalf("both signals set: got %v, want CallerKindVrooliAgent", got)
	}
}

func TestIsLikelyAgentContext_UnknownIsFalse(t *testing.T) {
	clearAllSignalEnv(t)
	if IsLikelyAgentContext() {
		t.Fatal("no signals: IsLikelyAgentContext should be false")
	}
}

func TestCallerKindString(t *testing.T) {
	cases := []struct {
		kind CallerKind
		want string
	}{
		{CallerKindUnknown, "unknown"},
		{CallerKindHuman, "human"},
		{CallerKindVrooliAgent, "vrooli-agent"},
		{CallerKindExternalAgent, "external-agent"},
		{CallerKindOverride, "override-agent"},
	}
	for _, tc := range cases {
		if got := tc.kind.String(); got != tc.want {
			t.Errorf("CallerKind(%d).String() = %q, want %q", tc.kind, got, tc.want)
		}
	}
}
