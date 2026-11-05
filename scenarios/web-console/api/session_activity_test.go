package main

import "testing"

// [REQ:P0-017e] A pane launched by another scenario records no agent type;
// its activity still reads the right harness's screen.
func TestAgentFromLaunchCommand(t *testing.T) {
	cases := map[string]string{
		`cd '/home/u/Vrooli' && CLAUDE_CODE_AGENT_TAG='x' '/home/u/.vrooli/shims/claude' --model 'opus'`: "claude",
		"codex --yolo":                "codex",
		"/usr/local/bin/opencode":     "opencode",
		"grok -m fast":                "grok",
		"bash -lc 'make test'":        "",
		"":                            "",
		"echo claude-is-not-a-binary": "",
	}
	for command, want := range cases {
		if got := agentFromLaunchCommand(command); got != want {
			t.Errorf("agentFromLaunchCommand(%q) = %q, want %q", command, got, want)
		}
	}
}
