package hosthandlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func TestDesktopSessionManifestDispatchValidatesSelectors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing selector", args: []string{"--session-id", "2"}, want: "exactly one of --peer-pid or --display"},
		{name: "both selectors", args: []string{"--session-id", "2", "--peer-pid", "42", "--display", "0"}, want: "exactly one of --peer-pid or --display"},
		{name: "missing session", args: []string{"--display", "0"}, want: "--session-id is required"},
		{name: "zero peer pid rejected", args: []string{"--session-id", "2", "--peer-pid", "0"}, want: "--peer-pid must be a positive integer"},
		{name: "negative peer pid rejected", args: []string{"--session-id", "2", "--peer-pid", "-1"}, want: "--peer-pid must be a positive integer"},
		{name: "display overflow", args: []string{"--session-id", "2", "--display", "65536"}, want: "--display must be an integer"},
		{name: "display malformed", args: []string{"--session-id", "2", "--display", "local"}, want: "--display must be an integer"},
		{name: "remote display rejected", args: []string{"--session-id", "2", "--display", "localhost:0"}, want: "--display must be an integer"},
		{name: "display zero reaches session validation before socket", args: []string{"--session-id", "../2", "--display", "0"}, want: "invalid desktop session identity"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := &rootcli.CommandContext{Stdout: &output, Stderr: &output}
			group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "host", hostBindings(ctx, hostCommandNames))
			if err != nil {
				t.Fatal(err)
			}
			core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli host", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
			args := append([]string{"desktop-session"}, tc.args...)
			err = core.RunWithWriters(args, &output, &output)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q; output=%s", err, tc.want, output.String())
			}
		})
	}
}
