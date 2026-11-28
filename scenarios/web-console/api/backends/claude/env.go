// Package claude holds claude-code-specific helpers that the web-console
// session pipeline reaches for when the user's shell happens to launch
// the claude CLI. Currently: environment filtering so a web-console
// process that was itself spawned from Claude Code doesn't leak its
// session markers into nested claude invocations.
package claude

import "strings"

// FilterEnv removes Claude's exported shell wrapper from a terminal
// environment. Parent session identity is filtered centrally by the shared
// foreign-child policy; provider selectors and CLAUDE_CONFIG_DIR are user
// configuration and must survive into an interactive terminal.
func FilterEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, v := range env {
		if strings.HasPrefix(v, "BASH_FUNC_claude_code::") {
			continue
		}
		result = append(result, v)
	}
	return result
}
