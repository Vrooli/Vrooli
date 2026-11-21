package apply

import (
	"os"
	"path/filepath"
	"testing"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply"

	"github.com/vrooli/cli-core/cliapp"
)

// TestApplyManifestCoversApplyService asserts that every RPC declared on
// ApplyService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason. Catches drift between proto and
// CLI: adding a new RPC without binding/omitting it fails here.
func TestApplyManifestCoversApplyService(t *testing.T) {
	manifest := readApplyManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, applyv1.File_brand_manager_v1_apply_apply_proto, "ApplyService")
}

func readApplyManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
