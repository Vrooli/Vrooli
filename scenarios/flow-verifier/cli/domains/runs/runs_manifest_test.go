package runs

import (
	"os"
	"path/filepath"
	"testing"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRunsManifestCoversRunsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, runsv1.File_flow_verifier_v1_runs_runs_proto, "RunsService")
}
