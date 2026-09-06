package hosthandlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	hostapp "github.com/vrooli/vrooli/internal/app/host"
	"github.com/vrooli/vrooli/internal/cli/hostcli"
)

func TestDesktopSessionManifestDispatch(t *testing.T) {
	var output bytes.Buffer
	ctx := &hostcli.Context{Stdout: &output, Stderr: &output}
	group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "host", hostBindings(&hostapp.App{}, ctx, hostCommandNames))
	if err != nil {
		t.Fatal(err)
	}
	core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli host", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
	err = core.RunWithWriters([]string{"desktop-session", "--session-id", "../2", "--peer-pid", "42", "--json"}, &output, &output)
	if err == nil || !strings.Contains(err.Error(), "invalid desktop session identity") {
		t.Fatalf("manifest dispatch error = %v; output = %s", err, output.String())
	}
}
