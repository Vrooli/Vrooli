package runs

import (
	"os"
	"path/filepath"
	"testing"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"

	"github.com/vrooli/cli-core/cliapp"
)

// TestRunsManifestCoversRunsService asserts every RPC on RunsService has either
// a manifest command binding or an `omitted` entry (ReportRunEvent is node-
// facing and omitted) — catching proto↔CLI drift.
func TestRunsManifestCoversRunsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, runsv1.File_vrooli_bridge_v1_runs_runs_proto, "RunsService")
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
