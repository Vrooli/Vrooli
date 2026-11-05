package domains

import (
	"os"
	"path/filepath"
	"testing"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func TestLocalManifestCommandsMatchRegisteredFlatCommands(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	registered := map[string]bool{}
	for _, group := range CommandGroups(&appctx.Context{}) {
		for _, command := range group.Commands {
			registered[command.Name] = true
		}
	}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			if command.Binding.Kind != "local" {
				continue
			}
			path := group.Name
			if !group.Flat {
				path += " " + command.Name
			}
			if !registered[path] {
				t.Errorf("manifest local command %q has no registered dispatch command", path)
			}
		}
	}
}
