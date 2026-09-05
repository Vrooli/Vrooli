package focus

import (
	"os"
	"path/filepath"
	"testing"

	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

// TestFocusManifestCoversFocusService asserts that every RPC declared on
// FocusService either has a manifest command binding (in the focus or gaps
// group) or is documented in the manifest's `omitted` array with a reason.
func TestFocusManifestCoversFocusService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, focusv1.File_meta_optimization_manager_v1_focus_focus_proto, "FocusService")
}

func TestProjectionLabelsCoverAct(t *testing.T) {
	for _, p := range []sharedv1.Projection{
		sharedv1.Projection_PROJECTION_ANSWER,
		sharedv1.Projection_PROJECTION_VALIDATE,
		sharedv1.Projection_PROJECTION_GUIDE,
		sharedv1.Projection_PROJECTION_ACT,
	} {
		label := projectionLabel(p)
		if label == "cross-cutting" || label == "" {
			t.Fatalf("projection %v rendered default label %q", p, label)
		}
	}
	if got := projectionLabel(sharedv1.Projection_PROJECTION_ACT); got != "act" {
		t.Fatalf("act projection label = %q", got)
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
