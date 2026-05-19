package flows

import (
	"os"
	"path/filepath"
	"testing"

	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows"

	"github.com/vrooli/cli-core/cliapp"
)

func TestFlowsManifestCoversFlowsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, flowsv1.File_flow_verifier_v1_flows_flows_proto, "FlowsService")
}
