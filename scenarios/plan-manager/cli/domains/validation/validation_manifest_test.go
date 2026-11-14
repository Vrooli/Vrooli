package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"

	"github.com/vrooli/cli-core/cliapp"
)

// TestValidationManifestCoversValidationService asserts that every RPC declared
// on ValidationService has a manifest command binding or is documented in the
// manifest's `omitted` array.
func TestValidationManifestCoversValidationService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, validationv1.File_plan_manager_v1_validation_validation_proto, "ValidationService")
}

func TestDisplacedValidationLifecycleCommandsCannotReturn(t *testing.T) {
	var manifest struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name string `json:"name"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(readManifest(t), &manifest); err != nil {
		t.Fatal(err)
	}
	removed := map[string]bool{"baseline-scope": true, "wait": true, "resume": true, "run": true, "verify-dod": true}
	for _, group := range manifest.Groups {
		if group.Name != "validate" {
			continue
		}
		for _, command := range group.Commands {
			if removed[command.Name] {
				t.Fatalf("displaced caller-owned validation command %q returned", command.Name)
			}
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
