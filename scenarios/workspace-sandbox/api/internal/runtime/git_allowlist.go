package runtime

import (
	"strings"

	"workspace-sandbox/internal/types"
)

// GitDenyMessages bundles the two rejection-message templates rendered by
// EvaluateProtectedGitAllowlist. Empty fields fall back to the hardcoded
// strong defaults (see defaultBlockedTemplate / defaultNoVerbTemplate).
//
// Supported placeholders:
//   - Blocked: {verb}, {allowlist}
//   - NoVerb:  {allowlist}
type GitDenyMessages struct {
	// Blocked is the template for the "verb on the disallow list" case.
	Blocked string
	// NoVerb is the template for the "bare git, no verb" case.
	NoVerb string
}

// EvaluateProtectedGitAllowlist returns "" when the command should be
// allowed and a non-empty refusal message when the protected-mode
// git allowlist blocks it.
//
// Wraps the protected-sandbox-git-and-network-guardrails contract:
// agents should use Git Control Tower for mutating operations; direct
// `git` is limited to read-only verbs by default.
//
// The `msgs` parameter lets the caller override the rejection-message
// templates (e.g. from operator-provided `.vrooli/config.json` behavior).
// Empty fields use the hardcoded strong defaults so callers that don't
// care about templating can pass GitDenyMessages{}.
func EvaluateProtectedGitAllowlist(cfg types.ProtectedConfig, msgs GitDenyMessages, command string, args []string) string {
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
	allowlist := strings.Join(cfg.GitAllowlist, ", ")
	if verb == "" {
		return renderMessage(msgs.NoVerb, defaultNoVerbTemplate, "", allowlist)
	}
	for _, allowed := range cfg.GitAllowlist {
		if allowed == verb {
			return ""
		}
	}
	return renderMessage(msgs.Blocked, defaultBlockedTemplate, verb, allowlist)
}

const defaultNoVerbTemplate = "git invoked without a verb in a sandboxed run. Before retrying with a verb, note: git operations are almost always a human's responsibility, not an agent's. Direct git is restricted to read-only verbs in this sandbox ({allowlist}); anything mutating belongs to the human."

const defaultBlockedTemplate = "git verb \"{verb}\" is blocked in this sandboxed run.\n\n" +
	"Git operations are almost always a human's responsibility, not an agent's. Even verbs that look safe — commit, branch, stash, checkout, reset, rebase, merge, pull, push, clean — should not be run by an agent unless the user has explicitly authorized git work for this specific task. In the overwhelming majority of cases the correct action here is to STOP and leave the repository alone; let the human decide when and how to commit, reset, rebase, or clean up.\n\n" +
	"If you find yourself reaching for git to \"save progress\", \"clean up\", \"undo a mistake\", \"sync with main\", or \"check what changed\" beyond a read-only diff, you are very likely doing something the user did not ask for. Reconsider what you were about to do — the human will handle the git side.\n\n" +
	"If, and only if, you have explicit authorization from the user for a git operation as part of this task, route it through the git-control-tower CLI (not direct git). Direct git is restricted to read-only verbs ({allowlist}) in this sandbox, and you must not attempt to bypass this restriction by aliasing, scripting around, or invoking git through another tool."

// renderMessage substitutes {verb} and {allowlist} placeholders. Falls back to
// the hardcoded default when the operator-provided template is empty.
func renderMessage(template, fallback, verb, allowlist string) string {
	if template == "" {
		template = fallback
	}
	out := template
	if strings.Contains(out, "{verb}") {
		out = strings.ReplaceAll(out, "{verb}", verb)
	}
	if strings.Contains(out, "{allowlist}") {
		out = strings.ReplaceAll(out, "{allowlist}", allowlist)
	}
	return out
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
