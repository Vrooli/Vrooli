package ladder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	ladderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder"
)

func TestManifestCoversLadderService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, ladderv1.File_infrastructure_manager_v1_ladder_ladder_proto, "LadderService")
}
