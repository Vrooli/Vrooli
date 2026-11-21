// Package claude holds claude-code-specific helpers that the web-console
// session pipeline reaches for when the user's shell happens to launch
// the claude CLI. Currently: environment filtering so a web-console
// process that was itself spawned from Claude Code doesn't leak its
// session markers into nested claude invocations.
package claude

import "strings"

// FilterEnv removes Claude Code-specific environment variables from the
// given environment slice. Used by PTY factories so that running `claude`
// inside a web-console terminal does not trigger nested-session detection
// when the web-console server was itself started from Claude Code.
//
// Filtered patterns:
//   - CLAUDECODE (nested session detection marker)
//   - CLAUDE_* (session IDs, config paths, internal state)
//   - BASH_FUNC_claude_code::* (exported shell functions)
func FilterEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, v := range env {
		if strings.HasPrefix(v, "CLAUDECODE=") {
			continue
		}
		if strings.HasPrefix(v, "CLAUDE_") {
			continue
		}
		if strings.HasPrefix(v, "BASH_FUNC_claude_code::") {
			continue
		}
		result = append(result, v)
	}
	return result
}
