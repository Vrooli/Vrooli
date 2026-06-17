package provider

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/testutil"
)

// TestRegisterShape asserts the provider SubcommandGroup matches the
// CLI command surface the user docs and skills depend on. This is a
// drift gate: a missing or renamed verb breaks here before it ships.
func TestRegisterShape(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := Register(app, raw)
	require.NoError(t, err)

	require.Equal(t, "provider", group.Name)
	require.True(t, group.NeedsAPI, "provider commands must declare NeedsAPI=true")

	got := make(map[string]cliapp.Command, len(group.Subcommands))
	for _, c := range group.Subcommands {
		got[c.Name] = c
	}

	want := []string{"list", "start", "stop", "restart", "pull-model", "logs"}
	for _, name := range want {
		_, ok := got[name]
		require.Truef(t, ok, "missing subcommand %q", name)
	}
	require.Equalf(t, len(want), len(group.Subcommands), "unexpected subcommands: %#v", got)

	// Verbs that take a provider-id positional must declare it required.
	for _, name := range []string{"start", "stop", "restart", "logs"} {
		cmd := got[name]
		require.Lenf(t, cmd.Args.Positionals, 1, "%s: expected one positional", name)
		require.Equalf(t, "provider-id", cmd.Args.Positionals[0].Name, "%s: positional name", name)
		require.Truef(t, cmd.Args.Positionals[0].Required, "%s: provider-id must be Required", name)
	}

	pull := got["pull-model"]
	require.Len(t, pull.Args.Positionals, 1)
	require.Equal(t, "model-name", pull.Args.Positionals[0].Name)
	require.True(t, pull.Args.Positionals[0].Required)

	logs := got["logs"]
	flagNames := make([]string, 0, len(logs.Args.Flags))
	for _, f := range logs.Args.Flags {
		flagNames = append(flagNames, f.Name)
	}
	require.ElementsMatch(t, []string{"follow", "tail"}, flagNames, "logs flag surface")
}
