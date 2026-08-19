// Command vrooli-agent-launcher is the shared, best-effort attribution
// boundary for native coding-agent binaries. It never interprets agent
// arguments as shell text.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var launchErr *cliutil.AgentLaunchError
		if errors.As(err, &launchErr) {
			fmt.Fprintln(os.Stderr, "vrooli-agent-launcher:", err)
			os.Exit(1)
		}
		if code := cliutil.ChildExitCode(err); code >= 0 {
			os.Exit(code)
		}
		fmt.Fprintln(os.Stderr, "vrooli-agent-launcher:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("vrooli-agent-launcher", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agent := fs.String("agent", "", "coding-agent runner")
	task := fs.String("task", "", "optional task UUID to associate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if strings.TrimSpace(*agent) == "" {
		if len(remaining) == 0 {
			return errors.New("usage: vrooli-agent-launcher --agent <runner> [--task <uuid>] -- [agent args]")
		}
		*agent = remaining[0]
		remaining = remaining[1:]
	}

	return cliutil.LaunchCodingAgent(context.Background(), cliutil.AgentLaunchRequest{
		Agent:  *agent,
		TaskID: *task,
		Args:   remaining,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}
