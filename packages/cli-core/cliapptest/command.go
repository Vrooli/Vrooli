package cliapptest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// ReadManifest loads the CLI manifest that owns the package under test.
//
// Domain tests live at <scenario>/cli/domains/<domain>, and the manifest they
// must register against is the scenario's single cli/manifest.json. Walking up
// for it keeps the helper independent of how deep a domain package sits.
func ReadManifest(tb testing.TB) []byte {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("cliapptest: resolve working directory: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "manifest.json")
		if raw, readErr := os.ReadFile(candidate); readErr == nil {
			return raw
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatalf("cliapptest: no manifest.json found in any parent of the test working directory")
			return nil
		}
		dir = parent
	}
}

// FindCommand returns one subcommand by group and name, failing the test with
// the available names when it is absent. Tests previously indexed into the
// group slice by position, which silently bound the wrong command whenever a
// group's order changed.
func FindCommand(tb testing.TB, groups []cliapp.SubcommandGroup, group, name string) cliapp.Command {
	tb.Helper()
	for _, g := range groups {
		if g.Name != group {
			continue
		}
		for _, cmd := range g.Subcommands {
			if cmd.Name == name {
				return cmd
			}
			for _, alias := range cmd.Aliases {
				if alias == name {
					return cmd
				}
			}
		}
		available := make([]string, 0, len(g.Subcommands))
		for _, cmd := range g.Subcommands {
			available = append(available, cmd.Name)
		}
		tb.Fatalf("cliapptest: group %q has no subcommand %q; available: %v", group, name, available)
		return cliapp.Command{}
	}
	available := make([]string, 0, len(groups))
	for _, g := range groups {
		available = append(available, g.Name)
	}
	tb.Fatalf("cliapptest: no subcommand group %q; available: %v", group, available)
	return cliapp.Command{}
}

// RunCommand parses argv through the production parser and invokes the command,
// returning whatever it wrote to stdout.
//
// It deliberately goes through NewTestRunContextFromArgs rather than populating
// flags directly, so a test exercises the same parsing, defaulting and required
// argument enforcement a real invocation gets. The returned error is the
// command's own; a parse failure is returned too, because rejecting bad argv is
// behaviour worth asserting.
func RunCommand(tb testing.TB, cmd cliapp.Command, core *cliapp.ScenarioApp, argv ...string) (string, error) {
	tb.Helper()
	var stdout, stderr bytes.Buffer
	ctx, parseErr := cliapp.NewTestRunContextFromArgs(cmd.Args, argv, core, &stdout, &stderr)
	if parseErr != nil {
		return stdout.String(), parseErr
	}
	var runErr error
	switch {
	case cmd.RunCtx != nil:
		runErr = cmd.RunCtx(ctx)
	case cmd.Run != nil:
		runErr = cmd.Run(argv)
	default:
		return "", fmt.Errorf("cliapptest: command %q has neither Run nor RunCtx", cmd.Name)
	}
	// Read stdout only after the command has run; capturing it earlier returns
	// an empty string no matter what the command wrote.
	return stdout.String(), runErr
}
