package permissionscli

import (
	"os"

	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/permissions"

	"github.com/vrooli/cli-core/cliapp"
)

// guardCommand exposes one PreToolUse decision as a process boundary so Claude
// Code can invoke it directly as a hook. Arguments are the Bash deny patterns
// the hook was installed with; the tool event arrives on stdin as JSON.
//
// The decision is communicated through the exit code and, for an ask, a JSON
// permission decision on stdout, which is Claude's hook contract, so this
// command exits the process rather than returning an error.
func (h *Handlers) guardCommand() cliapp.Command {
	return cliapp.Command{
		Name:        "hook-guard",
		Description: "Decide one PreToolUse Bash event through the shared agent policy runtime (hook entrypoint)",
		Usage:       "resource-claude-code permissions hook-guard [deny-pattern]...",
		Run: func(args []string) error {
			stdin, stdout, stderr := h.Stdin, h.Stdout, h.Stderr
			if stdin == nil {
				stdin = os.Stdin
			}
			if stdout == nil {
				stdout = os.Stdout
			}
			if stderr == nil {
				stderr = os.Stderr
			}
			os.Exit(permissions.RunHookGuard(stdin, stdout, stderr, args, permissions.LoadGuardEnv()))
			return nil
		},
	}
}
