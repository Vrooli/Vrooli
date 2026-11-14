package hosthandlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func TestHyperframesManifestRequiresExplicitRequestAndOutput(t *testing.T) {
	var output bytes.Buffer
	ctx := &rootcli.CommandContext{Stdout: &output, Stderr: &output}
	group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "host", hostBindings(ctx, hostCommandNames))
	if err != nil {
		t.Fatal(err)
	}
	core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli host", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
	err = core.RunWithWriters([]string{"hyperframes", "--request", "/does-not-exist", "--out", "/also-does-not-exist", "--json"}, &output, &output)
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("wrong dispatch error=%v output=%s", err, output.String())
	}
}
