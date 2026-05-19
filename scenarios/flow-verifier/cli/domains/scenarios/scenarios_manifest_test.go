package scenarios

import (
	"os"
	"path/filepath"
	"testing"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"

	"github.com/vrooli/cli-core/cliapp"
)

func TestScenariosManifestCoversScenariosService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, scenariosv1.File_flow_verifier_v1_scenarios_scenarios_proto, "ScenariosService")
}
