package permissionscli

import (
	"os"

	"resource-claude-code/cli/internal/permissions"

	"github.com/vrooli/cli-core/cliapp"
)

// guardCommand exposes one PreToolUse decision as a process boundary so Claude
// Code can invoke it directly as a hook. Arguments are the Bash deny patterns
// the hook was installed with; the tool event arrives on stdin as JSON.
//
// The decision is communicated through the exit code, which is Claude's hook
// contract, so this command exits the process rather than returning an error.
func (h *Handlers) guardCommand() cliapp.Command {
	return cliapp.Command{
		Name:        "hook-guard",
		Description: "Evaluate one PreToolUse Bash event against the managed deny rules (hook entrypoint)",
		Usage:       "resource-claude-code permissions hook-guard <deny-pattern>...",
		Run: func(args []string) error {
			stdin, stderr := h.Stdin, h.Stderr
			if stdin == nil {
				stdin = os.Stdin
			}
			if stderr == nil {
				stderr = os.Stderr
			}
			os.Exit(permissions.RunHookGuard(stdin, stderr, args, permissions.LoadGuardEnv()))
			return nil
		},
	}
}
