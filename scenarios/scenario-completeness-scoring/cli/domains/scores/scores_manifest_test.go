package scores

import (
	"os"
	"path/filepath"
	"testing"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"

	"github.com/vrooli/cli-core/cliapp"
)

// TestScoreManifestCoversScoreService asserts every ScoreService RPC has a
// manifest command binding (or a documented omission). Adding an RPC to
// scoring.proto without a CLI surface fails here.
func TestScoreManifestCoversScoreService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, scoringv1.File_scenario_completeness_scoring_v1_scoring_scoring_proto, "ScoreService")
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
