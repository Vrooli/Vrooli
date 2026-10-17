package authoring

import (
	"os"
	"path/filepath"
	"testing"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAuthoringManifestCoversAuthoringService asserts that every RPC declared on
// AuthoringService has a manifest command binding or is documented in the
// manifest's `omitted` array.
func TestAuthoringManifestCoversAuthoringService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, authoringv1.File_plan_manager_v1_authoring_authoring_proto, "AuthoringService")
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
