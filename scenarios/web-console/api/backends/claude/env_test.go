package claude

import "testing"

func TestFilterEnvRemovesNestedClaudeMarkers(t *testing.T) {
	got := FilterEnv([]string{"PATH=/bin", "CLAUDECODE=1", "CLAUDE_SESSION=x", "BASH_FUNC_claude_code::=() { :;}", "HOME=/tmp"})
	if len(got) != 2 || got[0] != "PATH=/bin" || got[1] != "HOME=/tmp" {
		t.Fatalf("filtered env = %#v", got)
	}
}
