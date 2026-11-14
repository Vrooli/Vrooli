package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/vrooli/cli-core/cliutil"
)

// nativeUpdateRedirects names the agents whose own `update` subcommand cannot
// work on a Vrooli-managed install. Codex's updater runs `npm install -g`
// against the system Node prefix: it needs sudo, and the copy it writes under
// /usr is shadowed by the managed link, so the running version never changes.
// `vrooli resource update <agent>` is the one owner of that work.
var nativeUpdateRedirects = map[string]struct{}{
	"codex": {},
}

// managedUpdateCommand reports the control-plane command that replaces an
// agent's native `update` invocation. Only the bare subcommand is redirected;
// any other argument shape reaches the agent unchanged.
func managedUpdateCommand(runner string, args []string) ([]string, bool) {
	if _, ok := nativeUpdateRedirects[runner]; !ok || len(args) != 1 || args[0] != "update" {
		return nil, false
	}
	return []string{"vrooli", "resource", "update", runner}, true
}

// runManagedUpdate runs the control-plane update in this process's terminal
// and returns its exit status.
func runManagedUpdate(argv []string, self string) int {
	path, err := cliutil.ResolveAgentBinaryExcluding(argv[0], self)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vrooli-agent-launcher: cannot update through Vrooli: %v\n", err)
		return 1
	}
	command := exec.Command(path, argv[1:]...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		if code := cliutil.ChildExitCode(err); code >= 0 {
			return code
		}
		fmt.Fprintf(os.Stderr, "vrooli-agent-launcher: %v\n", err)
		return 1
	}
	return 0
}
