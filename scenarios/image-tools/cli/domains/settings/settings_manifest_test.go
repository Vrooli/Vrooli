package settings

import (
	"os"
	"path/filepath"
	"testing"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"

	"github.com/vrooli/cli-core/cliapp"
)

// TestSettingsManifestCoversModelsService asserts every RPC on ModelsService has
// a manifest command binding (or is documented as omitted) — the same contract
// the models domain enforces. The settings group binds the default-model RPCs
// (ListDefaults, SetDefaultModel), so a new ModelsService RPC left unbound fails
// here as well as in the models domain.
func TestSettingsManifestCoversModelsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, modelsv1.File_image_tools_v1_models_models_proto, "ModelsService")
}

// TestSettingsGroupRegisters confirms the settings group loads from the manifest
// and exposes the expected command surface (manifest-bound list/set-default plus
// the hand-appended clear-default).
func TestSettingsGroupRegisters(t *testing.T) {
	manifest := readManifest(t)
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	want := map[string]bool{"list": false, "set-default": false, "clear-default": false}
	for _, sub := range group.Subcommands {
		if _, ok := want[sub.Name]; ok {
			want[sub.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("settings group missing command %q", name)
		}
	}
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
