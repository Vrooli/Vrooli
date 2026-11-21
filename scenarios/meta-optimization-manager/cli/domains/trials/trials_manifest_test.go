package trials

import (
	"os"
	"path/filepath"
	"testing"

	trialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials"

	"github.com/vrooli/cli-core/cliapp"
)

// TestTrialsManifestCoversTrialsService asserts that every RPC declared on
// TrialsService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason.
func TestTrialsManifestCoversTrialsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, trialsv1.File_meta_optimization_manager_v1_trials_trials_proto, "TrialsService")
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
