package hosthandlers

import (
	"context"
	"io"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func TestWirelessPreflightRejectsMutationAndIncompleteSigningInputs(t *testing.T) {
	for _, tc := range []struct {
		apply       bool
		bundle, dir string
	}{{apply: true, bundle: "reviewed"}, {}, {bundle: "reviewed", dir: "signed"}} {
		run := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema:      cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action"}}, Flags: []cliapp.Flag{{Name: "profile"}, {Name: "bundle"}, {Name: "signed-dir"}, {Name: "cert"}, {Name: "recovery-kernel"}, {Name: "apply", Bool: true}}},
			Positionals: map[string]string{"action": "activation-preflight"}, Flags: map[string]string{"profile": "noble-mt6639", "bundle": tc.bundle, "signed-dir": tc.dir}, BoolFlags: map[string]bool{"apply": tc.apply}, Stdout: io.Discard,
		})
		if err := runHostWireless(run, &rootcli.CommandContext{Context: context.Background()}); err == nil {
			t.Fatal("accepted unsafe preflight request")
		}
	}
}
