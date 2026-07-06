package routing

import (
	"testing"

	clitest "ai-gateway/cli/internal/testutil"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestCoversRoutingService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS]
	cliapp.RequireProtoServiceCoverage(t, clitest.ManifestBytes(t), routingv1.File_ai_gateway_v1_routing_routing_proto, "RoutingService")
}
