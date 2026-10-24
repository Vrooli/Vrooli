package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

// ReadManifest loads cli/manifest.json relative to a domain test's working
// directory. Domain tests run with their package dir as CWD
// (cli/domains/<domain>/), so the manifest is two levels up at cli/manifest.json
// — the same relative location the existing *_manifest_test.go files use.
func ReadManifest(tb testing.TB) []byte {
	tb.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(tb, err, "read cli manifest")
	return raw
}

// FindCommand locates a registered command by group + command name. It fails
// the test if the group or command is absent, which keeps the per-verb tests
// honest if a manifest rename ever drops a command on the floor.
func FindCommand(tb testing.TB, groups []cliapp.SubcommandGroup, group, name string) cliapp.Command {
	tb.Helper()
	for _, g := range groups {
		if g.Name != group {
			continue
		}
		for _, c := range g.Subcommands {
			if c.Name == name {
				return c
			}
		}
		tb.Fatalf("command %q not found in group %q", name, group)
	}
	tb.Fatalf("group %q not found in registered groups", group)
	return cliapp.Command{}
}

// RunCommand parses argv against the command's manifest-derived ArgSchema and
// invokes the bound handler, capturing stdout. It drives the production parser
// (NewTestRunContextFromArgs) so a handler that reads a flag the manifest does
// not declare panics — exactly the handler<->manifest drift this suite guards.
//
// Two error sources are folded into the returned error:
//   - a parser error (unknown flag, missing required positional) — handler not reached
//   - the handler's own returned error (API failure, bad input)
//
// On a parser error the captured-output string carries whatever the parser
// wrote; on success it carries the rendered report.
func RunCommand(tb testing.TB, cmd cliapp.Command, app *cliapp.ScenarioApp, argv ...string) (string, error) {
	tb.Helper()
	var out, errOut bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, argv, app, &out, &errOut)
	if err != nil {
		return out.String() + errOut.String(), err
	}
	require.NotNil(tb, cmd.RunCtx, "command %q has no RunCtx handler bound", cmd.Name)
	return outWith(&out, cmd.RunCtx(ctx))
}

func outWith(out *bytes.Buffer, err error) (string, error) { return out.String(), err }
