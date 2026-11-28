package cliutil

import (
	"strings"
	"testing"
)

// clearCallerSignals isolates a test from the agent runtime that may be running it.
func clearCallerSignals(t *testing.T) {
	t.Helper()
	for _, name := range []string{EnvIdentityToken, AgentSessionEnv, "VROOLI_SWARM_MANAGER_SESSION_ID", "VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED", "CLAUDECODE", EnvClaudeCodeSessionID, "CODEX_CI", EnvCodexThreadID, "GROK_AGENT", "ANTIGRAVITY_AGENT", "ANTIGRAVITY_CONVERSATION_ID", "VROOLI_CALLER", "OPENCODE_PID"} {
		t.Setenv(name, "")
	}
}

func TestResolveCallerActorNamesTheAgentAndItsSession(t *testing.T) {
	clearCallerSignals(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv(EnvClaudeCodeSessionID, "abc")
	if got := ResolveCallerActor(""); got != "agent claude-code session abc" {
		t.Fatalf("actor = %q", got)
	}
}

func TestResolveCallerActorKeepsAnAgentsClaimAsAClaim(t *testing.T) {
	clearCallerSignals(t)
	t.Setenv(EnvCodexThreadID, "thread")
	got := ResolveCallerActor("op@example.com")
	if !strings.HasPrefix(got, "agent codex session thread") || !strings.Contains(got, "claiming to act for op@example.com") {
		t.Fatalf("actor = %q", got)
	}
}

func TestResolveCallerActorIsNotFooledByTheHumanOverride(t *testing.T) {
	clearCallerSignals(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("VROOLI_CALLER", "human")
	if got := ResolveCallerActor(""); !strings.HasPrefix(got, "agent claude-code") {
		t.Fatalf("actor = %q", got)
	}
}

func TestResolveCallerActorNeverWritesTheIdentityToken(t *testing.T) {
	clearCallerSignals(t)
	t.Setenv(EnvIdentityToken, "secret-token")
	t.Setenv(AgentSessionEnv, "run-1")
	if got := ResolveCallerActor(""); strings.Contains(got, "secret-token") || !strings.HasPrefix(got, "agent agent-manager run") {
		t.Fatalf("actor = %q", got)
	}
}

func TestResolveCallerActorWithoutAgentSignals(t *testing.T) {
	clearCallerSignals(t)
	if got := ResolveCallerActor("op@example.com"); got != "operator op@example.com" {
		t.Fatalf("actor = %q", got)
	}
	if got := ResolveCallerActor(""); !strings.Contains(got, "no agent signal") {
		t.Fatalf("actor = %q", got)
	}
}
