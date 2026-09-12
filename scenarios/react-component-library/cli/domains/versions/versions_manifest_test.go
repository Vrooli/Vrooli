package versions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"

	"github.com/vrooli/cli-core/cliapp"
)

func TestVersionsManifestCoversVersionsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, versionsv1.File_react_component_library_v1_versions_versions_proto, "VersionsService")
}

func TestVersionsManifestExposesMaterializeRecovery(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name    string `json:"name"`
				Binding struct {
					Service string `json:"service"`
					Method  string `json:"method"`
				} `json:"binding"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode cli/manifest.json: %v", err)
	}
	for _, group := range manifest.Groups {
		if group.Name != "versions" {
			continue
		}
		for _, command := range group.Commands {
			if command.Name == "materialize" && command.Binding.Service == "VersionLifecycleService" && command.Binding.Method == "MaterializeVersion" {
				return
			}
		}
	}
	t.Fatal("versions materialize command is not bound to VersionLifecycleService.MaterializeVersion")
}
