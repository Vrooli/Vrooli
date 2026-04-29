package runtime

import (
	"strings"

	"workspace-sandbox/internal/types"
)

// EvaluateProtectedGitAllowlist returns "" when the command should be
// allowed and a non-empty refusal message when the protected-mode
// git allowlist blocks it.
//
// Wraps the protected-sandbox-git-and-network-guardrails contract:
// agents should use Git Control Tower for mutating operations; direct
// `git` is limited to read-only verbs by default.
func EvaluateProtectedGitAllowlist(cfg types.ProtectedConfig, command string, args []string) string {
	if len(cfg.GitAllowlist) == 0 {
		return ""
	}
	for _, v := range cfg.GitAllowlist {
		if v == "*" {
			return ""
		}
	}
	base := command
	if idx := strings.LastIndex(command, "/"); idx >= 0 {
		base = command[idx+1:]
	}
	if base != "git" {
		return ""
	}
	verb := firstArg(args)
	if verb == "" {
		return "git invoked without a verb; allowlist enforces a verb-level policy. Use one of: " + strings.Join(cfg.GitAllowlist, ", ")
	}
	for _, allowed := range cfg.GitAllowlist {
		if allowed == verb {
			return ""
		}
	}
	return "git verb \"" + verb + "\" is not in the protected-mode allowlist (" + strings.Join(cfg.GitAllowlist, ", ") + "). For mutating git operations, route through Git Control Tower instead of direct git invocations."
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
