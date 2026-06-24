package focus

import (
	"os"
	"path/filepath"
	"testing"

	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus"

	"github.com/vrooli/cli-core/cliapp"
)

// TestFocusManifestCoversFocusService asserts that every RPC declared on
// FocusService either has a manifest command binding (in the focus or gaps
// group) or is documented in the manifest's `omitted` array with a reason.
func TestFocusManifestCoversFocusService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, focusv1.File_meta_optimization_manager_v1_focus_focus_proto, "FocusService")
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
