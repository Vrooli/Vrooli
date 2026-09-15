package graph

import (
	"os"
	"path/filepath"
	"testing"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"

	"github.com/vrooli/cli-core/cliapp"
)

// TestManifestCoversGoCodeGraphService asserts that every RPC declared on
// GoCodeGraphService either has a manifest command binding or is documented
// in the manifest's `omitted` array with a reason. Catches drift between
// proto and CLI: adding a new RPC without binding/omitting it fails here.
//
// All three RPCs (Extract, RewritePlan, RewriteApply) live on a single
// proto service even though the CLI splits them across the `graph` and
// `rewrite` groups; this test asserts coverage of the whole service.
// The matching test under domains/rewrite/ asserts the same contract from
// the rewrite-package vantage; either failing surfaces drift.
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
