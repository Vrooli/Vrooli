package portability

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
)

func TestManifestCoversPortabilityService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, portabilityv1.File_infrastructure_manager_v1_portability_portability_proto, "PortabilityService")
}
