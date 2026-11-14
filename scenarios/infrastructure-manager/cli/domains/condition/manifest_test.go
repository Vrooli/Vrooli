package condition

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	conditionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition"
)

func TestManifestCoversConditionService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, conditionv1.File_infrastructure_manager_v1_condition_condition_proto, "ConditionService")
}
