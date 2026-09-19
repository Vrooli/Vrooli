package rewrite

import (
	"os"
	"path/filepath"
	"testing"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"

	"github.com/vrooli/cli-core/cliapp"
)

// TestManifestCoversGoCodeGraphService asserts coverage of the whole
// GoCodeGraphService from the rewrite-package vantage. See the matching
// test under domains/graph/ for the rationale; either failing surfaces
// drift between proto and CLI.
func TestManifestCoversGoCodeGraphService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, graphv1.File_go_code_graph_v1_graph_graph_proto, "GoCodeGraphService")
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
