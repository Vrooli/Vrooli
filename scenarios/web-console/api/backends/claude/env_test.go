package claude

import "testing"

func TestFilterEnvPreservesClaudeConfigurationAndRemovesShellWrapper(t *testing.T) {
	env := []string{"PATH=/bin", "CLAUDE_CONFIG_DIR=/profiles/work", "CLAUDE_CODE_USE_BEDROCK=1", "BASH_FUNC_claude_code::=() { :;}", "HOME=/tmp"}
	got := FilterEnv(env)
	if len(got) != 4 || got[0] != env[0] || got[1] != env[1] || got[2] != env[2] || got[3] != env[4] {
		t.Fatalf("filtered env = %#v", got)
	}
}
