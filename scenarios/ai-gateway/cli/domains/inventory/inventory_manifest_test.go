package inventory

import (
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"

	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestCoversInventoryService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS]
	cliapp.RequireProtoServiceCoverage(t, clitest.ManifestBytes(t), inventoryv1.File_ai_gateway_v1_inventory_inventory_proto, "InventoryService")
}
