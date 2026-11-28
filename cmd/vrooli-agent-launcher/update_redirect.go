package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/vrooli/cli-core/cliutil"
)

// updateRedirect names the native update subcommands that cannot work on a
// Vrooli-managed install. Codex's updater runs `npm install -g` against the
// system Node prefix: it needs sudo, and the copy it writes under /usr is
// shadowed by the managed link, so the running version never changes.
// OpenCode's updater replaces the managed binary directly and its `--major`
// path installs a new release line, bypassing the dependency gate and the
// retained rollback prefix. `vrooli resource update <agent>` is the one owner
// of that work.
type updateRedirect struct {
	subcommands map[string]struct{}
	// majorArgs replaces `--major` with the channel selector the managed
	// updater understands. Empty means the agent has no major-upgrade path.
	majorArgs []string
}

var nativeUpdateRedirects = map[string]updateRedirect{
	"codex":    {subcommands: map[string]struct{}{"update": {}}},
	"opencode": {subcommands: map[string]struct{}{"update": {}, "upgrade": {}}, majorArgs: []string{"--channel", "v2"}},
}

// managedUpdateCommand reports the control-plane command that replaces an
// agent's native update invocation. Only the bare subcommand and the declared
// major form are redirected; any other argument shape reaches the agent
// unchanged.
func managedUpdateCommand(runner string, args []string) ([]string, bool) {
	redirect, ok := nativeUpdateRedirects[runner]
	if !ok || len(args) == 0 {
		return nil, false
	}
	if _, ok := redirect.subcommands[args[0]]; !ok {
		return nil, false
	}
	if len(args) == 1 {
		return []string{"vrooli", "resource", "update", runner}, true
	}
	if len(args) == 2 && args[1] == "--major" && len(redirect.majorArgs) > 0 {
		command := []string{"vrooli", "resource", "update", runner}
		return append(command, redirect.majorArgs...), true
	}
	return nil, false
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
