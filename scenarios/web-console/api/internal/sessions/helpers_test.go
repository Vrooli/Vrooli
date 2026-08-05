package sessions

import (
	"strings"
	"testing"

	"web-console/internal/sessionstore"
)

func TestNormalizeAgentType_AcceptsAllKnownAgents(t *testing.T) {
	cases := map[string]sessionstore.Agent{
		"codex":    sessionstore.AgentCodex,
		"claude":   sessionstore.AgentClaude,
		"opencode": sessionstore.AgentOpenCode,
		"grok":     sessionstore.AgentGrok,
		"none":     sessionstore.AgentNone,
		"bogus":    sessionstore.AgentNone, // unknown rolls forward to none
	}
	for in, want := range cases {
		if got := NormalizeAgentType(in); got != want {
			t.Errorf("NormalizeAgentType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRecoverability_RequiresSessionIDForOpenCodeAndGrok(t *testing.T) {
	for _, agent := range []sessionstore.Agent{sessionstore.AgentOpenCode, sessionstore.AgentGrok} {
		// Missing session id → not recoverable, with a specific reason.
		ok, reason := Recoverability(sessionstore.Metadata{AgentType: agent})
		if ok {
			t.Errorf("%s with no session id should not be recoverable", agent)
		}
		if !strings.Contains(reason, "session id is required") {
			t.Errorf("%s missing-id reason = %q", agent, reason)
		}
		// With a session id → recoverable.
		ok, _ = Recoverability(sessionstore.Metadata{AgentType: agent, AgentSessionID: "abc"})
		if !ok {
			t.Errorf("%s with session id should be recoverable", agent)
		}
	}
}

func TestBuildResumeCommand_OpenCodeAndGrok(t *testing.T) {
	got := BuildResumeCommand(sessionstore.Metadata{AgentType: sessionstore.AgentOpenCode, AgentSessionID: "ses_x"})
	if got != "opencode --session ses_x\n" {
		t.Errorf("opencode resume = %q", got)
	}
	got = BuildResumeCommand(sessionstore.Metadata{AgentType: sessionstore.AgentGrok, AgentSessionID: "grk_y"})
	if got != "grok --resume grk_y\n" {
		t.Errorf("grok resume = %q", got)
	}
	// Missing ids fall back to a safe no-op echo, never an unsafe --last.
	for _, agent := range []sessionstore.Agent{sessionstore.AgentOpenCode, sessionstore.AgentGrok} {
		got = BuildResumeCommand(sessionstore.Metadata{AgentType: agent})
		if !strings.HasPrefix(got, "echo ") {
			t.Errorf("%s with no id should echo, got %q", agent, got)
		}
	}
}
