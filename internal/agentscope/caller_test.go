package agentscope

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// Captured 2026-11-07 from a live web-console operator pane (pid 2775) and
// from this host's Claude Code and agent-manager Codex sessions. Values that
// are not markers are elided.
var (
	operatorPaneCgroup = "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/tmux-spawn-a21c3737-0e07-4132-83a7-a22e5fff677b.scope\n"
	operatorPaneEnv    = map[string]string{
		"CODEX_CI": "1", "CODEX_MANAGED_BY_NPM": "1", "CODEX_SESSION_ID": "01a060ba-0d9d-7f01-b81b-c03c0f6822c9",
		"WC_WEB_CONSOLE_SESSION_ID": "fc461e15-b815-463d-99f3-c7653195209c", "WC_SESSION_STATE_ROOT": "/home/op/.vrooli/state/vrooli/web-console/sessions",
		"TMUX": "/home/op/.vrooli/state/vrooli/web-console/sessions/tmux/wc.sock,26193,1608", "SSH_TTY": "/dev/pts/3",
	}
	claudeCodeCgroup = "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-claude-code-946a880f99ff35ffbb4a0f3e77e3b25c.scope\n"
	claudeCodeEnv    = map[string]string{"CLAUDECODE": "1", "AI_AGENT": "claude-code_2-1-316_agent", "VROOLI_AGENT_SESSION": "vrooli-agent-claude-code-946a880f99ff35ffbb4a0f3e77e3b25c", "WC_WEB_CONSOLE_SESSION_ID": "08f53642"}
	codexRunEnv      = map[string]string{"CODEX_CI": "1", "CODEX_THREAD_ID": "019a0000-thread"}
)

// withCallerEnvironment replaces every agent marker the canonical detector
// reads with exactly env.
func withCallerEnvironment(t *testing.T, env map[string]string) {
	t.Helper()
	for _, key := range append(cliutil.AmbientAgentEnvironment(), "AI_AGENT", "VROOLI_AGENT_SESSION", "CODEX_SESSION_ID") {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
	for key, value := range env {
		t.Setenv(key, value)
	}
}

func classify(t *testing.T, env map[string]string, cgroup string) string {
	t.Helper()
	withCallerEnvironment(t, env)
	return SessionEvidence(cliutil.DetectCallerKind(), cgroup)
}

func TestSessionEvidenceAllowsRealOperatorPanesAndRefusesRealAgents(t *testing.T) {
	if got := classify(t, operatorPaneEnv, operatorPaneCgroup); got != "" {
		t.Fatalf("a live web-console operator pane was refused: %s", got)
	}
	if got := classify(t, nil, "0::/user.slice/user-1000.slice/session-3.scope\n"); got != "" {
		t.Fatalf("an ssh login shell was refused: %s", got)
	}
	for name, tc := range map[string]struct {
		env    map[string]string
		cgroup string
	}{
		"claude code session":        {claudeCodeEnv, claudeCodeCgroup},
		"claude code env only":       {claudeCodeEnv, operatorPaneCgroup},
		"agent session scope only":   {nil, claudeCodeCgroup},
		"agent-manager codex run":    {codexRunEnv, operatorPaneCgroup},
		"grok tool shell":            {map[string]string{"GROK_AGENT": "1"}, ""},
		"antigravity tool shell":     {map[string]string{"ANTIGRAVITY_AGENT": "1"}, ""},
		"launcher identity":          {map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "x"}, ""},
		"explicit agent declaration": {map[string]string{"VROOLI_CALLER": "agent"}, ""},
	} {
		if classify(t, tc.env, tc.cgroup) == "" {
			t.Errorf("%s was not refused", name)
		}
	}
}
