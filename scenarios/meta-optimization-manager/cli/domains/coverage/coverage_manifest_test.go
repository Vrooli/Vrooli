package coverage

import (
	"os"
	"path/filepath"
	"testing"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/coverage"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

// TestCoverageManifestCoversCoverageService asserts that every RPC declared on
// CoverageService either has a manifest command binding or is documented in the
// manifest's `omitted` array with a reason. Catches drift between proto and CLI:
// adding a new RPC without binding/omitting it fails here.
func TestCoverageManifestCoversCoverageService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, coveragev1.File_meta_optimization_manager_v1_coverage_coverage_proto, "CoverageService")
}

func TestProjectionLabelsCoverEveryKnownProjection(t *testing.T) {
	labels := map[sharedv1.Projection]string{}
	for _, p := range []sharedv1.Projection{
		sharedv1.Projection_PROJECTION_ANSWER,
		sharedv1.Projection_PROJECTION_VALIDATE,
		sharedv1.Projection_PROJECTION_GUIDE,
		sharedv1.Projection_PROJECTION_ACT,
	} {
		label := projectionLabel(p)
		if label == "unspecified" || label == "" {
			t.Fatalf("projection %v rendered default label %q", p, label)
		}
		if prior, ok := labels[p]; ok && prior != label {
			t.Fatalf("projection %v rendered inconsistently", p)
		}
		labels[p] = label
	}
	if labels[sharedv1.Projection_PROJECTION_ACT] != "act" {
		t.Fatalf("act projection label = %q", labels[sharedv1.Projection_PROJECTION_ACT])
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
